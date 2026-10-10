package continuity_test

import (
	"context"
	"testing"
	"time"

	"github.com/kizuna-org/akari/internal/action"
	"github.com/kizuna-org/akari/internal/continuity"
	"github.com/kizuna-org/akari/internal/mind"
	"github.com/kizuna-org/akari/internal/thought"
)

const defaultAccept = "accept"

func TestThoughtAdoptionUsesDurableOwner(t *testing.T) {
	t.Parallel()

	for _, mode := range []string{defaultAccept, "accept with actions", "cancel while saving", "shutdown while saving"} {
		t.Run(mode, func(t *testing.T) {
			t.Parallel()

			owner, repo, gateway := rig(t, action.Reversible, func(context.Context, string) ([]byte, error) { return nil, nil })
			supervisor, err := thought.New(t.Context(), owner, 2)
			check(t, err, nil)
			t.Cleanup(func() {
				ctx, cancel := context.WithTimeout(context.WithoutCancel(t.Context()), time.Second)
				defer cancel()

				check(t, supervisor.Close(ctx), nil)
			})

			initialOwner := owner

			_, err = supervisor.Start("interaction", func(context.Context, mind.Snapshot) (mind.Proposal, error) {
				return proposal(initialOwner), nil
			})
			check(t, err, nil)
			result, err := supervisor.Next(t.Context())
			check(t, err, nil)

			result.Proposal.Writes[stateKey] = "mutated preview"
			_, err = supervisor.AcceptWith(result.Token, nil)
			check(t, err, thought.ErrToken)

			started := make(chan struct{})
			release := make(chan struct{})
			finished := make(chan error, 1)
			repo.save = func(ctx context.Context, expected uint64, next continuity.Frame) error {
				close(started)

				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-release:
					return repo.base.Save(ctx, expected, next)
				}
			}

			go func() {
				var err error

				if mode == defaultAccept {
					_, err = supervisor.Accept(result.Token)
				} else {
					_, err = supervisor.AcceptWith(result.Token, func(ctx context.Context, proposal mind.Proposal) (uint64, error) {
						return owner.Adopt(ctx, proposal, intents())
					})
				}

				finished <- err
			}()

			<-started

			_, err = supervisor.Accept(result.Token)
			check(t, err, thought.ErrToken)
			check(t, supervisor.Discard(result.Token), thought.ErrToken)
			_, err = supervisor.Start("independent", func(context.Context, mind.Snapshot) (mind.Proposal, error) {
				return proposal(initialOwner), nil
			})
			check(t, err, nil)

			if owner.Snapshot().Revision != 0 {
				t.Fatal("unsaved thought was published")
			}

			want := error(nil)

			switch mode {
			case "cancel while saving":
				supervisor.Cancel("interaction")

				want = context.Canceled
			case "shutdown while saving":
				check(t, supervisor.Close(t.Context()), nil)

				want = context.Canceled
			default:
				close(release)
			}

			check(t, <-finished, want)
			check(t, supervisor.Close(t.Context()), nil)

			repo.save = nil
			owner = reopen(t, repo, gateway)
			checkSavedThought(t, owner, want, mode)
		})
	}
}

func checkSavedThought(t *testing.T, owner *continuity.Owner, canceled error, mode string) {
	t.Helper()

	if canceled != nil {
		if owner.Snapshot().Revision != 0 || len(owner.Export().Outbox) != 0 {
			t.Fatal("canceled adoption survived restart")
		}

		return
	}

	if owner.Snapshot().Items[stateKey].Content != "adopted" || owner.Snapshot().Revision != 1 {
		t.Fatal("private delivered thought did not survive restart")
	}

	if (len(owner.Export().Outbox) == 1) != (mode != defaultAccept) {
		t.Fatal("thought and outbox were not adopted together")
	}
}
