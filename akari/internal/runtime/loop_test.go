//nolint:wsl,wsl_v5 // Tests keep each fake flow readable as a table case.
package runtime

import (
	"context"
	"errors"
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kizuna-org/akari/internal/action"
	"github.com/kizuna-org/akari/internal/continuity"
	"github.com/kizuna-org/akari/internal/lifecycle"
	"github.com/kizuna-org/akari/internal/mind"
	"github.com/kizuna-org/akari/internal/persistence"
	"github.com/kizuna-org/akari/internal/thought"
)

const fakeProposalID = "fake"

//nolint:cyclop,gocognit // The table covers success and invalid-input behavior through one lifecycle fixture.
func TestLoopHandleAndLifecycle(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		input   Input
		run     bool
		wantErr error
	}{
		{name: "full fake model flow", input: Input{ID: "one", Content: "hello"}, run: true, wantErr: nil},
		{name: "input identity required", input: Input{ID: "", Content: ""}, run: false, wantErr: ErrInput},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			session := newSession(t)
			var deliveredMu sync.Mutex
			var delivered []string
			loop, err := New(session, func(input Input) thought.Runner {
				return func(_ context.Context, snapshot mind.Snapshot) (mind.Proposal, error) {
					key := "interaction/" + input.ID + "/input"
					if snapshot.Items[key].Content != input.Content {
						t.Error("input missing from model snapshot")
					}
					responseKey := "interaction/" + input.ID + "/response"
					reads := map[string]uint64{key: snapshot.Items[key].Version,
						responseKey: snapshot.Items[responseKey].Version}

					return mind.Proposal{ID: fakeProposalID, Reads: reads,
						Writes: map[string]string{responseKey: "echo: " + input.Content}}, nil
				}
			}, func(_ context.Context, input Input, response string) error {
				deliveredMu.Lock()
				defer deliveredMu.Unlock()
				delivered = append(delivered, input.ID+":"+response)

				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			if test.run {
				inputs := make(chan Input, 1)
				inputs <- test.input
				close(inputs)
				err = loop.Run(t.Context(), inputs)
			} else {
				err = loop.Handle(t.Context(), test.input)
			}
			if test.wantErr != nil {
				if err == nil || err.Error() != test.wantErr.Error() {
					t.Fatalf("error = %v", err)
				}

				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(delivered) != 1 || delivered[0] != "one:echo: hello" {
				t.Fatalf("delivered = %v", delivered)
			}
			if session.Snapshot().Items["interaction/one/response"].Content != "echo: hello" {
				t.Fatal("response was sent without adoption")
			}
		})
	}
}

//nolint:noinlineerr // Each operation's error is asserted inline for clarity.
func TestPauseResumeAndClose(t *testing.T) {
	t.Parallel()
	session := newSession(t)
	started := make(chan struct{})
	var calls atomic.Int32
	loop, err := New(session, func(Input) thought.Runner {
		return func(ctx context.Context, snapshot mind.Snapshot) (mind.Proposal, error) {
			if calls.Add(1) == 1 {
				close(started)
				<-ctx.Done()

				return mind.Proposal{}, ctx.Err()
			}
			reads := map[string]uint64{
				"interaction/resume/input":    snapshot.Items["interaction/resume/input"].Version,
				"interaction/resume/response": snapshot.Items["interaction/resume/response"].Version,
			}

			return mind.Proposal{ID: fakeProposalID, Reads: reads,
				Writes: map[string]string{"interaction/resume/response": "resumed"}}, nil
		}
	}, func(_ context.Context, _ Input, response string) error {
		if response != "resumed" {
			t.Errorf("response = %q", response)
		}

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- loop.Handle(t.Context(), Input{ID: "paused", Content: "x"}) }()
	<-started
	loop.Pause()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("Handle error = %v", err)
	}

	loop.Pause()
	paused := make(chan error, 1)
	go func() { paused <- loop.Handle(t.Context(), Input{ID: "resume", Content: "x"}) }()
	loop.Resume()
	select {
	case err := <-paused:
		if err != nil {
			t.Fatalf("Handle error = %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("resume did not reopen input")
	}
	loop.Close()
	if err := loop.Handle(t.Context(), Input{ID: "closed", Content: "x"}); !errors.Is(err, lifecycle.ErrClosed) {
		t.Fatalf("Handle error = %v", err)
	}
}

//nolint:noinlineerr // The two request outcomes are the test assertions.
func TestCallerCancellationReleasesInteractionSlot(t *testing.T) {
	t.Parallel()
	session := newSession(t)
	started := make(chan struct{})
	var calls atomic.Int32
	loop, err := New(session, func(Input) thought.Runner {
		return func(ctx context.Context, snapshot mind.Snapshot) (mind.Proposal, error) {
			if calls.Add(1) == 1 {
				close(started)
				<-ctx.Done()

				return mind.Proposal{}, ctx.Err()
			}
			inputKey := "interaction/after/input"
			responseKey := "interaction/after/response"

			return mind.Proposal{ID: fakeProposalID, Reads: map[string]uint64{
				inputKey: snapshot.Items[inputKey].Version, responseKey: snapshot.Items[responseKey].Version,
			}, Writes: map[string]string{responseKey: "after cancellation"}}, nil
		}
	}, func(context.Context, Input, string) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	first := make(chan error, 1)
	go func() { first <- loop.Handle(ctx, Input{ID: "first", Content: "cancel me"}) }()
	<-started
	cancel()
	if err := <-first; !errors.Is(err, context.Canceled) {
		t.Fatalf("first Handle error = %v", err)
	}

	secondCtx, secondCancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer secondCancel()
	if err := loop.Handle(secondCtx, Input{ID: "after", Content: "continue"}); err != nil {
		t.Fatalf("second Handle after cancellation = %v", err)
	}
}

func TestRunSourceLifecycle(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		cancel bool
		want   error
	}{
		{name: "closed source", cancel: false, want: nil},
		{name: "canceled source", cancel: true, want: context.Canceled},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			loop, err := New(newSession(t), func(Input) thought.Runner {
				return func(context.Context, mind.Snapshot) (mind.Proposal, error) {
					return mind.Proposal{ID: "unused", Reads: nil, Writes: nil, Inner: nil, Prediction: false}, nil
				}
			}, func(context.Context, Input, string) error { return nil })
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			if test.cancel {
				cancel()
			}
			inputs := make(chan Input)
			if !test.cancel {
				close(inputs)
			}
			err = loop.Run(ctx, inputs)
			if !errors.Is(err, test.want) {
				t.Fatalf("Run error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestLoopFailureOutcomes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		input     string
		sinkError error
		want      error
	}{
		{name: "model failure is discarded", input: "model-error", sinkError: nil, want: io.ErrUnexpectedEOF},
		{name: "missing response is discarded", input: "missing-response", sinkError: nil, want: ErrResponse},
		{name: "sink failure follows adoption", input: "sink-error", sinkError: io.ErrClosedPipe, want: io.ErrClosedPipe},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			session := newSession(t)
			loop, err := New(session, func(input Input) thought.Runner {
				return func(_ context.Context, snapshot mind.Snapshot) (mind.Proposal, error) {
					if input.Content == "model-error" {
						return mind.Proposal{}, io.ErrUnexpectedEOF
					}
					if input.Content == "missing-response" {
						return mind.Proposal{ID: fakeProposalID, Reads: nil, Writes: nil, Inner: nil, Prediction: false}, nil
					}
					inputKey := "interaction/" + input.ID + "/input"
					responseKey := "interaction/" + input.ID + "/response"

					return mind.Proposal{ID: fakeProposalID, Reads: map[string]uint64{
						inputKey:    snapshot.Items[inputKey].Version,
						responseKey: snapshot.Items[responseKey].Version,
					}, Writes: map[string]string{responseKey: "answer"}}, nil
				}
			}, func(context.Context, Input, string) error { return test.sinkError })
			if err != nil {
				t.Fatal(err)
			}
			err = loop.Handle(t.Context(), Input{ID: test.name, Content: test.input})
			if !errors.Is(err, test.want) {
				t.Fatalf("Handle error = %v, want %v", err, test.want)
			}
			if test.name == "sink failure follows adoption" &&
				session.Snapshot().Items["interaction/"+test.name+"/response"].Content != "answer" {
				t.Fatal("sink failure reverted adopted response")
			}
		})
	}
}

func TestLoopConfigurationAndPausedCancellation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		noSession bool
		noRunner  bool
		noSink    bool
	}{
		{name: "missing session", noSession: true, noRunner: false, noSink: false},
		{name: "missing runner", noSession: false, noRunner: true, noSink: false},
		{name: "missing sink", noSession: false, noRunner: false, noSink: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			session := newSession(t)
			if test.noSession {
				session = nil
			}
			runner := RunnerFactory(func(Input) thought.Runner {
				return func(context.Context, mind.Snapshot) (mind.Proposal, error) {
					return mind.Proposal{ID: "", Reads: nil, Writes: nil, Inner: nil, Prediction: false}, nil
				}
			})
			if test.noRunner {
				runner = nil
			}
			sink := Sink(func(context.Context, Input, string) error { return nil })
			if test.noSink {
				sink = nil
			}
			_, err := New(session, runner, sink)
			if !errors.Is(err, ErrConfig) {
				t.Fatalf("New error = %v", err)
			}
		})
	}

	loop, err := New(newSession(t), func(Input) thought.Runner {
		return func(context.Context, mind.Snapshot) (mind.Proposal, error) {
			return mind.Proposal{ID: "", Reads: nil, Writes: nil, Inner: nil, Prediction: false}, nil
		}
	}, func(context.Context, Input, string) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	loop.Pause()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	err = loop.Handle(ctx, Input{ID: "waiting", Content: "x"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Handle error = %v", err)
	}
}

//nolint:noinlineerr // Cleanup reports errors through the test handle.
func newSession(t *testing.T) *lifecycle.Session {
	t.Helper()
	store, err := persistence.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	gateway, err := action.New(nil, 1)
	if err != nil {
		t.Fatal(err)
	}
	config := lifecycle.FoundationConfig()
	config.State.Inner.Clock = func() time.Time { return time.Unix(100, 0).UTC() }
	session, err := lifecycle.Open(t.Context(), store, gateway, config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := session.Close(ctx); err != nil && !errors.Is(err, continuity.ErrUnavailable) {
			t.Error(err)
		}
	})

	return session
}
