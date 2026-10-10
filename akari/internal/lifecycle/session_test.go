package lifecycle

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kizuna-org/akari/internal/action"
	"github.com/kizuna-org/akari/internal/continuity"
	"github.com/kizuna-org/akari/internal/meaning"
	"github.com/kizuna-org/akari/internal/memory"
	"github.com/kizuna-org/akari/internal/mind"
	"github.com/kizuna-org/akari/internal/persistence"
)

const (
	toolName    = "local-test"
	actionID    = "work-1"
	destination = "local-target"
)

func configuration() Config {
	config := FoundationConfig()
	config.State.Inner.Clock = func() time.Time { return time.Unix(100, 0).UTC() }
	config.State.Scope = func(mind.Snapshot) action.Scope {
		return action.Scope{AllowedDestinations: []string{destination}, ReadyRecipients: []string{destination}}
	}

	return config
}

func testError(t *testing.T, actual, expected error) {
	t.Helper()

	if !errors.Is(actual, expected) {
		t.Fatalf("error = %v, want %v", actual, expected)
	}
}

func fixture(
	t *testing.T, call func(context.Context, string) ([]byte, error),
) (*Session, *persistence.File, *action.Gateway) {
	t.Helper()

	store, err := persistence.New(t.TempDir())
	testError(t, err, nil)
	gateway, err := action.New(map[string]action.Prepare{toolName: func(arguments []byte) (action.Operation, error) {
		return action.Operation{Impact: action.Reversible, Destination: destination,
			ConflictKey: string(arguments), Call: nil, CallWithKey: call}, nil
	}}, 2)
	testError(t, err, nil)
	session, err := Open(t.Context(), store, gateway, configuration())
	testError(t, err, nil)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(t.Context()), time.Second)
		defer cancel()

		_ = session.Close(ctx)
	})

	return session, store, gateway
}

func proposal() mind.Proposal {
	return mind.Proposal{ID: "adopted", Reads: map[string]uint64{"context": 0},
		Writes: map[string]string{"context": "kept"}}
}

func intents() []continuity.Intent {
	return []continuity.Intent{{ID: actionID, Tool: toolName, Arguments: []byte("target")}}
}

func TestNormalStopRestoresStateAndRejectsDuplicate(t *testing.T) {
	t.Parallel()

	for _, execute := range []bool{false, true} {
		t.Run(map[bool]string{false: "queued", true: "succeeded"}[execute], func(t *testing.T) {
			t.Parallel()

			var calls atomic.Int32

			session, store, gateway := fixture(t, func(context.Context, string) ([]byte, error) {
				calls.Add(1)

				return []byte("complete"), nil
			})
			_, err := session.Adopt(t.Context(), proposal(), intents())
			testError(t, err, nil)

			if execute {
				_, err = session.Dispatch(t.Context(), actionID)
				testError(t, err, nil)
			}

			before := session.Export().Sequence
			testError(t, session.Close(t.Context()), nil)
			testError(t, session.Close(t.Context()), nil)
			<-session.Done()

			restored, err := Open(t.Context(), store, gateway, configuration())

			testError(t, err, nil)

			defer func() { testError(t, restored.Close(t.Context()), nil) }()

			if restored.Snapshot().Items["context"].Content != "kept" || restored.Export().Sequence != before+1 {
				t.Fatal("final adopted state was not restored")
			}

			if execute {
				_, err = restored.Dispatch(t.Context(), actionID)
				testError(t, err, continuity.ErrAction)

				if calls.Load() != 1 {
					t.Fatal("completed action was duplicated")
				}
			} else if restored.Export().Outbox[actionID].Stage != continuity.Queued || calls.Load() != 0 {
				t.Fatal("startup automatically dispatched queued work")
			}
		})
	}
}

func TestStopCancelsOrSealsRunningActions(t *testing.T) {
	t.Parallel()

	for _, cooperative := range []bool{false, true} {
		t.Run(map[bool]string{false: "deadline", true: "cooperative"}[cooperative], func(t *testing.T) {
			t.Parallel()

			started := make(chan struct{})
			release := make(chan struct{})
			finished := make(chan error, 1)
			session, store, gateway := fixture(t, func(ctx context.Context, _ string) ([]byte, error) {
				close(started)

				if cooperative {
					<-ctx.Done()

					return nil, ctx.Err()
				}

				<-release

				return []byte("late success"), nil
			})
			_, err := session.Adopt(t.Context(), proposal(), intents())

			testError(t, err, nil)

			go func() {
				_, err := session.Dispatch(t.Context(), actionID)
				finished <- err
			}()

			<-started

			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			if !cooperative {
				cancel()
			}

			err = session.Close(ctx)
			if cooperative {
				testError(t, err, nil)
				testError(t, <-finished, context.Canceled)
			} else {
				testError(t, err, context.Canceled)
				_, openErr := Open(t.Context(), store, gateway, configuration())
				testError(t, openErr, continuity.ErrBusy)
				close(release)
				testError(t, <-finished, continuity.ErrClosed)
			}

			<-session.Done()

			frame, err := store.Load(t.Context())
			testError(t, err, nil)

			if frame.Outbox[actionID].Stage != continuity.Unknown {
				t.Fatal("interrupted action was recorded as repeatable or successful")
			}

			restored, err := Open(t.Context(), store, gateway, configuration())
			testError(t, err, nil)
			_, err = restored.Dispatch(t.Context(), actionID)
			testError(t, err, continuity.ErrAction)
			testError(t, restored.Close(t.Context()), nil)
		})
	}
}

func TestSessionRejectsAdmissionDuringStop(t *testing.T) {
	t.Parallel()

	for _, canceled := range []bool{false, true} {
		t.Run(map[bool]string{false: "stopped", true: "canceled caller"}[canceled], func(t *testing.T) {
			t.Parallel()

			session, _, _ := fixture(t, func(context.Context, string) ([]byte, error) { return nil, nil })

			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			want := error(ErrClosed)

			if canceled {
				cancel()

				want = context.Canceled
			} else {
				testError(t, session.Close(t.Context()), nil)
			}

			_, err := session.Adopt(ctx, proposal(), nil)
			testError(t, err, want)
			_, err = session.Dispatch(ctx, actionID)
			testError(t, err, want)
			testError(t, session.Approve(ctx, actionID), want)
			testError(t, session.CancelAction(ctx, actionID), want)
			_, err = session.Recall(t.Context(), memory.Query{
				Cue: meaning.Point{Text: "cue", Space: "test", Vector: []float64{1}},
				At:  time.Unix(100, 0).UTC(), Freshness: time.Hour, Limit: 1,
			})
			testError(t, err, nil)
		})
	}
}
