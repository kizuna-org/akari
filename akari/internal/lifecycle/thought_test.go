package lifecycle

import (
	"context"
	"testing"

	"github.com/kizuna-org/akari/internal/continuity"
	"github.com/kizuna-org/akari/internal/mind"
	"github.com/kizuna-org/akari/internal/persistence"
	"github.com/kizuna-org/akari/internal/thought"
)

func TestThoughtCancellationDuringShutdown(t *testing.T) {
	t.Parallel()

	for _, adopt := range []bool{false, true} {
		t.Run(map[bool]string{false: "cancel running thought", true: "adopt delivered thought"}[adopt], func(t *testing.T) {
			t.Parallel()

			session, store, gateway := fixture(t, func(context.Context, string) ([]byte, error) { return nil, nil })
			started := make(chan struct{})
			_, err := session.Start("interaction", func(ctx context.Context, _ mind.Snapshot) (mind.Proposal, error) {
				close(started)

				if !adopt {
					<-ctx.Done()

					return mind.Proposal{}, ctx.Err()
				}

				return proposal(), nil
			})
			testError(t, err, nil)
			<-started

			if adopt {
				result, err := session.Next(t.Context())
				testError(t, err, nil)
				_, err = session.Accept(t.Context(), result.Token, intents())
				testError(t, err, nil)
				testError(t, session.CancelAction(t.Context(), actionID), nil)
			} else {
				session.Cancel("interaction")
			}

			testError(t, session.Close(t.Context()), nil)
			_, err = session.Start("new", func(context.Context, mind.Snapshot) (mind.Proposal, error) {
				return proposal(), nil
			})
			testError(t, err, ErrClosed)
			restored, err := Open(t.Context(), store, gateway, configuration())
			testError(t, err, nil)

			if (restored.Snapshot().Revision == 1) != adopt {
				t.Fatal("canceled thought became an adopted experience")
			}

			if adopt {
				if restored.Export().Outbox[actionID].Stage != continuity.NotExecuted {
					t.Fatal("explicitly canceled outbox entry became runnable")
				}

				_, err = restored.Accept(t.Context(), thought.Token{Channel: "interaction", Generation: 1}, nil)
				testError(t, err, thought.ErrToken)
			}

			testError(t, restored.Close(t.Context()), nil)
		})
	}
}

type failingRepository struct {
	*persistence.File

	save func(context.Context, uint64, continuity.Frame) error
}

func (repository failingRepository) Save(ctx context.Context, expected uint64, next continuity.Frame) error {
	return repository.save(ctx, expected, next)
}

func TestStopDuringCanceledSaveKeepsAuthoritativeState(t *testing.T) {
	t.Parallel()

	for _, ambiguous := range []bool{false, true} {
		name := map[bool]string{false: "before write", true: "write then failed acknowledgment"}[ambiguous]
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			initial, store, gateway := fixture(t, func(context.Context, string) ([]byte, error) { return nil, nil })
			testError(t, initial.Close(t.Context()), nil)

			started := make(chan struct{})
			finished := make(chan error, 1)
			save := func(ctx context.Context, expected uint64, next continuity.Frame) error {
				if ambiguous {
					testError(t, store.Save(ctx, expected, next), nil)
				}

				close(started)
				<-ctx.Done()

				return ctx.Err()
			}
			repository := failingRepository{File: store, save: save}
			session, err := Open(t.Context(), repository, gateway, configuration())

			testError(t, err, nil)

			go func() {
				_, err := session.Adopt(t.Context(), proposal(), intents())
				finished <- err
			}()

			<-started
			testError(t, session.Close(t.Context()), continuity.ErrUnavailable)
			testError(t, <-finished, context.Canceled)
			<-session.Done()

			restored, err := Open(t.Context(), store, gateway, configuration())
			testError(t, err, nil)

			if (restored.Snapshot().Revision == 1) != ambiguous {
				t.Fatal("shutdown guessed the outcome of an ambiguous save")
			}

			testError(t, restored.Close(t.Context()), nil)
		})
	}
}
