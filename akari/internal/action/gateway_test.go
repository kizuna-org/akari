package action

import (
	"context"
	"errors"
	"io"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const (
	testDestination = "local"
	testTool        = "tool"
	testFile        = "file"
	testPerson      = "person"
	testTimeout     = 3 * time.Second
)

func TestConfiguration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		tools    map[string]Prepare
		capacity int
		want     error
	}{
		{name: "zero capacity", tools: nil, capacity: 0, want: ErrConfig},
		{name: "empty name", tools: map[string]Prepare{"": prepareRead}, capacity: 1, want: ErrConfig},
		{name: "nil adapter", tools: map[string]Prepare{testTool: nil}, capacity: 1, want: ErrConfig},
		{name: "no tools", tools: nil, capacity: 1, want: nil},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, err := New(test.tools, test.capacity)
			assertError(t, err, test.want)
		})
	}
}

func TestPlanning(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		impact      Impact
		destination string
		key         string
		prediction  bool
		ready       bool
		missingCall bool
		want        error
		approval    bool
	}{
		{name: "read", impact: Read, destination: testDestination, key: "", prediction: false,
			ready: false, missingCall: false, want: nil, approval: false},
		{name: "predictive read", impact: Read, destination: testDestination, key: "", prediction: true,
			ready: false, missingCall: false, want: nil, approval: false},
		{name: "predictive mutation", impact: Reversible, destination: testDestination, key: testFile, prediction: true,
			ready: false, missingCall: false, want: ErrPrediction, approval: false},
		{name: "reversible", impact: Reversible, destination: testDestination, key: testFile, prediction: false,
			ready: false, missingCall: false, want: nil, approval: false},
		{name: "ready speech", impact: Speech, destination: testDestination, key: testPerson, prediction: false,
			ready: true, missingCall: false, want: nil, approval: false},
		{name: "unready speech", impact: Speech, destination: testDestination, key: testPerson, prediction: false,
			ready: false, missingCall: false, want: nil, approval: true},
		{name: "irreversible", impact: Irreversible, destination: testDestination, key: testFile, prediction: false,
			ready: true, missingCall: false, want: nil, approval: true},
		{name: "undeclared impact", impact: "invented", destination: testDestination, key: "", prediction: false,
			ready: false, missingCall: false, want: ErrOperation, approval: false},
		{name: "missing call", impact: Read, destination: testDestination, key: "", prediction: false,
			ready: false, missingCall: true, want: ErrOperation, approval: false},
		{name: "missing destination", impact: Read, destination: "", key: "", prediction: false,
			ready: false, missingCall: false, want: ErrOperation, approval: false},
		{name: "unscoped disclosure", impact: Read, destination: "remote-search", key: "", prediction: false,
			ready: false, missingCall: false, want: ErrScope, approval: false},
		{name: "missing conflict key", impact: Reversible, destination: testDestination, key: "", prediction: false,
			ready: false, missingCall: false, want: ErrOperation, approval: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			operation := Operation{
				Impact: test.impact, Destination: test.destination, ConflictKey: test.key, Call: readCall,
			}
			if test.missingCall {
				operation.Call = nil
			}

			gateway := newGateway(t, 1, func([]byte) (Operation, error) { return operation, nil })
			scope := Scope{AllowedDestinations: []string{testDestination}, ReadyRecipients: nil}

			if test.ready {
				scope.ReadyRecipients = []string{testDestination}
			}

			pending, err := gateway.Open(test.prediction, scope).Plan(t.Context(), testTool, nil)
			assertError(t, err, test.want)

			if err == nil && pending.NeedsApproval() != test.approval {
				t.Fatal("incorrect approval requirement")
			}
		})
	}
}

func TestPlanFailures(t *testing.T) {
	t.Parallel()

	for _, reason := range []string{"canceled", "unknown", "adapter"} {
		t.Run(reason, func(t *testing.T) {
			t.Parallel()

			gateway := newGateway(t, 1, func([]byte) (Operation, error) {
				return Operation{}, io.ErrUnexpectedEOF
			})

			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			name := testTool
			want := io.ErrUnexpectedEOF

			switch reason {
			case "canceled":
				cancel()

				want = context.Canceled
			case "unknown":
				name = "missing"
				want = ErrUnknown
			}

			_, err := gateway.Open(false, localScope()).Plan(ctx, name, nil)
			assertError(t, err, want)
		})
	}
}

func TestExecutionOutcomes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		call   func(context.Context) ([]byte, error)
		want   error
		status Status
	}{
		{name: "success", call: readCall, want: nil, status: Succeeded},
		{name: "response lost", call: func(context.Context) ([]byte, error) {
			return []byte("partial"), io.ErrUnexpectedEOF
		}, want: io.ErrUnexpectedEOF, status: Unknown},
		{name: "adapter panic", call: func(context.Context) ([]byte, error) {
			panic("adapter bug")
		}, want: ErrPanic, status: Unknown},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			calls := new(atomic.Int32)
			gateway := newGateway(t, 1, func([]byte) (Operation, error) {
				return Operation{Impact: Read, Destination: testDestination, ConflictKey: "",
					Call: func(ctx context.Context) ([]byte, error) {
						calls.Add(1)

						return test.call(ctx)
					}}, nil
			})
			pending := plan(t, gateway, nil)
			observation, err := pending.Execute(t.Context())
			assertError(t, err, test.want)

			if observation.Status != test.status {
				t.Fatalf("status = %s", observation.Status)
			}

			observation, err = pending.Execute(t.Context())
			assertError(t, err, ErrUsed)

			if observation.Status != NotExecuted || calls.Load() != 1 {
				t.Fatal("one operation was dispatched more than once")
			}
		})
	}
}

func TestApprovalBinding(t *testing.T) {
	t.Parallel()

	for _, approved := range []bool{false, true} {
		t.Run(map[bool]string{false: "unapproved", true: "approved"}[approved], func(t *testing.T) {
			t.Parallel()

			gateway := newGateway(t, 1, func([]byte) (Operation, error) {
				return Operation{Impact: Irreversible, Destination: testDestination, ConflictKey: testFile, Call: readCall}, nil
			})
			pending := plan(t, gateway, nil)
			other := plan(t, gateway, nil)
			alien := newGateway(t, 1, prepareRead)
			assertError(t, gateway.Approve(nil), ErrApproval)
			assertError(t, alien.Approve(pending), ErrApproval)
			assertError(t, alien.Approve(plan(t, alien, nil)), ErrApproval)

			ctx, cancel := context.WithCancel(t.Context())
			cancel()

			_, err := pending.Execute(ctx)
			assertError(t, err, context.Canceled)

			want := error(ErrApproval)
			status := NotExecuted

			if approved {
				assertError(t, gateway.Approve(pending), nil)

				want = nil
				status = Succeeded
			}

			observation, err := pending.Execute(t.Context())
			assertError(t, err, want)

			if observation.Status != status {
				t.Fatalf("approval outcome = %s", observation.Status)
			}

			_, err = other.Execute(t.Context())
			assertError(t, err, ErrApproval)

			if approved {
				assertError(t, gateway.Approve(pending), ErrApproval)
			}
		})
	}
}

func TestOwnership(t *testing.T) {
	t.Parallel()

	for _, impact := range []Impact{Read, Speech} {
		t.Run(string(impact), func(t *testing.T) {
			t.Parallel()

			tools := map[string]Prepare{testTool: func(arguments []byte) (Operation, error) {
				return Operation{Impact: impact, Destination: testDestination, ConflictKey: testPerson,
					Call: func(context.Context) ([]byte, error) {
						return arguments, nil
					}}, nil
			}}
			gateway, err := New(tools, 1)
			assertError(t, err, nil)
			delete(tools, testTool)

			scope := Scope{AllowedDestinations: []string{testDestination}, ReadyRecipients: []string{testDestination}}
			session := gateway.Open(false, scope)
			scope.AllowedDestinations[0] = "changed"
			scope.ReadyRecipients[0] = "changed"
			arguments := []byte("original")
			pending, err := session.Plan(t.Context(), testTool, arguments)
			assertError(t, err, nil)

			arguments[0] = 'X'

			observation, err := pending.Execute(t.Context())
			assertError(t, err, nil)

			if string(observation.Data) != "original" || pending.NeedsApproval() {
				t.Fatal("caller mutation altered the frozen operation")
			}
		})
	}
}

func TestConcurrentDispatch(t *testing.T) {
	t.Parallel()

	for _, sameTarget := range []bool{false, true} {
		t.Run(map[bool]string{false: "independent", true: "conflicting"}[sameTarget], func(t *testing.T) {
			t.Parallel()

			started := make(chan string, 2)
			release := make(chan struct{})
			once := new(sync.Once)
			finish := func() { once.Do(func() { close(release) }) }
			t.Cleanup(finish)

			gateway := blockingGateway(t, 2, started, release)
			first := plan(t, gateway, []byte("first"))
			secondKey := "second"

			if sameTarget {
				secondKey = "first"
			}

			second := plan(t, gateway, []byte(secondKey))

			completed := make(chan error, 2)
			go execute(first, t.Context(), completed)

			receive(t, started)

			go execute(second, t.Context(), completed)

			if sameTarget {
				waitStats(t, gateway, Stats{Executing: 1, TargetUsers: 2})

				select {
				case <-started:
					t.Fatal("conflicting operations overlapped")
				default:
				}
			} else {
				receive(t, started)
			}

			finish()
			assertError(t, receive(t, completed), nil)
			assertError(t, receive(t, completed), nil)
			waitStats(t, gateway, Stats{Executing: 0, TargetUsers: 0})
		})
	}
}

func TestQueuedCancellation(t *testing.T) {
	t.Parallel()

	for _, sameTarget := range []bool{false, true} {
		t.Run(map[bool]string{false: "execution capacity", true: "target lock"}[sameTarget], func(t *testing.T) {
			t.Parallel()

			started := make(chan string, 2)
			release := make(chan struct{})
			once := new(sync.Once)
			finish := func() { once.Do(func() { close(release) }) }
			t.Cleanup(finish)

			gateway := blockingGateway(t, 1, started, release)
			first := plan(t, gateway, []byte("first"))
			key := "second"

			if sameTarget {
				key = "first"
			}

			second := plan(t, gateway, []byte(key))

			completed := make(chan error, 1)
			go execute(first, t.Context(), completed)

			receive(t, started)

			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			queued := make(chan error, 1)
			go execute(second, ctx, queued)

			waitStats(t, gateway, Stats{Executing: 1, TargetUsers: 2})
			cancel()
			assertError(t, receive(t, queued), context.Canceled)
			waitStats(t, gateway, Stats{Executing: 1, TargetUsers: 1})
			finish()
			assertError(t, receive(t, completed), nil)
			waitStats(t, gateway, Stats{Executing: 0, TargetUsers: 0})

			select {
			case <-started:
				t.Fatal("canceled queued operation reached the adapter")
			default:
			}
		})
	}
}

func prepareRead([]byte) (Operation, error) {
	return Operation{Impact: Read, Destination: testDestination, ConflictKey: "", Call: readCall}, nil
}

func readCall(context.Context) ([]byte, error) { return []byte("ok"), nil }

func newGateway(t *testing.T, capacity int, prepare Prepare) *Gateway {
	t.Helper()

	gateway, err := New(map[string]Prepare{testTool: prepare}, capacity)
	assertError(t, err, nil)

	return gateway
}

func localScope() Scope {
	return Scope{AllowedDestinations: []string{testDestination}, ReadyRecipients: nil}
}

func plan(t *testing.T, gateway *Gateway, arguments []byte) *Pending {
	t.Helper()

	pending, err := gateway.Open(false, localScope()).Plan(t.Context(), testTool, arguments)
	assertError(t, err, nil)

	return pending
}

func blockingGateway(t *testing.T, capacity int, started chan<- string, release <-chan struct{}) *Gateway {
	t.Helper()

	return newGateway(t, capacity, func(arguments []byte) (Operation, error) {
		key := string(arguments)

		return Operation{Impact: Reversible, Destination: testDestination, ConflictKey: key,
			Call: func(ctx context.Context) ([]byte, error) {
				started <- key

				select {
				case <-release:
					return nil, nil
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			}}, nil
	})
}

func execute(pending *Pending, ctx context.Context, completed chan<- error) {
	observation, err := pending.Execute(ctx)
	if err != nil && observation.Status != NotExecuted {
		completed <- ErrOperation

		return
	}

	completed <- err
}

func receive[T any](t *testing.T, source <-chan T) T {
	t.Helper()

	select {
	case value := <-source:
		return value
	case <-time.After(testTimeout):
		t.Fatal("operation timed out")

		return *new(T)
	}
}

func waitStats(t *testing.T, gateway *Gateway, want Stats) {
	t.Helper()

	ctx, cancel := context.WithTimeout(t.Context(), testTimeout)
	defer cancel()

	for gateway.Stats() != want {
		if ctx.Err() != nil {
			t.Fatalf("stats = %+v; want %+v", gateway.Stats(), want)
		}

		runtime.Gosched()
	}
}

func assertError(t *testing.T, got, want error) {
	t.Helper()

	if !errors.Is(got, want) {
		t.Fatalf("error = %v; want %v", got, want)
	}
}
