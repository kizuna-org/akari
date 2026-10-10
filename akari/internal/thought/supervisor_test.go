package thought

import (
	"context"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/kizuna-org/akari/internal/mind"
)

const (
	testTimeout = 3 * time.Second
	fastChannel = "fast"
)

func TestParallelIndependence(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		key    string
		cancel bool
		want   error
	}{
		{name: "unrelated change", key: "slow", cancel: false, want: nil},
		{name: "canceled generation", key: "slow", cancel: true, want: context.Canceled},
		{name: "changed dependency", key: fastChannel, cancel: false, want: mind.ErrConflict},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			supervisor, workspace := newSupervisor(t, 2)
			started := make(chan struct{})
			release := make(chan struct{})
			once := new(sync.Once)
			finish := func() { once.Do(func() { close(release) }) }
			t.Cleanup(finish)

			_, err := supervisor.Start("slow", func(_ context.Context, _ mind.Snapshot) (mind.Proposal, error) {
				close(started)
				<-release

				return proposal(test.key), nil
			})
			if err != nil {
				t.Fatal(err)
			}

			<-started

			_, err = supervisor.Start(fastChannel, func(_ context.Context, _ mind.Snapshot) (mind.Proposal, error) {
				return proposal(fastChannel), nil
			})
			if err != nil {
				t.Fatal(err)
			}

			result := next(t, supervisor)
			if result.Token.Channel != fastChannel {
				t.Fatal("slow worker blocked conversation")
			}

			// The caller cannot alter the adopted proposal through its preview.
			result.Proposal.Writes[fastChannel] = "forged"

			_, err = supervisor.Accept(result.Token)
			if err != nil || workspace.Snapshot().Items[fastChannel].Content != fastChannel {
				t.Fatalf("accept fast: %v", err)
			}

			if test.cancel {
				supervisor.Cancel("slow")
			}

			finish()

			result = next(t, supervisor)

			_, err = supervisor.Accept(result.Token)
			assertError(t, err, test.want)

			_, err = supervisor.Accept(result.Token)
			assertError(t, err, ErrToken)
		})
	}
}

func TestCompletionOutcomes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		runner  Runner
		want    error
		discard bool
	}{
		{name: "success",
			runner: func(context.Context, mind.Snapshot) (mind.Proposal, error) { return proposal("a"), nil },
			want:   nil, discard: false},
		{name: "model failure", runner: func(context.Context, mind.Snapshot) (mind.Proposal, error) {
			return mind.Proposal{}, io.ErrUnexpectedEOF
		}, want: io.ErrUnexpectedEOF, discard: false},
		{name: "panic isolated",
			runner: func(context.Context, mind.Snapshot) (mind.Proposal, error) { panic("test") },
			want:   ErrPanic, discard: false},
		{name: "discard",
			runner: func(context.Context, mind.Snapshot) (mind.Proposal, error) { return proposal("a"), nil },
			want:   nil, discard: true},
		{name: "invalid dependency", runner: func(context.Context, mind.Snapshot) (mind.Proposal, error) {
			return mind.Proposal{ID: "ignored", Reads: nil, Writes: map[string]string{"a": "a"}}, nil
		}, want: mind.ErrProposal, discard: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			supervisor, _ := newSupervisor(t, 1)

			token, err := supervisor.Start("a", test.runner)
			if err != nil {
				t.Fatal(err)
			}

			_, err = supervisor.Accept(token)
			if !errors.Is(err, ErrToken) {
				t.Fatal("undelivered result was accepted")
			}

			result := next(t, supervisor)
			_, err = supervisor.Start("another", test.runner)
			assertError(t, err, ErrCapacity)

			bad := Token{Channel: token.Channel, Generation: token.Generation + 1}
			if test.discard {
				err = supervisor.Discard(bad)
			} else {
				_, err = supervisor.Accept(bad)
			}

			assertError(t, err, ErrToken)

			if test.discard {
				err = supervisor.Discard(result.Token)
			} else {
				_, err = supervisor.Accept(result.Token)
			}

			assertError(t, err, test.want)

			err = supervisor.Discard(token)
			assertError(t, err, ErrToken)

			nextToken, err := supervisor.Start("a", test.runner)
			if err != nil || nextToken.Generation <= token.Generation {
				t.Fatalf("slot was not released: %v", err)
			}
		})
	}
}

func TestWaitingCancellation(t *testing.T) {
	t.Parallel()

	for _, shutdown := range []bool{false, true} {
		t.Run(map[bool]string{false: "caller", true: "supervisor"}[shutdown], func(t *testing.T) {
			t.Parallel()

			supervisor, _ := newSupervisor(t, 1)

			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			// Observe entry into Next's wait, without relying on a sleep or scheduling luck.
			waiting := make(chan struct{}, 2)
			observed := waitContext{Context: ctx, waiting: waiting}
			completed := make(chan error, 1)

			go func() {
				_, err := supervisor.Next(observed)
				completed <- err
			}()

			<-waiting

			if shutdown {
				assertError(t, supervisor.Close(t.Context()), nil)
			} else {
				cancel()
			}

			select {
			case err := <-completed:
				assertError(t, err, context.Canceled)
			case <-time.After(testTimeout):
				t.Fatal("Next did not unblock")
			}
		})
	}
}

type waitContext struct {
	context.Context //nolint:containedctx // Decorates a context to observe entry into a blocking wait.

	waiting chan<- struct{}
}

func (ctx waitContext) Done() <-chan struct{} {
	ctx.waiting <- struct{}{}

	return ctx.Context.Done()
}

func TestAdmissionAndShutdown(t *testing.T) {
	t.Parallel()

	for _, closeEarly := range []bool{false, true} {
		t.Run(map[bool]string{false: "admission", true: "closed"}[closeEarly], func(t *testing.T) {
			t.Parallel()

			supervisor, _ := newSupervisor(t, 1)

			if closeEarly {
				closeCtx, cancel := context.WithTimeout(t.Context(), testTimeout)
				defer cancel()

				err := supervisor.Close(closeCtx)
				if err != nil {
					t.Fatal(err)
				}
			}

			_, err := supervisor.Start("", nil)
			if closeEarly {
				assertError(t, err, context.Canceled)

				_, err = supervisor.Next(t.Context())
				assertError(t, err, context.Canceled)

				return
			}

			assertError(t, err, ErrRunner)

			waitCtx, cancel := context.WithCancel(t.Context())
			cancel()

			_, err = supervisor.Next(waitCtx)
			assertError(t, err, context.Canceled)

			runner := func(ctx context.Context, _ mind.Snapshot) (mind.Proposal, error) {
				<-ctx.Done()

				return mind.Proposal{}, ctx.Err()
			}

			_, err = supervisor.Start("held", runner)
			if err != nil {
				t.Fatal(err)
			}

			_, err = supervisor.Start("held", runner)
			assertError(t, err, ErrActive)

			_, err = supervisor.Start("another", runner)
			assertError(t, err, ErrCapacity)

			supervisor.Cancel("missing")
			supervisor.Cancel("held")
			next(t, supervisor)
		})
	}
}

func TestConfigurationAndCloseDeadline(t *testing.T) {
	t.Parallel()

	for _, capacity := range []int{0, 1} {
		t.Run(map[int]string{0: "invalid", 1: "deadline"}[capacity], func(t *testing.T) {
			t.Parallel()

			_, err := New(t.Context(), nil, capacity)
			if !errors.Is(err, ErrConfig) {
				t.Fatal(err)
			}

			workspace, err := mind.New(1)
			if err != nil {
				t.Fatal(err)
			}

			_, err = New(t.Context(), workspace, 0)
			if !errors.Is(err, ErrConfig) {
				t.Fatal(err)
			}

			if capacity == 0 {
				return
			}

			supervisor, _ := newSupervisor(t, capacity)
			started := make(chan struct{})
			release := make(chan struct{})

			_, err = supervisor.Start("ignores cancellation", func(context.Context, mind.Snapshot) (mind.Proposal, error) {
				close(started)
				<-release

				return mind.Proposal{ID: "", Reads: nil, Writes: nil}, nil
			})
			if err != nil {
				t.Fatal(err)
			}

			<-started

			cancelCtx, cancel := context.WithCancel(t.Context())
			cancel()

			err = supervisor.Close(cancelCtx)

			close(release)

			if !errors.Is(err, context.Canceled) {
				t.Fatalf("shutdown did not respect caller: %v", err)
			}
		})
	}
}

func newSupervisor(t *testing.T, capacity int) (*Supervisor, *mind.Workspace) {
	t.Helper()

	workspace, err := mind.New(2)
	if err != nil {
		t.Fatal(err)
	}

	supervisor, err := New(t.Context(), workspace, capacity)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()

		err := supervisor.Close(ctx)
		if err != nil {
			t.Error(err)
		}
	})

	return supervisor, workspace
}

func next(t *testing.T, supervisor *Supervisor) Result {
	t.Helper()

	ctx, cancel := context.WithTimeout(t.Context(), testTimeout)
	defer cancel()

	result, err := supervisor.Next(ctx)
	if err != nil {
		t.Fatal(err)
	}

	return result
}

func proposal(key string) mind.Proposal {
	return mind.Proposal{ID: "assigned by runtime", Reads: map[string]uint64{key: 0}, Writes: map[string]string{key: key}}
}

func assertError(t *testing.T, got, want error) {
	t.Helper()

	if !errors.Is(got, want) {
		t.Fatalf("error = %v; want %v", got, want)
	}
}
