// Package runtime connects a conversation input boundary to thought adoption and response delivery.
//
//nolint:wsl,wsl_v5 // Keep lifecycle state transitions grouped by operation.
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
	ErrConfig   fault = "runtime requires a session, runner and response sink"
	ErrInput    fault = "input ID is required"
	ErrResponse fault = "thought proposal has no response"
)

type fault string

func (err fault) Error() string { return string(err) }

// Input is a medium-neutral conversation event.
type Input struct {
	ID      string
	Content string
}

// RunnerFactory builds a thought runner for one input. The runner sees the adopted input in its snapshot.
type RunnerFactory func(Input) thought.Runner

// Sink delivers only an adopted response.
type Sink func(context.Context, Input, string) error

// Loop owns one always-on interaction Channel and serializes its inputs.
type Loop struct {
	session *lifecycle.Session
	runner  RunnerFactory
	sink    Sink
	mu      sync.Mutex
	paused  bool
	closed  bool
	changed chan struct{}
	serial  sync.Mutex
	cleanup <-chan struct{}
}

func New(session *lifecycle.Session, runner RunnerFactory, sink Sink) (*Loop, error) {
	if session == nil || runner == nil || sink == nil {
		return nil, ErrConfig
	}
	loop := &Loop{session: session, runner: runner, sink: sink, mu: sync.Mutex{}, paused: false, closed: false,
		changed: make(chan struct{}), serial: sync.Mutex{}, cleanup: nil}

	return loop, nil
}

// Pause stops admitting new inputs and cancels the current thought. Resume reopens admission.
func (loop *Loop) Pause() {
	loop.mu.Lock()
	loop.paused = true
	loop.signalLocked()
	loop.mu.Unlock()
	loop.session.Cancel("interaction")
}

func (loop *Loop) Resume() {
	loop.mu.Lock()
	loop.paused = false
	loop.signalLocked()
	loop.mu.Unlock()
}

// Close prevents new work and cancels the current thought. Session.Close owns final shutdown.
func (loop *Loop) Close() {
	loop.mu.Lock()
	loop.closed = true
	loop.signalLocked()
	loop.mu.Unlock()
	loop.session.Cancel("interaction")
}

// Run continuously consumes inputs until the source closes or the loop is canceled.
//
//nolint:noinlineerr // The returned error belongs to the Run boundary.
func (loop *Loop) Run(ctx context.Context, inputs <-chan Input) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case input, open := <-inputs:
			if !open {
				return nil
			}
			if err := loop.Handle(ctx, input); err != nil {
				if errors.Is(err, context.Canceled) && ctx.Err() != nil {
					return ctx.Err()
				}

				return err
			}
		}
	}
}

// Handle runs one input through input adoption, thought, proposal adoption, and response delivery.
//
//nolint:noinlineerr,cyclop // Errors are returned to the conversation source across lifecycle boundaries.
func (loop *Loop) Handle(ctx context.Context, input Input) error {
	if input.ID == "" {
		return ErrInput
	}
	if err := loop.awaitOpen(ctx); err != nil {
		return err
	}
	loop.serial.Lock()
	defer loop.serial.Unlock()
	if loop.cleanup != nil {
		select {
		case <-loop.cleanup:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	if err := loop.awaitOpen(ctx); err != nil {
		return err
	}
	inputKey := "interaction/" + input.ID + "/input"
	snapshot := loop.session.Snapshot()
	base := snapshot.Items[inputKey].Version
	if _, err := loop.session.Adopt(ctx, mind.Proposal{ID: "input/" + input.ID,
		Reads: map[string]uint64{inputKey: base}, Writes: map[string]string{inputKey: input.Content}}, nil); err != nil {
		return err
	}
	if _, err := loop.session.Start("interaction", loop.runner(input)); err != nil {
		return err
	}
	result, err := loop.session.Next(ctx)
	if err != nil {
		loop.session.Cancel("interaction")
		if ctx.Err() != nil {
			loop.reapCanceled(context.WithoutCancel(ctx))
		}

		return err
	}
	if result.Err != nil {
		_ = loop.session.Discard(ctx, result.Token)

		return result.Err
	}
	responseKey := "interaction/" + input.ID + "/response"
	response := result.Proposal.Writes[responseKey]
	if response == "" {
		_ = loop.session.Discard(ctx, result.Token)

		return ErrResponse
	}
	if _, err = loop.session.Accept(ctx, result.Token, []continuity.Intent{}); err != nil {
		return err
	}

	return loop.sink(ctx, input, response)
}

// reapCanceled resolves the canceled generation so it cannot occupy the Channel slot forever.
func (loop *Loop) reapCanceled(ctx context.Context) {
	done := make(chan struct{})
	loop.cleanup = done

	go func() {
		defer close(done)

		result, err := loop.session.Next(ctx)
		if err != nil {
			return
		}
		_ = loop.session.Discard(ctx, result.Token)
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
			err := ctx.Err()
			if err != nil {
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

func (loop *Loop) signalLocked() {
	close(loop.changed)
	loop.changed = make(chan struct{})
}
