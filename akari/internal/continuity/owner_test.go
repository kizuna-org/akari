package continuity_test

import (
	"context"
	"errors"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kizuna-org/akari/internal/action"
	"github.com/kizuna-org/akari/internal/continuity"
	"github.com/kizuna-org/akari/internal/emotion"
	"github.com/kizuna-org/akari/internal/goal"
	"github.com/kizuna-org/akari/internal/meaning"
	"github.com/kizuna-org/akari/internal/memory"
	"github.com/kizuna-org/akari/internal/mind"
	"github.com/kizuna-org/akari/internal/persistence"
	"github.com/kizuna-org/akari/internal/persona"
)

const (
	beforeAdoption      = "before adoption"
	afterClaim          = "after claim"
	privateConversation = "private"
	duringCall          = "during call"
	toolName            = "local-test"
	intentID            = "work-1"
	destination         = "local-target"
	stateKey            = "context"
)

type repository struct {
	base continuity.Repository
	save func(context.Context, uint64, continuity.Frame) error
}

func (repo *repository) Load(ctx context.Context) (continuity.Frame, error) {
	return repo.base.Load(ctx)
}

func (repo *repository) Save(ctx context.Context, expected uint64, next continuity.Frame) error {
	if repo.save != nil {
		return repo.save(ctx, expected, next)
	}

	return repo.base.Save(ctx, expected, next)
}

type lookupFunc func(context.Context, string, action.Command) (action.Observation, error)

func (lookup lookupFunc) Lookup(
	ctx context.Context, identity string, command action.Command,
) (action.Observation, error) {
	return lookup(ctx, identity, command)
}

func settings() continuity.Config {
	return continuity.Config{
		Persona: persona.Default(), WorkspaceCapacity: 100, OutboxCapacity: 10, SaveTimeout: time.Second,
		Inner: mind.Settings{Contents: 100, Clock: func() time.Time { return time.Unix(100, 0).UTC() },
			Dynamics: emotion.Dynamics{EmotionHalfLife: time.Hour, MoodHalfLife: time.Hour}, RestUnit: time.Hour},
		Scope: func(mind.Snapshot) action.Scope {
			return action.Scope{AllowedDestinations: []string{destination}, ReadyRecipients: []string{destination}}
		},
	}
}

func rig(t *testing.T, impact action.Impact, call func(context.Context, string) ([]byte, error)) (
	*continuity.Owner, *repository, *action.Gateway,
) {
	t.Helper()

	store, err := persistence.New(t.TempDir())
	check(t, err, nil)

	repo := &repository{base: store, save: nil}
	gateway, err := action.New(map[string]action.Prepare{toolName: func(arguments []byte) (action.Operation, error) {
		return action.Operation{Impact: impact, Destination: destination,
			ConflictKey: string(arguments), Call: nil, CallWithKey: call}, nil
	}}, 2)
	check(t, err, nil)
	owner, err := continuity.Open(t.Context(), repo, gateway, settings())
	check(t, err, nil)

	return owner, repo, gateway
}

func proposal(owner *continuity.Owner) mind.Proposal {
	snapshot := owner.Snapshot()

	return mind.Proposal{ID: "adoption", Reads: map[string]uint64{stateKey: snapshot.Items[stateKey].Version},
		Writes: map[string]string{stateKey: "adopted"}}
}

func intents() []continuity.Intent {
	return []continuity.Intent{{ID: intentID, Tool: toolName, Arguments: []byte("target")}}
}

func check(t *testing.T, err, expected error) {
	t.Helper()

	if !errors.Is(err, expected) {
		t.Fatalf("error = %v, want %v", err, expected)
	}
}

func reopen(t *testing.T, repo continuity.Repository, gateway *action.Gateway) *continuity.Owner {
	t.Helper()

	owner, err := continuity.Open(t.Context(), repo, gateway, settings())
	check(t, err, nil)

	return owner
}

func TestDurableCrashBoundaries(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name     string
		sequence uint64
		written  bool
		stage    continuity.Stage
		calls    int32
		revision uint64
	}{
		{name: beforeAdoption, sequence: 2, written: false, stage: "", calls: 0, revision: 0},
		{name: "adoption acknowledged ambiguously", sequence: 2, written: true,
			stage: continuity.Queued, calls: 0, revision: 1},
		{name: "before claim", sequence: 3, written: false, stage: continuity.Queued, calls: 0, revision: 1},
		{name: "after claim before dispatch", sequence: 3, written: true,
			stage: continuity.Unknown, calls: 0, revision: 1},
		{name: "after effect before outcome", sequence: 4, written: false,
			stage: continuity.Unknown, calls: 1, revision: 1},
		{name: "outcome acknowledged ambiguously", sequence: 4, written: true,
			stage: continuity.Succeeded, calls: 1, revision: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			var calls atomic.Int32

			owner, repo, gateway := rig(t, action.Reversible, func(context.Context, string) ([]byte, error) {
				calls.Add(1)

				return []byte("done"), nil
			})
			repo.save = func(ctx context.Context, expected uint64, next continuity.Frame) error {
				if next.Sequence != test.sequence || test.written {
					check(t, repo.base.Save(ctx, expected, next), nil)
				}

				if next.Sequence == test.sequence {
					return continuity.ErrConflict
				}

				return nil
			}

			_, err := owner.Adopt(t.Context(), proposal(owner), intents())
			if test.sequence == 2 {
				check(t, err, continuity.ErrUnavailable)

				if owner.Snapshot().Revision != 0 {
					t.Fatal("failed save exposed candidate")
				}
			} else {
				check(t, err, nil)
				_, err = owner.Dispatch(t.Context(), intentID)
				check(t, err, continuity.ErrUnavailable)
			}

			_, err = owner.Adopt(t.Context(), proposal(owner), nil)
			check(t, err, continuity.ErrUnavailable)

			repo.save = nil
			restored := reopen(t, repo, gateway)
			checkRecovered(t, restored, test.revision, test.stage, calls.Load(), test.calls)
		})
	}
}

func TestDurableIdentityApprovalAndReconciliation(t *testing.T) {
	t.Parallel()

	for _, impact := range []action.Impact{action.Reversible, action.Irreversible} {
		t.Run(string(impact), func(t *testing.T) {
			t.Parallel()

			var identity string

			owner, repo, gateway := rig(t, impact, func(_ context.Context, key string) ([]byte, error) {
				identity = key

				return nil, context.DeadlineExceeded
			})
			_, err := owner.Adopt(t.Context(), proposal(owner), intents())
			check(t, err, nil)

			if impact == action.Irreversible {
				_, err = owner.Dispatch(t.Context(), intentID)
				check(t, err, action.ErrApproval)
				check(t, owner.Approve(t.Context(), intentID), nil)
			}

			owner = reopen(t, repo, gateway)
			observation, err := owner.Dispatch(t.Context(), intentID)
			check(t, err, context.DeadlineExceeded)

			if observation.Status != action.Unknown || identity != owner.Export().Identity+"/"+intentID {
				t.Fatal("unknown outcome or durable identity lost")
			}

			owner = reopen(t, repo, gateway)
			_, err = owner.Dispatch(t.Context(), intentID)
			check(t, err, continuity.ErrAction)
			check(t, owner.Reconcile(t.Context(), intentID, lookupFunc(func(
				_ context.Context, key string, command action.Command,
			) (action.Observation, error) {
				if key != identity || command.Tool != toolName {
					t.Fatal("reconciliation identity changed")
				}

				return action.Observation{Status: action.Succeeded, Data: []byte("confirmed")}, nil
			})), nil)

			owner = reopen(t, repo, gateway)
			_, err = owner.Adopt(t.Context(), proposal(owner), intents())
			check(t, err, continuity.ErrAction)

			if owner.Export().Outbox[intentID].Stage != continuity.Succeeded {
				t.Fatal("reconciled outcome lost")
			}
		})
	}
}

func TestDurableCancellationBoundaries(t *testing.T) {
	t.Parallel()

	for _, boundary := range []string{beforeAdoption, "before dispatch", afterClaim, duringCall, "queued cancel"} {
		t.Run(boundary, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			var calls atomic.Int32

			owner, repo, gateway := rig(t, action.Reversible, func(ctx context.Context, _ string) ([]byte, error) {
				calls.Add(1)
				cancel()

				return nil, ctx.Err()
			})

			if boundary == beforeAdoption {
				cancel()

				_, err := owner.Adopt(ctx, proposal(owner), intents())
				check(t, err, context.Canceled)

				if owner.Snapshot().Revision != 0 {
					t.Fatal("canceled adoption changed state")
				}

				return
			}

			_, err := owner.Adopt(ctx, proposal(owner), intents())
			check(t, err, nil)

			if boundary == "queued cancel" {
				check(t, owner.Cancel(ctx, intentID), nil)
				owner = reopen(t, repo, gateway)
				_, err = owner.Dispatch(ctx, intentID)
				check(t, err, continuity.ErrAction)

				return
			}

			switch boundary {
			case "before dispatch":
				cancel()
			case afterClaim:
				repo.save = func(ctx context.Context, expected uint64, next continuity.Frame) error {
					err := repo.base.Save(ctx, expected, next)

					if next.Outbox[intentID].Stage == continuity.Running {
						cancel()
					}

					return err
				}
			}

			_, err = owner.Dispatch(ctx, intentID)
			check(t, err, context.Canceled)

			repo.save = nil
			owner = reopen(t, repo, gateway)
			checkCanceled(t, owner, boundary, calls.Load())
		})
	}
}

func TestDurableParallelDispatchAndAdoption(t *testing.T) {
	t.Parallel()

	for _, saveBlocked := range []bool{false, true} {
		t.Run(map[bool]string{false: "external wait", true: "save wait"}[saveBlocked], func(t *testing.T) {
			t.Parallel()

			started := make(chan struct{})
			release := make(chan struct{})
			finished := make(chan error, 1)
			owner, repo, _ := rig(t, action.Reversible, func(_ context.Context, key string) ([]byte, error) {
				if key[len(key)-len(intentID):] == intentID {
					close(started)
					<-release
				}

				return nil, nil
			})

			work := append(intents(), continuity.Intent{ID: "work-2", Tool: toolName, Arguments: []byte("different")})
			_, err := owner.Adopt(t.Context(), proposal(owner), work)
			check(t, err, nil)

			if saveBlocked {
				repo.save = func(ctx context.Context, expected uint64, next continuity.Frame) error {
					close(started)
					<-release

					return repo.base.Save(ctx, expected, next)
				}

				go func() {
					_, err := owner.Adopt(t.Context(), proposal(owner), nil)
					finished <- err
				}()
			} else {
				go func() {
					_, err := owner.Dispatch(t.Context(), intentID)
					finished <- err
				}()
			}

			<-started

			if owner.Snapshot().Revision != 1 {
				t.Fatal("unsaved revision was visible")
			}

			_, err = owner.Adopt(t.Context(), proposal(owner), nil)
			if saveBlocked {
				check(t, err, continuity.ErrBusy)
			} else {
				check(t, err, nil)
				_, err = owner.Dispatch(t.Context(), "work-2")
				check(t, err, nil)
			}

			close(release)
			check(t, <-finished, nil)
		})
	}
}

func TestDurableRestoresInnerExperience(t *testing.T) {
	t.Parallel()

	for _, paused := range []bool{false, true} {
		t.Run(map[bool]string{false: "working", true: "paused"}[paused], func(t *testing.T) {
			t.Parallel()

			owner, repo, gateway := rig(t, action.Reversible, func(context.Context, string) ([]byte, error) { return nil, nil })
			point := meaning.Point{Text: "keep promise", Space: "test-space", Vector: []float64{1}}
			fragment := memory.Fragment{ID: "experience", Meaning: point, OccurredAt: time.Unix(100, 0).UTC(),
				Emotion: 1, Will: 1, Strength: 1, Accesses: 0, Sources: []string{privateConversation}}
			adoption := proposal(owner)
			adoption.Inner = &mind.Update{Expected: owner.Snapshot().Experience.Versions,
				Appraisal: &emotion.Appraisal{Feelings: []emotion.Feeling{{Meaning: point, Strength: 1}},
					Others: nil, Readiness: nil, MoodDelta: emotion.Mood{1, 0, 0}},
				Memory: &memory.Change{Context: []memory.Fragment{fragment}, Working: []memory.Fragment{fragment},
					Remember: []memory.Fragment{fragment}, Recalled: nil,
					Agreements: []memory.Agreement{{Conversation: privateConversation, Recipients: []string{destination}}}},
				Goals: &goal.Change{Desires: nil, End: nil, Adopt: map[string]goal.Intention{
					"promise": {Meaning: point, Claims: map[string]string{"promise": "keep"},
						Steps: []meaning.Point{point, point}, Next: 1, Paused: paused},
				}}, Interests: nil, Load: 1, Rest: 0, Advance: false}
			_, err := owner.Adopt(t.Context(), adoption, intents())
			check(t, err, nil)

			before := owner.Export()

			owner = reopen(t, repo, gateway)
			if !reflect.DeepEqual(before, owner.Export()) ||
				!owner.AllowsDisclosure([]string{privateConversation}, destination) {
				t.Fatal("inner experience or confidentiality agreement changed on restart")
			}

			recalled, err := owner.Recall(t.Context(), memory.Query{Cue: point, At: time.Unix(100, 0).UTC(),
				Freshness: time.Hour, Limit: 1})
			check(t, err, nil)

			if len(recalled) != 1 || owner.Snapshot().Experience.Goals.Intentions["promise"].Next != 1 {
				t.Fatal("memory or continuation position lost")
			}

			before.Outbox[intentID].Command.Arguments[0] = 'X'
			before.Checkpoint.Snapshot.Experience.Goals.Intentions["promise"].Claims["promise"] = "changed"

			if owner.Export().Outbox[intentID].Command.Arguments[0] == 'X' {
				t.Fatal("administrative copy mutated live state")
			}
		})
	}
}

func checkRecovered(
	t *testing.T, owner *continuity.Owner, revision uint64, stage continuity.Stage, actual, expected int32,
) {
	t.Helper()

	frame := owner.Export()
	if frame.Checkpoint.Snapshot.Revision != revision || frame.Outbox[intentID].Stage != stage || actual != expected {
		t.Fatal("restored revision, stage or call count differs")
	}

	if stage == continuity.Unknown || stage == continuity.Succeeded {
		_, err := owner.Dispatch(t.Context(), intentID)
		check(t, err, continuity.ErrAction)
	}
}

func checkCanceled(t *testing.T, owner *continuity.Owner, boundary string, calls int32) {
	t.Helper()

	stage := continuity.Queued

	switch boundary {
	case afterClaim:
		stage = continuity.NotExecuted
	case duringCall:
		stage = continuity.Unknown
	}

	if owner.Export().Outbox[intentID].Stage != stage || (calls == 1) != (boundary == duringCall) {
		t.Fatal("cancellation was mistaken for a completed or repeatable operation")
	}
}
