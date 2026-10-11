package continuity_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/kizuna-org/akari/internal/action"
	"github.com/kizuna-org/akari/internal/continuity"
	"github.com/kizuna-org/akari/internal/memory"
	"github.com/kizuna-org/akari/internal/mind"
	"github.com/kizuna-org/akari/internal/persistence"
)

type loadedRepo struct {
	frame continuity.Frame
	err   error
}

func TestDurableDisclosureRechecksLatestAgreement(t *testing.T) {
	t.Parallel()

	for _, revoked := range []bool{false, true} {
		t.Run(map[bool]string{false: "permitted", true: "revoked after adoption"}[revoked], func(t *testing.T) {
			t.Parallel()

			store, err := persistence.New(t.TempDir())
			check(t, err, nil)

			calls := 0
			gateway, err := action.New(map[string]action.Prepare{toolName: func(arguments []byte) (action.Operation, error) {
				return action.Operation{Impact: action.Reversible, Destination: destination,
					ConflictKey: string(arguments), Sources: []string{privateConversation},
					Call: func(context.Context) ([]byte, error) {
						calls++

						return nil, nil
					}}, nil
			}}, 1)
			check(t, err, nil)
			owner := reopen(t, store, gateway)
			adoption := proposal(owner)
			adoption.Inner = agreementUpdate(owner, []string{destination})
			_, err = owner.Adopt(t.Context(), adoption, intents())
			check(t, err, nil)

			if revoked {
				adoption = proposal(owner)
				adoption.Inner = agreementUpdate(owner, nil)
				_, err = owner.Adopt(t.Context(), adoption, nil)
				check(t, err, nil)
			}

			owner = reopen(t, store, gateway)

			_, err = owner.Dispatch(t.Context(), intentID)
			if revoked {
				check(t, err, action.ErrDisclosure)

				if calls != 0 {
					t.Fatal("revoked private source was disclosed")
				}
			} else {
				check(t, err, nil)
			}
		})
	}
}

func agreementUpdate(owner *continuity.Owner, recipients []string) *mind.Update {
	return &mind.Update{Expected: owner.Snapshot().Experience.Versions,
		Memory: &memory.Change{Context: nil, Working: nil, Remember: nil, Recalled: nil,
			Agreements: []memory.Agreement{{Conversation: privateConversation, Recipients: recipients}}},
		Appraisal: nil, Interests: nil, Goals: nil, Load: 0, Rest: 0, Advance: false}
}

func TestOpenConfigurationAndStorageFailure(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name   string
		change func(*continuity.Config)
		load   error
		want   error
	}{
		{name: "scope", change: func(config *continuity.Config) { config.Scope = nil },
			load: nil, want: continuity.ErrConfig},
		{name: "outbox capacity", change: func(config *continuity.Config) { config.OutboxCapacity = 0 },
			load: nil, want: continuity.ErrConfig},
		{name: "save timeout", change: func(config *continuity.Config) { config.SaveTimeout = 0 },
			load: nil, want: continuity.ErrConfig},
		{name: "load failure", change: func(*continuity.Config) {}, load: context.Canceled, want: context.Canceled},
		{name: "initial save failure", change: func(*continuity.Config) {},
			load: continuity.ErrAbsent, want: continuity.ErrConflict},
		{name: "workspace capacity", change: func(config *continuity.Config) { config.WorkspaceCapacity = 0 },
			load: continuity.ErrAbsent, want: mind.ErrConfig},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			owner, _, gateway := rig(t, action.Reversible, func(context.Context, string) ([]byte, error) { return nil, nil })
			config := settings()
			test.change(&config)
			_, err := continuity.Open(t.Context(), loadedRepo{frame: owner.Export(), err: test.load}, gateway, config)
			check(t, err, test.want)
		})
	}
}

func TestQueuedDescriptorIsRevalidated(t *testing.T) {
	t.Parallel()

	for _, altered := range []bool{false, true} {
		t.Run(map[bool]string{false: "stable adapter", true: "changed adapter"}[altered], func(t *testing.T) {
			t.Parallel()

			store, err := persistence.New(t.TempDir())
			check(t, err, nil)

			impact := action.Reversible
			calls := 0
			gateway, err := action.New(map[string]action.Prepare{toolName: func(arguments []byte) (action.Operation, error) {
				return action.Operation{Impact: impact, Destination: destination, ConflictKey: string(arguments),
					Call: func(context.Context) ([]byte, error) {
						calls++

						return nil, nil
					}}, nil
			}}, 1)
			check(t, err, nil)
			owner := reopen(t, store, gateway)
			_, err = owner.Adopt(t.Context(), proposal(owner), intents())
			check(t, err, nil)

			if altered {
				impact = action.Read
			}

			owner = reopen(t, store, gateway)

			_, err = owner.Dispatch(t.Context(), intentID)
			if altered {
				check(t, err, continuity.ErrState)

				if calls != 0 {
					t.Fatal("changed descriptor dispatched")
				}
			} else {
				check(t, err, nil)
			}
		})
	}
}

func (repo loadedRepo) Load(context.Context) (continuity.Frame, error) {
	return repo.frame.Clone(), repo.err
}

func (repo loadedRepo) Save(context.Context, uint64, continuity.Frame) error {
	return continuity.ErrConflict
}

func TestRejectedDurableAdoption(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name   string
		change func(*mind.Proposal, *[]continuity.Intent)
		want   error
	}{
		{name: "prediction", change: func(proposal *mind.Proposal, _ *[]continuity.Intent) {
			proposal.Prediction = true
		}, want: mind.ErrPrediction},
		{name: "missing proposal", change: func(proposal *mind.Proposal, _ *[]continuity.Intent) {
			proposal.ID = ""
		}, want: mind.ErrProposal},
		{name: "stale dependency", change: func(proposal *mind.Proposal, _ *[]continuity.Intent) {
			proposal.Reads[stateKey] = 99
		}, want: mind.ErrConflict},
		{name: "missing action identity", change: func(_ *mind.Proposal, intents *[]continuity.Intent) {
			(*intents)[0].ID = ""
		}, want: continuity.ErrAction},
		{name: "reused identity", change: func(_ *mind.Proposal, intents *[]continuity.Intent) {},
			want: continuity.ErrAction},
		{name: "unknown tool", change: func(_ *mind.Proposal, intents *[]continuity.Intent) {
			(*intents)[0].ID = "new-work"
			(*intents)[0].Tool = "unknown"
		}, want: action.ErrUnknown},
		{name: "capacity", change: func(_ *mind.Proposal, intents *[]continuity.Intent) {
			*intents = make([]continuity.Intent, 11)
		}, want: mind.ErrCapacity},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			owner, _, _ := rig(t, action.Reversible, func(context.Context, string) ([]byte, error) { return nil, nil })
			_, err := owner.Adopt(t.Context(), proposal(owner), intents())
			check(t, err, nil)

			before := owner.Export()
			adoption, work := proposal(owner), intents()
			test.change(&adoption, &work)
			_, err = owner.Adopt(t.Context(), adoption, work)
			check(t, err, test.want)

			if !reflect.DeepEqual(before, owner.Export()) {
				t.Fatal("rejected adoption partly committed")
			}
		})
	}
}

func TestOpenRejectsInvalidDurableState(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name   string
		change func(*continuity.Frame)
	}{
		{name: "schema", change: func(frame *continuity.Frame) { frame.Schema++ }},
		{name: "installation", change: func(frame *continuity.Frame) { frame.Identity = "" }},
		{name: "sequence", change: func(frame *continuity.Frame) { frame.Sequence = 0 }},
		{name: "outbox", change: func(frame *continuity.Frame) { frame.Outbox = nil }},
		{name: "stage", change: func(frame *continuity.Frame) {
			record := frame.Outbox[intentID]
			record.Stage = "invalid"
			frame.Outbox[intentID] = record
		}},
		{name: "identity", change: func(frame *continuity.Frame) {
			record := frame.Outbox[intentID]
			record.ID = "other"
			frame.Outbox[intentID] = record
		}},
		{name: "effect", change: func(frame *continuity.Frame) {
			record := frame.Outbox[intentID]
			record.Command.Impact = "invalid"
			frame.Outbox[intentID] = record
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			owner, _, gateway := rig(t, action.Reversible, func(context.Context, string) ([]byte, error) { return nil, nil })
			_, err := owner.Adopt(t.Context(), proposal(owner), intents())
			check(t, err, nil)

			frame := owner.Export()
			test.change(&frame)
			_, err = continuity.Open(t.Context(), loadedRepo{frame: frame, err: nil}, gateway, settings())
			check(t, err, continuity.ErrState)
		})
	}
}

func TestDurableAdminGuardsAndStaleOwners(t *testing.T) {
	t.Parallel()

	for _, canceled := range []bool{false, true} {
		t.Run(map[bool]string{false: "state guards", true: "context guards"}[canceled], func(t *testing.T) {
			t.Parallel()

			owner, repo, gateway := rig(t, action.Reversible, func(context.Context, string) ([]byte, error) { return nil, nil })

			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			want := error(continuity.ErrAction)

			if canceled {
				cancel()

				want = context.Canceled
			}

			check(t, owner.Approve(ctx, intentID), want)
			check(t, owner.Cancel(ctx, intentID), want)
			check(t, owner.Reconcile(ctx, intentID, nil), want)
			other := reopen(t, repo, gateway)
			_, err := owner.Adopt(t.Context(), proposal(owner), intents())
			check(t, err, nil)
			_, err = other.Adopt(t.Context(), proposal(other), intents())
			check(t, err, continuity.ErrUnavailable)
			_, err = other.Dispatch(t.Context(), intentID)
			check(t, err, continuity.ErrUnavailable)
		})
	}
}

func TestDurableResolverFailures(t *testing.T) {
	t.Parallel()

	for _, status := range []action.Status{action.Unknown, action.NotExecuted} {
		t.Run(string(status), func(t *testing.T) {
			t.Parallel()

			owner, _, _ := rig(t, action.Reversible, func(context.Context, string) ([]byte, error) {
				return nil, context.DeadlineExceeded
			})
			_, err := owner.Adopt(t.Context(), proposal(owner), intents())
			check(t, err, nil)
			_, err = owner.Dispatch(t.Context(), intentID)
			check(t, err, context.DeadlineExceeded)

			resolver := lookupFunc(func(context.Context, string, action.Command) (action.Observation, error) {
				return action.Observation{Status: status, Data: nil}, nil
			})

			want := error(nil)
			if status == action.Unknown {
				want = continuity.ErrState
			}

			check(t, owner.Reconcile(t.Context(), intentID, resolver), want)

			if status == action.Unknown {
				resolver = func(context.Context, string, action.Command) (action.Observation, error) {
					return action.Observation{Status: action.Unknown, Data: nil}, context.DeadlineExceeded
				}
				check(t, owner.Reconcile(t.Context(), intentID, resolver), context.DeadlineExceeded)
			}
		})
	}
}
