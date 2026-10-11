package lifecycle

import (
	"context"
	"testing"

	"github.com/kizuna-org/akari/internal/action"
	"github.com/kizuna-org/akari/internal/continuity"
)

func TestFailedStartupReleasesLease(t *testing.T) {
	t.Parallel()

	for _, corrupt := range []bool{false, true} {
		t.Run(map[bool]string{false: "configuration", true: "corrupt checkpoint"}[corrupt], func(t *testing.T) {
			t.Parallel()

			session, store, gateway := fixture(t, func(context.Context, string) ([]byte, error) { return nil, nil })
			testError(t, session.Close(t.Context()), nil)

			config := configuration()
			want := error(ErrConfig)

			if corrupt {
				frame, err := store.Load(t.Context())
				testError(t, err, nil)

				sequence := frame.Sequence
				frame.Sequence++
				frame.Schema++
				testError(t, store.Save(t.Context(), sequence, frame), nil)

				want = continuity.ErrState
			} else {
				config.ThoughtCapacity = 0
			}

			_, err := Open(t.Context(), store, gateway, config)
			testError(t, err, want)
			release, err := store.Acquire(t.Context())
			testError(t, err, nil)
			release()
		})
	}
}

func TestSessionApprovalSurvivesRestart(t *testing.T) {
	t.Parallel()

	for _, approved := range []bool{false, true} {
		t.Run(map[bool]string{false: "unapproved", true: "approved"}[approved], func(t *testing.T) {
			t.Parallel()

			session, store, _ := fixture(t, func(context.Context, string) ([]byte, error) { return nil, nil })
			testError(t, session.Close(t.Context()), nil)

			gateway, err := action.New(map[string]action.Prepare{toolName: func(arguments []byte) (action.Operation, error) {
				return action.Operation{Impact: action.Irreversible, Destination: destination,
					ConflictKey: string(arguments), Call: func(context.Context) ([]byte, error) { return nil, nil }}, nil
			}}, 1)
			testError(t, err, nil)
			session, err = Open(t.Context(), store, gateway, configuration())
			testError(t, err, nil)
			_, err = session.Adopt(t.Context(), proposal(), intents())
			testError(t, err, nil)

			if approved {
				testError(t, session.Approve(t.Context(), actionID), nil)
			}

			testError(t, session.Close(t.Context()), nil)
			session, err = Open(t.Context(), store, gateway, configuration())
			testError(t, err, nil)

			_, err = session.Dispatch(t.Context(), actionID)
			if approved {
				testError(t, err, nil)
			} else {
				testError(t, err, action.ErrApproval)
			}

			testError(t, session.Close(t.Context()), nil)
		})
	}
}
