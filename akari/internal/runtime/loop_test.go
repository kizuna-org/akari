//nolint:wsl,wsl_v5,noinlineerr,exhaustruct,goconst,nlreturn // Tests keep each fake flow readable as a table case.
package runtime

import (
	"context"
	"errors"
	"fmt"
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
					key := inputKey
					if snapshot.Items[key].Content != input.Content {
						t.Error("input missing from model snapshot")
					}
					responseStateKey := responseKey
					reads := map[string]uint64{key: snapshot.Items[key].Version,
						responseStateKey: snapshot.Items[responseStateKey].Version}

					return mind.Proposal{ID: fakeProposalID, Reads: reads,
						Writes: map[string]string{responseStateKey: "echo: " + input.Content}}, nil
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
			if session.Snapshot().Items[responseKey].Content != "echo: hello" {
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
				inputKey:    snapshot.Items[inputKey].Version,
				responseKey: snapshot.Items[responseKey].Version,
			}

			return mind.Proposal{ID: fakeProposalID, Reads: reads,
				Writes: map[string]string{responseKey: "resumed"}}, nil
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

//nolint:cyclop // The gated lifecycle transitions are asserted in one deterministic interleaving.
func TestPauseCancelsBeforeResumeAdmitsNewInteraction(t *testing.T) {
	t.Parallel()

	session := newSession(t)
	firstStarted := make(chan struct{})
	secondStarted := make(chan struct{})
	resumeEntered := make(chan struct{})
	cancelEntered := make(chan struct{})
	releaseCancel := make(chan struct{})
	delivered := make(chan struct{})
	var cancelGate sync.Once
	var releaseGate sync.Once
	var calls atomic.Int32
	loop, err := New(session, func(Input) thought.Runner {
		if calls.Add(1) == 1 {
			return func(ctx context.Context, _ mind.Snapshot) (mind.Proposal, error) {
				close(firstStarted)
				<-ctx.Done()

				return mind.Proposal{}, ctx.Err()
			}
		}

		return func(_ context.Context, snapshot mind.Snapshot) (mind.Proposal, error) {
			close(secondStarted)
			return mind.Proposal{ID: fakeProposalID,
				Reads: map[string]uint64{inputKey: snapshot.Items[inputKey].Version,
					responseKey: snapshot.Items[responseKey].Version},
				Writes: map[string]string{responseKey: "resumed"}}, nil
		}
	}, func(context.Context, Input, string) error {
		close(delivered)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	loop.cancelInteraction = func() {
		cancelGate.Do(func() {
			close(cancelEntered)
			<-releaseCancel
		})
		session.Cancel("interaction")
	}
	t.Cleanup(func() { releaseGate.Do(func() { close(releaseCancel) }) })

	firstDone := make(chan error, 1)
	go func() { firstDone <- loop.Handle(t.Context(), Input{ID: "first", Content: "pause"}) }()
	<-firstStarted
	pauseDone := make(chan struct{})
	go func() {
		loop.Pause()
		close(pauseDone)
	}()
	<-cancelEntered
	resumeDone := make(chan struct{})
	go func() {
		loop.Resume()
		close(resumeEntered)
		close(resumeDone)
	}()
	secondDone := make(chan error, 1)
	go func() { secondDone <- loop.Handle(t.Context(), Input{ID: "second", Content: "resume"}) }()

	select {
	case <-resumeEntered:
		t.Fatal("Resume passed the admission lock before Pause canceled the active thought")
	default:
	}
	select {
	case <-secondStarted:
		t.Fatal("new interaction started before Pause canceled the active thought")
	default:
	}

	releaseGate.Do(func() { close(releaseCancel) })
	<-pauseDone
	<-resumeDone
	if err := <-firstDone; !errors.Is(err, context.Canceled) {
		t.Fatalf("paused interaction error = %v", err)
	}
	select {
	case err := <-secondDone:
		if err != nil {
			t.Fatalf("resumed interaction error = %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("resumed interaction did not finish")
	}
	select {
	case <-delivered:
	case <-time.After(3 * time.Second):
		t.Fatal("resumed response was not delivered")
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

func TestRunRechecksCloseAfterAwaitOpen(t *testing.T) {
	t.Parallel()
	loop, err := New(newSession(t), func(Input) thought.Runner {
		return func(context.Context, mind.Snapshot) (mind.Proposal, error) {
			return mind.Proposal{}, nil
		}
	}, func(context.Context, Input, string) error { return nil })
	if err != nil {
		t.Fatal(err)
	}

	if err := loop.awaitOpen(t.Context()); err != nil {
		t.Fatal(err)
	}
	// Force Close into the gap after awaitOpen and before Run captures changed.
	loop.Close()
	changed, closed, paused := loop.runState()
	if !closed || paused {
		t.Fatalf("captured lifecycle state = closed %t, paused %t", closed, paused)
	}
	select {
	case <-changed:
		t.Fatal("runState captured a notification channel that Close had already signaled")
	default:
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
				session.Snapshot().Items[responseKey].Content != "answer" {
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

func TestRunContinuesAfterPauseAndResume(t *testing.T) {
	t.Parallel()
	session := newSession(t)
	started := make(chan struct{})
	secondDelivered := make(chan struct{})
	var calls atomic.Int32
	loop, err := New(session, func(Input) thought.Runner {
		return func(ctx context.Context, snapshot mind.Snapshot) (mind.Proposal, error) {
			if calls.Add(1) == 1 {
				close(started)
				<-ctx.Done()

				return mind.Proposal{}, ctx.Err()
			}

			return mind.Proposal{ID: fakeProposalID,
				Reads: map[string]uint64{inputKey: snapshot.Items[inputKey].Version,
					responseKey: snapshot.Items[responseKey].Version},
				Writes: map[string]string{responseKey: "after resume"}}, nil
		}
	}, func(context.Context, Input, string) error {
		close(secondDelivered)

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	inputs := make(chan Input, 2)
	inputs <- Input{ID: "paused", Content: "first"}
	inputs <- Input{ID: "resumed", Content: "second"}
	done := make(chan error, 1)
	go func() { done <- loop.Run(t.Context(), inputs) }()
	<-started
	loop.Pause()
	loop.Resume()
	select {
	case <-secondDelivered:
	case <-time.After(3 * time.Second):
		t.Fatal("Run did not consume the next input after resume")
	}
	close(inputs)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestRunWaitsForResumeWhenPauseInterruptsRunnerFactory(t *testing.T) {
	t.Parallel()

	session := newSession(t)
	factoryStarted := make(chan struct{})
	releaseFactory := make(chan struct{})
	var factoryCalls atomic.Int32
	resumed := make(chan struct{})
	loop, err := New(session, func(Input) thought.Runner {
		if factoryCalls.Add(1) == 1 {
			close(factoryStarted)
			<-releaseFactory
		}
		return func(_ context.Context, snapshot mind.Snapshot) (mind.Proposal, error) {
			return mind.Proposal{ID: fakeProposalID,
				Reads: map[string]uint64{inputKey: snapshot.Items[inputKey].Version,
					responseKey: snapshot.Items[responseKey].Version},
				Writes: map[string]string{responseKey: "resumed"}}, nil
		}
	}, func(_ context.Context, _ Input, response string) error {
		if response == "resumed" {
			close(resumed)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	inputs := make(chan Input, 2)
	inputs <- Input{ID: "interrupted", Content: "first"}
	inputs <- Input{ID: "resumed", Content: "second"}
	done := make(chan error, 1)
	go func() { done <- loop.Run(t.Context(), inputs) }()
	<-factoryStarted
	loop.Pause()
	close(releaseFactory)
	select {
	case err := <-done:
		t.Fatalf("Run exited during pause: %v", err)
	case <-time.After(30 * time.Millisecond):
	}
	loop.Resume()
	select {
	case <-resumed:
	case <-time.After(3 * time.Second):
		t.Fatal("Run did not process input after Resume")
	}
	close(inputs)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestRunCloseWakesIdleInputSource(t *testing.T) {
	t.Parallel()
	loop, err := New(newSession(t), func(Input) thought.Runner {
		return func(context.Context, mind.Snapshot) (mind.Proposal, error) {
			return mind.Proposal{}, nil
		}
	}, func(context.Context, Input, string) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- loop.Run(t.Context(), make(chan Input)) }()
	loop.Close()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Run did not wake after Close")
	}
}

func TestRunCloseCancelsActiveThought(t *testing.T) {
	t.Parallel()
	started := make(chan struct{})
	loop, err := New(newSession(t), func(Input) thought.Runner {
		return func(ctx context.Context, _ mind.Snapshot) (mind.Proposal, error) {
			close(started)
			<-ctx.Done()
			return mind.Proposal{}, ctx.Err()
		}
	}, func(context.Context, Input, string) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	inputs := make(chan Input, 1)
	inputs <- Input{ID: "active", Content: "x"}
	done := make(chan error, 1)
	go func() { done <- loop.Run(t.Context(), inputs) }()
	<-started
	loop.Close()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Run did not stop after closing active thought")
	}
}

func TestCloseUnblocksWaitWhenRunnerIgnoresCancellation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		runnerErr error
	}{
		{name: "late result", runnerErr: nil},
		{name: "late failure", runnerErr: io.ErrClosedPipe},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			started := make(chan struct{})
			releaseRunner := make(chan struct{})
			loop, err := New(newSession(t), func(Input) thought.Runner {
				return func(_ context.Context, snapshot mind.Snapshot) (mind.Proposal, error) {
					close(started)
					<-releaseRunner
					return mind.Proposal{ID: fakeProposalID, Reads: map[string]uint64{
						inputKey: snapshot.Items[inputKey].Version, responseKey: snapshot.Items[responseKey].Version,
					}, Writes: map[string]string{responseKey: "late response"}}, test.runnerErr
				}
			}, func(context.Context, Input, string) error { return nil })
			if err != nil {
				t.Fatal(err)
			}
			done := make(chan error, 1)
			go func() { done <- loop.Handle(t.Context(), Input{ID: "blocked", Content: "x"}) }()
			<-started
			loop.Close()
			select {
			case err := <-done:
				if !errors.Is(err, lifecycle.ErrClosed) {
					t.Fatalf("Handle error = %v, want closed", err)
				}
			case <-time.After(3 * time.Second):
				t.Fatal("Handle remained blocked after Close while runner ignored cancellation")
			}
			close(releaseRunner)
		})
	}
}

func TestLoopWaitsForItsOwnCompletionAndBoundsWorkspace(t *testing.T) {
	t.Parallel()
	session := newSession(t)
	otherReady := make(chan struct{})
	other, err := session.Start("other", func(context.Context, mind.Snapshot) (mind.Proposal, error) {
		close(otherReady)
		return mind.Proposal{ID: fakeProposalID}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	<-otherReady
	loop, err := New(session, func(Input) thought.Runner {
		return func(_ context.Context, snapshot mind.Snapshot) (mind.Proposal, error) {
			return mind.Proposal{ID: fakeProposalID,
				Reads: map[string]uint64{inputKey: snapshot.Items[inputKey].Version,
					responseKey: snapshot.Items[responseKey].Version},
				Writes: map[string]string{responseKey: "answer"}}, nil
		}
	}, func(context.Context, Input, string) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	for i := range 8 {
		if err := loop.Handle(t.Context(), Input{ID: fmt.Sprintf("event-%d", i), Content: "x"}); err != nil {
			t.Fatal(err)
		}
	}
	result, err := session.Next(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if result.Token != other {
		t.Fatalf("unrelated completion token = %+v, want %+v", result.Token, other)
	}
	if err := session.Discard(t.Context(), other); err != nil {
		t.Fatal(err)
	}
	if got := len(session.Snapshot().Items); got != 2 {
		t.Fatalf("workspace items = %d, want fixed input and response keys only", got)
	}
}

func TestLoopLifecycleTransitionDuringRunnerFactoryPreventsAdmission(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		transition func(*Loop)
		want       error
	}{
		{name: "pause", transition: (*Loop).Pause, want: context.Canceled},
		{name: "close", transition: (*Loop).Close, want: lifecycle.ErrClosed},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			session := newSession(t)
			factoryStarted := make(chan struct{})
			releaseFactory := make(chan struct{})
			var runnerCalls atomic.Int32
			loop, err := New(session, func(Input) thought.Runner {
				close(factoryStarted)
				<-releaseFactory
				return func(context.Context, mind.Snapshot) (mind.Proposal, error) {
					runnerCalls.Add(1)
					return mind.Proposal{}, nil
				}
			}, func(context.Context, Input, string) error { return nil })
			if err != nil {
				t.Fatal(err)
			}
			done := make(chan error, 1)
			go func() { done <- loop.Handle(t.Context(), Input{ID: "race", Content: "x"}) }()
			<-factoryStarted
			test.transition(loop)
			close(releaseFactory)
			handleErr := <-done
			if !errors.Is(handleErr, test.want) {
				t.Fatalf("Handle error = %v, want %v", handleErr, test.want)
			}
			if test.name == "pause" && !errors.Is(handleErr, errPaused) {
				t.Fatalf("pause interruption = %v, want Run-recognized pause sentinel", handleErr)
			}
			if runnerCalls.Load() != 0 {
				t.Fatal("runner started after lifecycle transition")
			}
		})
	}
}

func TestLoopSerializationWaitObservesCancellationAndClose(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		cancel bool
		want   error
	}{
		{name: "caller cancellation", cancel: true, want: context.Canceled},
		{name: "loop close", cancel: false, want: lifecycle.ErrClosed},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			sinkStarted := make(chan struct{})
			releaseSink := make(chan struct{})
			loop, err := New(newSession(t), func(Input) thought.Runner {
				return func(_ context.Context, snapshot mind.Snapshot) (mind.Proposal, error) {
					return mind.Proposal{ID: fakeProposalID,
						Reads: map[string]uint64{inputKey: snapshot.Items[inputKey].Version,
							responseKey: snapshot.Items[responseKey].Version},
						Writes: map[string]string{responseKey: "answer"}}, nil
				}
			}, func(context.Context, Input, string) error {
				close(sinkStarted)
				<-releaseSink
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			first := make(chan error, 1)
			go func() { first <- loop.Handle(t.Context(), Input{ID: "first", Content: "x"}) }()
			<-sinkStarted
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			if !test.cancel {
				ctx = t.Context()
			}
			second := make(chan error, 1)
			go func() { second <- loop.Handle(ctx, Input{ID: "second", Content: "x"}) }()
			if test.cancel {
				cancel()
			} else {
				loop.Close()
			}
			if err := <-second; !errors.Is(err, test.want) {
				t.Fatalf("waiting Handle error = %v, want %v", err, test.want)
			}
			close(releaseSink)
			if err := <-first; err != nil {
				t.Fatal(err)
			}
		})
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
