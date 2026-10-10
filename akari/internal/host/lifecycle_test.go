package host

import (
	"context"
	"errors"
	"io"
	"net"
	"sync/atomic"
	"testing"
)

type serviceFunc func(context.Context) error

func (service serviceFunc) Close(ctx context.Context) error { return service(ctx) }

func TestRunWithRestorationFailure(t *testing.T) {
	t.Parallel()

	for _, failure := range []error{io.ErrUnexpectedEOF, context.Canceled} {
		t.Run(failure.Error(), func(t *testing.T) {
			t.Parallel()

			listener, err := new(net.ListenConfig).Listen(t.Context(), "tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}

			address := listener.Addr().String()
			_ = listener.Close()

			err = RunWith(t.Context(), address, func(context.Context) (Service, error) { return nil, failure })
			if !errors.Is(err, failure) {
				t.Fatal("restoration failure was hidden")
			}

			listener, err = new(net.ListenConfig).Listen(t.Context(), "tcp", address)
			if err != nil {
				t.Fatal("failed startup leaked its listener")
			}

			_ = listener.Close()
		})
	}
}

func TestRunWithRestoredService(t *testing.T) {
	t.Parallel()

	for _, failure := range []error{nil, io.ErrClosedPipe} {
		name := "clean"
		if failure != nil {
			name = failure.Error()
		}

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			var closes atomic.Int32

			err := RunWith(ctx, "127.0.0.1:0", func(context.Context) (Service, error) {
				cancel()

				return serviceFunc(func(context.Context) error {
					closes.Add(1)

					return failure
				}), nil
			})
			if !errors.Is(err, failure) || closes.Load() != 1 {
				t.Fatal("restored service was not finalized exactly once")
			}
		})
	}
}

func TestHostFinalizesStateBeforeClosingListener(t *testing.T) {
	t.Parallel()

	for _, failure := range []bool{false, true} {
		t.Run(map[bool]string{false: "clean", true: "save failed"}[failure], func(t *testing.T) {
			t.Parallel()

			listener, err := new(net.ListenConfig).Listen(t.Context(), "tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}

			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			var finalized atomic.Bool

			service := serviceFunc(func(ctx context.Context) error {
				if ctx.Err() != nil {
					t.Fatal("finalization inherited the signal cancellation")
				}

				checkHealthyContext(t, ctx, listener.Addr().String())
				finalized.Store(true)

				if failure {
					return io.ErrClosedPipe
				}

				return nil
			})
			done := make(chan error, 1)

			go func() { done <- serveManaged(ctx, listener, service) }()

			checkHealthy(t, listener.Addr().String())
			cancel()

			err = <-done
			if !finalized.Load() || (err != nil) != failure {
				t.Fatal("state finalization was skipped or its failure hidden")
			}
		})
	}
}
