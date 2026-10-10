// Package lifecycle owns process startup, admission, cancellation and final durable state.
package lifecycle

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/kizuna-org/akari/internal/action"
	"github.com/kizuna-org/akari/internal/continuity"
	"github.com/kizuna-org/akari/internal/memory"
	"github.com/kizuna-org/akari/internal/mind"
	"github.com/kizuna-org/akari/internal/thought"
)

type fault string

func (err fault) Error() string { return string(err) }

const (
	ErrConfig fault = "lifecycle requires positive thought and shutdown limits"
	ErrClosed fault = "process session is stopping or stopped"
)

type Repository interface {
	continuity.Repository
	Acquire(ctx context.Context) (func(), error)
}

type Config struct {
	State           continuity.Config
	ThoughtCapacity int
	DrainTimeout    time.Duration
	FinalizeTimeout time.Duration
}

type Session struct {
	mu         sync.Mutex
	ctx        context.Context //nolint:containedctx // Owns this process's worker lifetime.
	cancel     context.CancelFunc
	owner      *continuity.Owner
	supervisor *thought.Supervisor
	config     Config
	closing    bool
	workers    sync.WaitGroup
	waitOnce   sync.Once
	workDone   chan struct{}
	closeOnce  sync.Once
	closeErr   error
	release    func()
	done       chan struct{}
}

// Open restores under an exclusive process lease before admitting any work.
func Open(ctx context.Context, repository Repository, gateway *action.Gateway, config Config) (*Session, error) {
	if repository == nil || config.ThoughtCapacity < 1 || config.DrainTimeout <= 0 || config.FinalizeTimeout <= 0 {
		return nil, ErrConfig
	}

	release, err := repository.Acquire(ctx)
	if err != nil {
		return nil, err
	}

	owner, err := continuity.Open(ctx, repository, gateway, config.State)
	if err != nil {
		release()

		return nil, err
	}

	root, cancel := context.WithCancel(ctx)

	supervisor, err := thought.New(root, owner, config.ThoughtCapacity)
	if err != nil {
		cancel()
		release()

		return nil, err
	}

	return &Session{
		mu: sync.Mutex{}, ctx: root, cancel: cancel, owner: owner, supervisor: supervisor,
		config: config, closing: false, workers: sync.WaitGroup{}, waitOnce: sync.Once{}, workDone: make(chan struct{}),
		closeOnce: sync.Once{}, closeErr: nil, release: release, done: make(chan struct{}),
	}, nil
}

func (session *Session) Snapshot() mind.Snapshot { return session.owner.Snapshot() }

func (session *Session) Recall(ctx context.Context, query memory.Query) ([]memory.Fragment, error) {
	return session.owner.Recall(ctx, query)
}

// Export is for management/tests only; it includes persona configuration and execution records.
func (session *Session) Export() continuity.Frame { return session.owner.Export() }

func (session *Session) Start(identity string, runner thought.Runner) (thought.Token, error) {
	session.mu.Lock()
	defer session.mu.Unlock()

	if session.closing || session.ctx.Err() != nil {
		return thought.Token{}, ErrClosed
	}

	return session.supervisor.Start(identity, runner)
}

func (session *Session) Next(ctx context.Context) (thought.Result, error) {
	return session.supervisor.Next(ctx)
}

func (session *Session) Cancel(identity string) { session.supervisor.Cancel(identity) }

func (session *Session) Accept(ctx context.Context, token thought.Token, intents []continuity.Intent) (uint64, error) {
	operation, end, err := session.begin(ctx)
	if err != nil {
		return 0, err
	}
	defer end()

	return session.supervisor.AcceptWith(token, func(work context.Context, proposal mind.Proposal) (uint64, error) {
		adoption, cancel := context.WithCancel(operation)

		stop := context.AfterFunc(work, cancel)
		defer stop()
		defer cancel()

		return session.owner.Adopt(adoption, proposal, intents)
	})
}

func (session *Session) Adopt(
	ctx context.Context, proposal mind.Proposal, intents []continuity.Intent,
) (uint64, error) {
	operation, end, err := session.begin(ctx)
	if err != nil {
		return 0, err
	}
	defer end()

	return session.owner.Adopt(operation, proposal, intents)
}

func (session *Session) Dispatch(ctx context.Context, identity string) (action.Observation, error) {
	operation, end, err := session.begin(ctx)
	if err != nil {
		return action.Observation{Status: action.NotExecuted, Data: nil}, err
	}
	defer end()

	return session.owner.Dispatch(operation, identity)
}

// Approve is an administrative runtime capability, never supplied to a thinking Channel.
func (session *Session) Approve(ctx context.Context, identity string) error {
	operation, end, err := session.begin(ctx)
	if err != nil {
		return err
	}
	defer end()

	return session.owner.Approve(operation, identity)
}

func (session *Session) CancelAction(ctx context.Context, identity string) error {
	operation, end, err := session.begin(ctx)
	if err != nil {
		return err
	}
	defer end()

	return session.owner.Cancel(operation, identity)
}

// Close stops admission, cancels workers, drains, then seals state before releasing the process lease.
// Finalization has its own bounded lifetime even when the drain deadline expires.
func (session *Session) Close(ctx context.Context) error {
	session.closeOnce.Do(func() { session.closeErr = session.close(ctx) })

	return session.closeErr
}

// Done closes only after all workers finish and the installation lease is released.
func (session *Session) Done() <-chan struct{} { return session.done }

func (session *Session) begin(ctx context.Context) (context.Context, func(), error) {
	session.mu.Lock()
	defer session.mu.Unlock()

	if session.closing || session.ctx.Err() != nil {
		return nil, nil, ErrClosed
	}

	err := ctx.Err()
	if err != nil {
		return nil, nil, err
	}

	session.workers.Add(1)

	operation, cancel := context.WithCancel(ctx)
	stop := context.AfterFunc(session.ctx, cancel)

	return operation, func() {
		stop()
		cancel()
		session.workers.Done()
	}, nil
}

func (session *Session) close(ctx context.Context) error {
	session.mu.Lock()
	session.closing = true
	session.cancel()
	session.mu.Unlock()

	drain, cancel := context.WithTimeout(ctx, session.config.DrainTimeout)
	defer cancel()

	thoughtErr := session.supervisor.Close(drain)
	session.waitOnce.Do(func() {
		go func() {
			session.workers.Wait()
			close(session.workDone)
		}()
	})

	workErr := session.wait(drain)

	finalize, finish := context.WithTimeout(context.WithoutCancel(ctx), session.config.FinalizeTimeout)
	defer finish()

	saveErr := session.owner.Seal(finalize)

	if thoughtErr == nil && workErr == nil {
		session.release()
		close(session.done)
	} else {
		// Keep the lease while noncooperative workers can still affect the world.
		go session.releaseWhenDrained(context.WithoutCancel(ctx))
	}

	return errors.Join(thoughtErr, workErr, saveErr)
}

func (session *Session) wait(ctx context.Context) error {
	select {
	case <-session.workDone:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (session *Session) releaseWhenDrained(ctx context.Context) {
	_ = session.supervisor.Close(ctx)
	<-session.workDone
	session.release()
	close(session.done)
}
