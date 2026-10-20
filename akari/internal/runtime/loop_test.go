//nolint:wsl,wsl_v5,noinlineerr,exhaustruct,goconst,nlreturn // Tests keep each fake flow readable as a table case.
package runtime

import (
	"context"
	"encoding/json"
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
	"github.com/kizuna-org/akari/internal/memory"
	"github.com/kizuna-org/akari/internal/mind"
	"github.com/kizuna-org/akari/internal/persistence"
	"github.com/kizuna-org/akari/internal/thought"
)

const fakeProposalID = "fake"
const testDeliveryTool = "runtime.test_delivery"

type testSink func(context.Context, Input, string) error

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
			var deliveredMu sync.Mutex
			var delivered []string
			session := newSession(t, func(_ context.Context, input Input, response string) error {
				deliveredMu.Lock()
				defer deliveredMu.Unlock()
				delivered = append(delivered, input.ID+":"+response)
				return nil
			})
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
			}, testDeliveryIntent)
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

func TestLoopDeliveryChecksTrustedSourcesBeforeDispatch(t *testing.T) {
	t.Parallel()

	var dispatched atomic.Int32
	session := newSession(t, func(context.Context, Input, string) error {
		dispatched.Add(1)
		return nil
	})
	loop, err := New(session, func(Input) thought.Runner {
		return func(_ context.Context, snapshot mind.Snapshot) (mind.Proposal, error) {
			return mind.Proposal{ID: fakeProposalID,
				Reads: map[string]uint64{inputKey: snapshot.Items[inputKey].Version,
					responseKey: snapshot.Items[responseKey].Version},
				Writes: map[string]string{responseKey: "private response"}}, nil
		}
	}, testDeliveryIntent)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := session.Snapshot()
	_, err = session.Adopt(t.Context(), mind.Proposal{ID: "private-agreement", Reads: nil, Writes: nil,
		Inner: &mind.Update{Expected: snapshot.Experience.Versions,
			Memory: &memory.Change{Context: nil, Working: nil, Remember: nil, Recalled: nil,
				Agreements: []memory.Agreement{{Conversation: "private-memory", Recipients: nil}}},
			Goals: nil, Interests: nil, Appraisal: nil, Load: 0, Rest: 0, Advance: false}, Prediction: false}, nil)
	if err != nil {
		t.Fatal(err)
	}

	err = loop.Handle(t.Context(), Input{ID: "private", Content: "private context", Sources: []string{"private-memory"}})
	if !errors.Is(err, action.ErrDisclosure) {
		t.Fatalf("Handle error = %v, want disclosure rejection", err)
	}
	if dispatched.Load() != 0 {
		t.Fatal("private response reached the delivery adapter")
	}
}

func TestSessionCloseDrainsBlockedResponseDelivery(t *testing.T) {
	t.Parallel()

	deliveryStarted := make(chan struct{})
	deliveryCanceled := make(chan struct{})
	releaseDelivery := make(chan struct{})
	session := newSession(t, func(ctx context.Context, _ Input, _ string) error {
		close(deliveryStarted)
		<-ctx.Done()
		close(deliveryCanceled)
		<-releaseDelivery
		return nil
	})
	loop, err := New(session, func(input Input) thought.Runner {
		return func(_ context.Context, snapshot mind.Snapshot) (mind.Proposal, error) {
			return mind.Proposal{ID: fakeProposalID, Reads: map[string]uint64{
				inputKey: snapshot.Items[inputKey].Version, responseKey: snapshot.Items[responseKey].Version,
			}, Writes: map[string]string{responseKey: input.ID}}, nil
		}
	}, testDeliveryIntent)
	if err != nil {
		t.Fatal(err)
	}

	handleDone := make(chan error, 1)
	go func() { handleDone <- loop.Handle(t.Context(), Input{ID: "blocked", Content: "hello"}) }()
	<-deliveryStarted
	closeDone := make(chan error, 1)
	go func() { closeDone <- session.Close(context.Background()) }()
	<-deliveryCanceled
	select {
	case err := <-closeDone:
		t.Fatalf("Session.Close returned before delivery drained: %v", err)
	case <-time.After(30 * time.Millisecond):
	}

	close(releaseDelivery)
	if err := <-handleDone; err != nil {
		t.Fatal(err)
	}
	if err := <-closeDone; err != nil {
		t.Fatal(err)
	}
}

//nolint:noinlineerr // Each operation's error is asserted inline for clarity.
func TestPauseResumeAndClose(t *testing.T) {
	t.Parallel()
	started := make(chan struct{})
	var calls atomic.Int32
	session := newSession(t, func(_ context.Context, _ Input, response string) error {
		if response != "resumed" {
			t.Errorf("response = %q", response)
		}
		return nil
	})
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
	}, testDeliveryIntent)
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

	firstStarted := make(chan struct{})
	secondStarted := make(chan struct{})
	resumeEntered := make(chan struct{})
	cancelEntered := make(chan struct{})
	releaseCancel := make(chan struct{})
	delivered := make(chan struct{})
	session := newSession(t, func(context.Context, Input, string) error {
		close(delivered)
		return nil
	})
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
	}, testDeliveryIntent)
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
	}, testDeliveryIntent)
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
			}, testDeliveryIntent)
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
	}, testDeliveryIntent)
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
			session := newSession(t, func(context.Context, Input, string) error { return test.sinkError })
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
			}, testDeliveryIntent)
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
		name       string
		noSession  bool
		noRunner   bool
		noDelivery bool
	}{
		{name: "missing session", noSession: true, noRunner: false, noDelivery: false},
		{name: "missing runner", noSession: false, noRunner: true, noDelivery: false},
		{name: "missing delivery planner", noSession: false, noRunner: false, noDelivery: true},
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
			sink := DeliveryPlanner(testDeliveryIntent)
			if test.noDelivery {
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
	}, testDeliveryIntent)
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
	started := make(chan struct{})
	secondDelivered := make(chan struct{})
	session := newSession(t, func(context.Context, Input, string) error {
		close(secondDelivered)
		return nil
	})
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
	}, testDeliveryIntent)
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

	factoryStarted := make(chan struct{})
	releaseFactory := make(chan struct{})
	var factoryCalls atomic.Int32
	delivered := make(chan string, 2)
	session := newSession(t, func(_ context.Context, _ Input, response string) error {
		delivered <- response
		return nil
	})
	loop, err := New(session, func(input Input) thought.Runner {
		if factoryCalls.Add(1) == 1 {
			close(factoryStarted)
			<-releaseFactory
		}
		return func(_ context.Context, snapshot mind.Snapshot) (mind.Proposal, error) {
			return mind.Proposal{ID: fakeProposalID,
				Reads: map[string]uint64{inputKey: snapshot.Items[inputKey].Version,
					responseKey: snapshot.Items[responseKey].Version},
				Writes: map[string]string{responseKey: input.ID}}, nil
		}
	}, testDeliveryIntent)
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
	loop.Resume()
	close(releaseFactory)
	select {
	case response := <-delivered:
		if response != "resumed" {
			t.Fatalf("delivered interrupted response %q", response)
		}
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
	}, testDeliveryIntent)
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
	}, testDeliveryIntent)
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
			}, testDeliveryIntent)
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
	}, testDeliveryIntent)
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
		cancel     bool
		want       error
	}{
		{name: "pause", transition: (*Loop).Pause, want: context.Canceled},
		{name: "close", transition: (*Loop).Close, want: lifecycle.ErrClosed},
		{name: "caller cancellation", cancel: true, want: context.Canceled},
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
			}, testDeliveryIntent)
			if err != nil {
				t.Fatal(err)
			}
			done := make(chan error, 1)
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			go func() { done <- loop.Handle(ctx, Input{ID: "race", Content: "x"}) }()
			<-factoryStarted
			if test.cancel {
				cancel()
			} else {
				test.transition(loop)
			}
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

func TestCleanupWaitHandlesRepeatedLifecycleNotificationsIteratively(t *testing.T) {
	t.Parallel()

	session := newSession(t)
	runnerStarted := make(chan struct{})
	releaseRunner := make(chan struct{})
	var calls atomic.Int32
	loop, err := New(session, func(Input) thought.Runner {
		if calls.Add(1) == 1 {
			return func(context.Context, mind.Snapshot) (mind.Proposal, error) {
				close(runnerStarted)
				<-releaseRunner
				return mind.Proposal{}, nil
			}
		}
		return func(_ context.Context, snapshot mind.Snapshot) (mind.Proposal, error) {
			return mind.Proposal{ID: fakeProposalID,
				Reads: map[string]uint64{inputKey: snapshot.Items[inputKey].Version,
					responseKey: snapshot.Items[responseKey].Version},
				Writes: map[string]string{responseKey: "after cleanup"}}, nil
		}
	}, testDeliveryIntent)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	first := make(chan error, 1)
	go func() { first <- loop.Handle(ctx, Input{ID: "first", Content: "cancel"}) }()
	<-runnerStarted
	cancel()
	if err := <-first; !errors.Is(err, context.Canceled) {
		t.Fatalf("first Handle error = %v", err)
	}
	second := make(chan error, 1)
	go func() { second <- loop.Handle(t.Context(), Input{ID: "second", Content: "continue"}) }()
	for range 1000 {
		loop.Resume()
	}
	close(releaseRunner)
	select {
	case err := <-second:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("second Handle did not resume after cleanup")
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
			session := newSession(t, func(context.Context, Input, string) error {
				close(sinkStarted)
				<-releaseSink
				return nil
			})
			loop, err := New(session, func(Input) thought.Runner {
				return func(_ context.Context, snapshot mind.Snapshot) (mind.Proposal, error) {
					return mind.Proposal{ID: fakeProposalID,
						Reads: map[string]uint64{inputKey: snapshot.Items[inputKey].Version,
							responseKey: snapshot.Items[responseKey].Version},
						Writes: map[string]string{responseKey: "answer"}}, nil
				}
			}, testDeliveryIntent)
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
func newSession(t *testing.T, sinks ...testSink) *lifecycle.Session {
	t.Helper()
	sink := testSink(func(context.Context, Input, string) error { return nil })
	if len(sinks) > 0 {
		sink = sinks[0]
	}
	store, err := persistence.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	tools := map[string]action.Prepare{
		testDeliveryTool: func(raw []byte) (action.Operation, error) {
			var arguments struct {
				InputID      string `json:"inputId"`
				InputContent string `json:"inputContent"`
				Destination  string `json:"destination"`
				Response     string `json:"response"`
			}
			if err := json.Unmarshal(raw, &arguments); err != nil {
				return action.Operation{}, err
			}
			return action.Operation{Impact: action.Speech, Destination: arguments.Destination,
				ConflictKey: arguments.Destination,
				Call: func(ctx context.Context) ([]byte, error) {
					return nil, sink(ctx, Input{ID: arguments.InputID, Content: arguments.InputContent}, arguments.Response)
				}, CallWithKey: nil, Sources: nil}, nil
		},
	}
	gateway, err := action.New(tools, 1)
	if err != nil {
		t.Fatal(err)
	}
	config := lifecycle.FoundationConfig()
	config.State.Inner.Clock = func() time.Time { return time.Unix(100, 0).UTC() }
	config.State.Scope = func(mind.Snapshot) action.Scope {
		return action.Scope{AllowedDestinations: []string{"test:recipient"}, ReadyRecipients: []string{"test:recipient"}}
	}
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

func testDeliveryIntent(input Input, response string) (continuity.Intent, error) {
	arguments, err := json.Marshal(map[string]string{
		"inputId": input.ID, "inputContent": input.Content, "destination": "test:recipient", "response": response,
	})
	if err != nil {
		return continuity.Intent{}, err
	}

	return continuity.Intent{ID: "interaction/response/" + input.ID, Tool: testDeliveryTool,
		Arguments: arguments, Sources: append([]string{}, input.Sources...)}, nil
}
