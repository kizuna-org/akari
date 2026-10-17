package main

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/kizuna-org/akari/internal/action"
	"github.com/kizuna-org/akari/internal/console"
	"github.com/kizuna-org/akari/internal/continuity"
	"github.com/kizuna-org/akari/internal/lifecycle"
	"github.com/kizuna-org/akari/internal/persistence"
)

type blockedReader struct {
	session  *lifecycle.Session
	started  chan struct{}
	resume   chan struct{}
	finished chan struct{}
}

func (reader blockedReader) Export() continuity.Frame {
	close(reader.started)

	<-reader.resume
	defer close(reader.finished)

	return reader.session.Export()
}

func TestManagedCloseRetainsLeaseAfterFailedConsoleDrain(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"console has an active request at deadline"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ctx, stop := context.WithCancel(t.Context())
			defer stop()

			directory := t.TempDir()
			store, err := persistence.New(directory)
			must(t, err)
			gateway, err := action.New(nil, 1)
			must(t, err)
			session, err := lifecycle.Open(ctx, store, gateway, lifecycle.FoundationConfig())
			must(t, err)

			reader := blockedReader{session: session, started: make(chan struct{}),
				resume: make(chan struct{}), finished: make(chan struct{})}
			server, err := console.Start(ctx, "127.0.0.1:0", directory, reader, true)
			must(t, err)

			service := managed{session: session, console: server}

			resume := sync.OnceFunc(func() { close(reader.resume) })

			defer func() {
				resume()

				cleanup, cancel := context.WithTimeout(context.WithoutCancel(t.Context()), time.Second)
				defer cancel()

				must(t, service.Close(cleanup))
			}()

			request, err := http.NewRequestWithContext(t.Context(), http.MethodGet,
				"http://"+server.Address()+"/api/status", nil)
			must(t, err)

			requestDone := make(chan struct{})

			go func() {
				defer close(requestDone)

				response, requestErr := http.DefaultClient.Do(request)
				if requestErr == nil {
					_ = response.Body.Close()
				}
			}()

			<-reader.started
			stop()

			deadline, cancel := context.WithCancel(t.Context())
			cancel()

			if service.Close(deadline) == nil {
				t.Fatal("console drain failure was not reported")
			}

			const leaseWait = 20 * time.Millisecond

			acquire, cancelAcquire := context.WithTimeout(t.Context(), leaseWait)
			defer cancelAcquire()

			release, err := store.Acquire(acquire)
			if err == nil {
				release()
				t.Fatal("installation lease released while workspace shutdown failed")
			}

			resume()
			<-reader.finished
			<-requestDone
			must(t, service.Close(t.Context()))
			release, err = store.Acquire(t.Context())
			must(t, err)
			release()
		})
	}
}

func must(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Fatal(err)
	}
}
