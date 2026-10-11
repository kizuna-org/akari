// Package thought runs independent thinking series without owning shared mind state.
package thought

import (
	"context"
	"fmt"
	"sync"

	"github.com/kizuna-org/akari/internal/mind"
)

type fault string

func (err fault) Error() string { return string(err) }

const (
	ErrConfig   fault = "workspace and positive capacity are required"
	ErrRunner   fault = "channel identity and runner are required"
	ErrCapacity fault = "channel infrastructure capacity reached"
	ErrActive   fault = "channel still owns running or unresolved work"
	ErrToken    fault = "completion token is not delivered or no longer current"
	ErrPanic    fault = "channel panicked"
)

type Runner func(context.Context, mind.Snapshot) (mind.Proposal, error)

// Workspace is consumed here; durable and in-memory owners provide the same thought context.
type Workspace interface {
	Snapshot() mind.Snapshot
	CommitContext(ctx context.Context, proposal mind.Proposal) (uint64, error)
}

type Adopter func(context.Context, mind.Proposal) (uint64, error)

type Token struct {
	Channel    string
	Generation uint64
}

type Result struct {
	Token    Token
	Proposal mind.Proposal
	Err      error
}

// The task owns its cancellation lifetime across Start, Cancel and completion.
type task struct { //nolint:containedctx // Lifecycle ownership, not request context propagation.
	ctx       context.Context
	cancel    context.CancelFunc
	token     Token
	proposal  mind.Proposal
	err       error
	delivered bool
	adopting  bool
}

// Supervisor bounds running AND unresolved work so completed results cannot grow without limit.
type Supervisor struct {
	mu         sync.Mutex
	ctx        context.Context //nolint:containedctx // The supervisor owns the worker lifetime.
	cancel     context.CancelFunc
	workspace  Workspace
	capacity   int
	generation uint64
	tasks      map[string]*task
	ready      chan Token
	workers    sync.WaitGroup
	waitOnce   sync.Once
	done       chan struct{}
}

func New(ctx context.Context, workspace Workspace, capacity int) (*Supervisor, error) {
	if workspace == nil || capacity < 1 {
		return nil, ErrConfig
	}

	root, cancel := context.WithCancel(ctx)

	return &Supervisor{
		mu: sync.Mutex{}, ctx: root, cancel: cancel, workspace: workspace, capacity: capacity,
		generation: 0, tasks: make(map[string]*task), ready: make(chan Token, capacity), workers: sync.WaitGroup{},
		waitOnce: sync.Once{}, done: make(chan struct{}),
	}, nil
}

// Start creates a new generation. Resource capacity is not persona's consciousness width.
func (supervisor *Supervisor) Start(identity string, runner Runner) (Token, error) {
	supervisor.mu.Lock()
	defer supervisor.mu.Unlock()

	err := supervisor.canStart(identity, runner)
	if err != nil {
		return Token{}, err
	}

	supervisor.generation++
	ctx, cancel := context.WithCancel(supervisor.ctx)
	token := Token{Channel: identity, Generation: supervisor.generation}
	work := &task{
		ctx: ctx, cancel: cancel, token: token,
		proposal: mind.Proposal{ID: "", Reads: nil, Writes: nil}, err: nil, delivered: false, adopting: false,
	}
	supervisor.tasks[identity] = work
	snapshot := supervisor.workspace.Snapshot()
	supervisor.workers.Add(1)

	go supervisor.run(work, snapshot, runner)

	return token, nil
}

// Named results let recovery return a failure instead of terminating the process.
//
//nolint:nonamedreturns // Panic recovery writes the returned error.
func evaluate(ctx context.Context, snapshot mind.Snapshot, runner Runner) (proposal mind.Proposal, err error) {
	defer func() {
		if recover() != nil {
			err = ErrPanic
		}
	}()

	return runner(ctx, snapshot)
}

// Next returns a frozen proposal; Accept uses the private stored copy, not this preview.
func (supervisor *Supervisor) Next(ctx context.Context) (Result, error) {
	contextErr := ctx.Err()
	if contextErr != nil {
		return Result{}, contextErr
	}

	rootErr := supervisor.ctx.Err()
	if rootErr != nil {
		return Result{}, rootErr
	}

	select {
	case token := <-supervisor.ready:
		supervisor.mu.Lock()
		defer supervisor.mu.Unlock()

		work := supervisor.tasks[token.Channel]
		work.delivered = true

		return Result{Token: token, Proposal: work.proposal.Clone(), Err: work.err}, nil
	case <-supervisor.ctx.Done():
		return Result{}, supervisor.ctx.Err()
	case <-ctx.Done():
		return Result{}, ctx.Err()
	}
}

// Accept commits only a current, delivered, uncanceled generation, and releases its slot.
func (supervisor *Supervisor) Accept(token Token) (uint64, error) {
	return supervisor.AcceptWith(token, supervisor.workspace.CommitContext)
}

// AcceptWith lets trusted runtime atomically couple the private proposal and planned actions.
// Disk I/O does not hold the supervisor lock; cancellation and other Channels can proceed.
func (supervisor *Supervisor) AcceptWith(token Token, adopt Adopter) (uint64, error) {
	supervisor.mu.Lock()

	work, err := supervisor.delivered(token)
	if err != nil || adopt == nil {
		supervisor.mu.Unlock()

		return 0, ErrToken
	}

	err = work.ctx.Err()
	if err == nil {
		err = work.err
	}

	if err != nil {
		supervisor.release(work)
		supervisor.mu.Unlock()

		return 0, err
	}

	work.adopting = true

	supervisor.workers.Add(1)
	supervisor.mu.Unlock()

	defer func() {
		supervisor.mu.Lock()
		supervisor.release(work)
		supervisor.mu.Unlock()
		supervisor.workers.Done()
	}()

	return adopt(work.ctx, work.proposal.Clone())
}

func (supervisor *Supervisor) Discard(token Token) error {
	supervisor.mu.Lock()
	defer supervisor.mu.Unlock()

	work, err := supervisor.delivered(token)
	if err != nil {
		return err
	}

	supervisor.release(work)

	return nil
}

// Cancel does not free a slot until the old worker is finished and its result resolved.
func (supervisor *Supervisor) Cancel(identity string) {
	supervisor.mu.Lock()
	defer supervisor.mu.Unlock()

	if work, exists := supervisor.tasks[identity]; exists {
		work.cancel()
	}
}

// Close cancels all workers and waits only up to the caller's deadline.
func (supervisor *Supervisor) Close(ctx context.Context) error {
	supervisor.mu.Lock()
	supervisor.cancel()
	supervisor.mu.Unlock()

	supervisor.waitOnce.Do(func() {
		go func() {
			supervisor.workers.Wait()
			close(supervisor.done)
		}()
	})

	select {
	case <-supervisor.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (supervisor *Supervisor) canStart(identity string, runner Runner) error {
	contextErr := supervisor.ctx.Err()
	if contextErr != nil {
		return contextErr
	}

	if identity == "" || runner == nil {
		return ErrRunner
	}

	if _, exists := supervisor.tasks[identity]; exists {
		return ErrActive
	}

	if len(supervisor.tasks) >= supervisor.capacity {
		return ErrCapacity
	}

	return nil
}

func (supervisor *Supervisor) run(work *task, snapshot mind.Snapshot, runner Runner) {
	defer supervisor.workers.Done()

	proposal, err := evaluate(work.ctx, snapshot, runner)
	proposal.ID = fmt.Sprintf("%s/%d", work.token.Channel, work.token.Generation)

	supervisor.mu.Lock()
	work.proposal = proposal.Clone()
	work.err = err
	supervisor.mu.Unlock()

	select {
	case supervisor.ready <- work.token:
	case <-supervisor.ctx.Done():
	}
}

func (supervisor *Supervisor) delivered(token Token) (*task, error) {
	work, exists := supervisor.tasks[token.Channel]
	if !exists || work.token != token || !work.delivered || work.adopting {
		return nil, ErrToken
	}

	return work, nil
}

func (supervisor *Supervisor) release(work *task) {
	work.cancel()
	delete(supervisor.tasks, work.token.Channel)
}
