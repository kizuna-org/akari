package action

import (
	"context"
	"testing"

	"github.com/kizuna-org/akari/internal/mind"
	"github.com/kizuna-org/akari/internal/thought"
)

func TestChannelsShareBoundary(t *testing.T) {
	t.Parallel()

	for _, adapterFailure := range []bool{false, true} {
		t.Run(map[bool]string{false: "slow lookup", true: "failed lookup"}[adapterFailure], func(t *testing.T) {
			t.Parallel()

			started := make(chan struct{}, 1)
			release := make(chan struct{})
			gateway := newGateway(t, 2, func(arguments []byte) (Operation, error) {
				key := string(arguments)

				return Operation{Impact: Read, Destination: testDestination, ConflictKey: "",
					Call: func(ctx context.Context) ([]byte, error) {
						return lookup(ctx, key, adapterFailure, started, release)
					}}, nil
			})
			workspace, err := mind.New(2)
			assertError(t, err, nil)
			supervisor, err := thought.New(t.Context(), workspace, 2)
			assertError(t, err, nil)

			t.Cleanup(func() {
				ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
				defer cancel()

				assertError(t, supervisor.Close(ctx), nil)
			})

			_, err = supervisor.Start("slow", lookupRunner(gateway.Open(false, localScope()), "slow"))
			assertError(t, err, nil)
			receive(t, started)

			_, err = supervisor.Start("fast", lookupRunner(gateway.Open(false, localScope()), "fast"))
			assertError(t, err, nil)
			fast := completion(t, supervisor)

			if fast.Token.Channel != "fast" {
				t.Fatal("slow external lookup blocked a different Channel")
			}

			_, err = supervisor.Accept(fast.Token)
			assertError(t, err, nil)
			close(release)

			slow := completion(t, supervisor)
			_, err = supervisor.Accept(slow.Token)
			want := map[bool]error{false: nil, true: ErrPanic}[adapterFailure]
			assertError(t, err, want)

			if workspace.Snapshot().Items["fast"].Content != "fast" {
				t.Fatal("external failure damaged the other Channel's adopted result")
			}

			waitStats(t, gateway, Stats{Executing: 0, TargetUsers: 0})
		})
	}
}

func lookup(
	ctx context.Context, key string, fail bool, started chan<- struct{}, release <-chan struct{},
) ([]byte, error) {
	if key == "slow" {
		started <- struct{}{}

		select {
		case <-release:
		case <-ctx.Done():
			return nil, ctx.Err()
		}

		if fail {
			panic("simulated adapter failure")
		}
	}

	return []byte(key), nil
}

func lookupRunner(session *Session, key string) thought.Runner {
	return func(ctx context.Context, snapshot mind.Snapshot) (mind.Proposal, error) {
		pending, err := session.Plan(ctx, testTool, []byte(key))
		if err != nil {
			return mind.Proposal{}, err
		}

		observation, err := pending.Execute(ctx)
		if err != nil {
			return mind.Proposal{}, err
		}

		return mind.Proposal{
			ID: "runtime assigns this", Reads: map[string]uint64{key: snapshot.Items[key].Version},
			Writes: map[string]string{key: string(observation.Data)},
		}, nil
	}
}

func completion(t *testing.T, supervisor *thought.Supervisor) thought.Result {
	t.Helper()

	ctx, cancel := context.WithTimeout(t.Context(), testTimeout)
	defer cancel()

	result, err := supervisor.Next(ctx)
	assertError(t, err, nil)

	return result
}
