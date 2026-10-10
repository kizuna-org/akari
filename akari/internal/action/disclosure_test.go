package action

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kizuna-org/akari/internal/emotion"
	"github.com/kizuna-org/akari/internal/memory"
	"github.com/kizuna-org/akari/internal/mind"
	"github.com/kizuna-org/akari/internal/persona"
)

const (
	deniedDisclosure = "denied"
	missingPolicy    = "missing policy"
	changedAgreement = "agreement changed"
)

func TestKnownMemoryDisclosure(t *testing.T) {
	t.Parallel()

	for _, reason := range []string{"allowed", deniedDisclosure, missingPolicy, changedAgreement} {
		t.Run(reason, func(t *testing.T) {
			t.Parallel()

			workspace, err := mind.NewWithInner(1, persona.Default(), mind.Settings{
				Contents: 10, Dynamics: emotion.Dynamics{EmotionHalfLife: time.Hour, MoodHalfLife: time.Hour},
				RestUnit: time.Hour, Clock: func() time.Time { return time.Unix(100, 0) },
			})
			assertError(t, err, nil)

			calls := new(atomic.Int32)
			sources := []string{"private"}
			gateway := newGateway(t, 1, func([]byte) (Operation, error) {
				return Operation{Impact: Read, Destination: testDestination, ConflictKey: "", Sources: sources,
					Call: func(ctx context.Context) ([]byte, error) {
						calls.Add(1)

						return readCall(ctx)
					}}, nil
			})
			scope := localScope()
			scope.Disclosure = workspace.AllowsDisclosure

			if reason == missingPolicy {
				scope.Disclosure = nil
			}

			if reason == deniedDisclosure {
				lockConversation(t, workspace)
			}

			pending, err := gateway.Open(false, scope).Plan(t.Context(), testTool, nil)
			if reason == deniedDisclosure || reason == missingPolicy {
				assertError(t, err, ErrDisclosure)

				return
			}

			assertError(t, err, nil)

			var want error

			if reason == changedAgreement {
				lockConversation(t, workspace)

				sources[0] = "public"
				want = ErrDisclosure
			}

			observation, err := pending.Execute(t.Context())
			assertError(t, err, want)

			if want != nil && (observation.Status != NotExecuted || calls.Load() != 0) {
				t.Fatal("new confidentiality agreement failed to stop a planned external search")
			}
		})
	}
}

func lockConversation(t *testing.T, workspace *mind.Workspace) {
	t.Helper()

	change := new(memory.Change)
	change.Agreements = []memory.Agreement{{Conversation: "private", Recipients: []string{"alice"}}}
	update := new(mind.Update)
	update.Expected = workspace.Snapshot().Experience.Versions
	update.Memory = change
	_, err := workspace.Commit(mind.Proposal{ID: "agreement", Reads: nil, Writes: nil, Inner: update, Prediction: false})
	assertError(t, err, nil)
}
