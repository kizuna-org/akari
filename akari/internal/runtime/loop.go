// Package runtime connects a conversation input boundary to thought adoption and response delivery.
//
//nolint:wsl,wsl_v5,cyclop,noinlineerr // Keep lifecycle transitions and their returned errors adjacent.
package runtime

import (
	"context"
	"errors"
	"sync"

	"github.com/kizuna-org/akari/internal/continuity"
	"github.com/kizuna-org/akari/internal/lifecycle"
	"github.com/kizuna-org/akari/internal/mind"
	"github.com/kizuna-org/akari/internal/thought"
)

const (
	ErrConfig   fault = "runtime: session, runner, and response sink are required"
	ErrInput    fault = "runtime: input ID is required"
	ErrResponse fault = "runtime: thought proposal has no response"
	errPaused   fault = "runtime: interaction canceled by pause"
)

type fault string

func (err fault) Error() string { return string(err) }

// Input is a medium-neutral conversation event.
type Input struct {
	ID      string
	Content string
}

// RunnerFactory builds a thought runner for one input.
type RunnerFactory func(Input) thought.Runner

// Sink delivers only an adopted response.
type Sink func(context.Context, Input, string) error

const (
	inputKey    = "interaction/input"
	responseKey = "interaction/response"
)

// Loop owns one always-on interaction Channel and serializes its inputs.
type Loop struct {
	session           *lifecycle.Session
	runner            RunnerFactory
	sink              Sink
	mu                sync.Mutex
	paused            bool
	pauseSeq          uint64
	closed            bool
	changed           chan struct{}
	closedCh          chan struct{}
	gate              chan struct{}
	cleanup           <-chan struct{}
	cancelInteraction func()
}

func New(session *lifecycle.Session, runner RunnerFactory, sink Sink) (*Loop, error) {
	if session == nil || runner == nil || sink == nil {
		return nil, ErrConfig
	}
	gate := make(chan struct{}, 1)
	gate <- struct{}{}

	return &Loop{session: session, runner: runner, sink: sink, mu: sync.Mutex{}, paused: false, pauseSeq: 0, closed: false,
		changed: make(chan struct{}), closedCh: make(chan struct{}), gate: gate, cleanup: nil,
		cancelInteraction: func() { session.Cancel("interaction") }}, nil
}

// Pause stops admitting new inputs and cancels the current thought. Resume reopens admission.
func (loop *Loop) Pause() {
	loop.mu.Lock()
	if !loop.closed {
		loop.paused = true
		loop.pauseSeq++
		loop.signalLocked()
		loop.cancelInteraction()
	}
	loop.mu.Unlock()
}

func (loop *Loop) Resume() {
	loop.mu.Lock()
	if !loop.closed {
		loop.paused = false
		loop.signalLocked()
	}
	loop.mu.Unlock()
}

// Close prevents new work and cancels the current thought. Session.Close owns final shutdown.
func (loop *Loop) Close() {
	loop.mu.Lock()
	if !loop.closed {
		loop.closed = true
		close(loop.closedCh)
		loop.signalLocked()
		loop.cancelInteraction()
	}
	loop.mu.Unlock()
}

// Run consumes inputs until the source closes, the caller cancels, or Close is called.
// It returns the first error from handling an input so a host supervisor can restart the loop.
func (loop *Loop) Run(ctx context.Context, inputs <-chan Input) error {
	for {
		if err := loop.awaitOpen(ctx); err != nil {
			if errors.Is(err, lifecycle.ErrClosed) {
				return nil
			}

			return err
		}
		changed, closed, paused := loop.runState()
		if closed {
			return nil
		}
		if paused {
			continue
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-changed:
			continue
		case input, open := <-inputs:
			if !open {
				return nil
			}
			if err := loop.Handle(ctx, input); err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				if errors.Is(err, lifecycle.ErrClosed) {
					return nil
				}
				if errors.Is(err, errPaused) {
					continue
				}

				return err
			}
		}
	}
}

// Handle runs one input through adoption, thought, response adoption, and delivery.
//
//nolint:funlen // One serialized input transaction keeps the admission and cleanup boundary explicit.
func (loop *Loop) Handle(ctx context.Context, input Input) error {
	if input.ID == "" {
		return ErrInput
	}
	if err := loop.awaitOpen(ctx); err != nil {
		return err
	}
	if err := loop.acquire(ctx); err != nil {
		return err
	}
	defer loop.release()
	if err := loop.awaitCleanup(ctx); err != nil {
		return err
	}
	if err := loop.awaitOpen(ctx); err != nil {
		return err
	}
	loop.mu.Lock()
	admissionPauseSeq := loop.pauseSeq
	loop.mu.Unlock()

	snapshot := loop.session.Snapshot()
	base := snapshot.Items[inputKey].Version
	if _, err := loop.session.Adopt(ctx, mind.Proposal{ID: "interaction/input", Reads: map[string]uint64{inputKey: base},
		Writes: map[string]string{inputKey: input.Content}}, nil); err != nil {
		return err
	}
	runner := loop.runner(input)
	loop.mu.Lock()
	if loop.closed {
		loop.mu.Unlock()

		return lifecycle.ErrClosed
	}
	if loop.paused || loop.pauseSeq != admissionPauseSeq {
		loop.mu.Unlock()

		return errors.Join(context.Canceled, errPaused)
	}
	pauseSeq := loop.pauseSeq
	token, err := loop.session.Start("interaction", runner)
	loop.mu.Unlock()
	if err != nil {
		return err
	}

	result, err := loop.wait(ctx, token)
	if err != nil {
		loop.session.Cancel("interaction")
		if ctx.Err() != nil || errors.Is(err, context.Canceled) {
			loop.reapCanceled(ctx, token)
		}

		return err
	}
	defer func() { _ = loop.session.Discard(context.WithoutCancel(ctx), token) }()
	if result.Err != nil {
		return loop.resolveInterruption(ctx, pauseSeq, result.Err)
	}
	response := result.Proposal.Writes[responseKey]
	if response == "" {
		return ErrResponse
	}
	if _, err = loop.session.Accept(ctx, token, []continuity.Intent{}); err != nil {
		return loop.resolveInterruption(ctx, pauseSeq, err)
	}

	return loop.session.Operate(ctx, func(deliveryCtx context.Context) error {
		return loop.sink(deliveryCtx, input, response)
	})
}

// wait observes Close even when the runner ignores the cancellation it receives.
func (loop *Loop) wait(ctx context.Context, token thought.Token) (thought.Result, error) {
	type outcome struct {
		result thought.Result
		err    error
	}
	waitCtx, cancel := context.WithCancel(ctx)
	done := make(chan outcome, 1)
	go func() {
		result, err := loop.session.Wait(waitCtx, token)
		done <- outcome{result: result, err: err}
	}()
	loop.mu.Lock()
	closedCh := loop.closedCh
	loop.mu.Unlock()
	select {
	case completed := <-done:
		cancel()

		return completed.result, completed.err
	case <-ctx.Done():
		cancel()
		completed := <-done

		return completed.result, completed.err
	case <-closedCh:
		cancel()
		completed := <-done
		if completed.err == nil {
			_ = loop.session.Discard(context.WithoutCancel(ctx), token)
		} else {
			loop.reapCanceled(ctx, token)
		}

		return thought.Result{}, lifecycle.ErrClosed
	}
}

// runState captures the notification channel with its lifecycle state so Run
// cannot miss a Close or Pause between awaitOpen and its select.
func (loop *Loop) runState() (<-chan struct{}, bool, bool) {
	loop.mu.Lock()
	defer loop.mu.Unlock()

	return loop.changed, loop.closed, loop.paused
}

func (loop *Loop) acquire(ctx context.Context) error {
	for {
		loop.mu.Lock()
		changed := loop.changed
		closed := loop.closed
		loop.mu.Unlock()
		if closed {
			return lifecycle.ErrClosed
		}

		select {
		case <-loop.gate:
			return nil
		case <-changed:
			if err := loop.awaitOpen(ctx); err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (loop *Loop) release() { loop.gate <- struct{}{} }

func (loop *Loop) awaitCleanup(ctx context.Context) error {
	loop.mu.Lock()
	cleanup := loop.cleanup
	changed := loop.changed
	closed := loop.closed
	loop.mu.Unlock()
	if closed {
		return lifecycle.ErrClosed
	}
	if cleanup == nil {
		return nil
	}
	select {
	case <-cleanup:
		loop.mu.Lock()
		if loop.cleanup == cleanup {
			loop.cleanup = nil
		}
		loop.mu.Unlock()

		return nil
	case <-changed:
		if err := loop.awaitOpen(ctx); err != nil {
			return err
		}

		return loop.awaitCleanup(ctx)
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (loop *Loop) reapCanceled(ctx context.Context, token thought.Token) {
	done := make(chan struct{})
	loop.mu.Lock()
	loop.cleanup = done
	loop.mu.Unlock()

	go func() {
		defer close(done)
		cleanupCtx := context.WithoutCancel(ctx)
		result, err := loop.session.Wait(cleanupCtx, token)
		if err == nil {
			_ = loop.session.Discard(cleanupCtx, result.Token)
		}
	}()
}

func (loop *Loop) awaitOpen(ctx context.Context) error {
	for {
		loop.mu.Lock()
		if loop.closed {
			loop.mu.Unlock()

			return lifecycle.ErrClosed
		}
		if !loop.paused {
			loop.mu.Unlock()
			if err := ctx.Err(); err != nil {
				return err
			}

			return nil
		}
		changed := loop.changed
		loop.mu.Unlock()
		select {
		case <-changed:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (loop *Loop) transitionsSince(pauseSeq uint64) (bool, bool) {
	loop.mu.Lock()
	defer loop.mu.Unlock()

	return loop.closed, loop.pauseSeq != pauseSeq
}

func (loop *Loop) resolveInterruption(ctx context.Context, pauseSeq uint64, err error) error {
	if errors.Is(err, continuity.ErrUnavailable) {
		return err
	}
	if !errors.Is(err, context.Canceled) || ctx.Err() != nil {
		return err
	}

	closed, paused := loop.transitionsSince(pauseSeq)
	if closed {
		return lifecycle.ErrClosed
	}
	if paused {
		return errors.Join(context.Canceled, errPaused)
	}

	return err
}

func (loop *Loop) signalLocked() {
	close(loop.changed)
	loop.changed = make(chan struct{})
}
