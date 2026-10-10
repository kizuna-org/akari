package host

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

const healthPath = "/healthz"

func TestNewHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		method string
		path   string
		status int
		body   string
	}{
		{
			name: "liveness is explicitly foundation", method: http.MethodGet,
			path: healthPath, status: http.StatusOK, body: healthBody,
		},
		{
			name: "head supports container health probe", method: http.MethodHead,
			path: healthPath, status: http.StatusOK, body: healthBody,
		},
		{
			name: "does not claim readiness", method: http.MethodGet,
			path: "/readyz", status: http.StatusNotFound, body: "404 page not found\n",
		},
		{
			name: "no accidental management endpoint", method: http.MethodGet,
			path: "/admin", status: http.StatusNotFound, body: "404 page not found\n",
		},
		{
			name: "rejects mutation", method: http.MethodPost,
			path: healthPath, status: http.StatusMethodNotAllowed, body: "Method Not Allowed\n",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			request := httptest.NewRequestWithContext(t.Context(), test.method, test.path, nil)
			recorder := httptest.NewRecorder()
			NewHandler().ServeHTTP(recorder, request)

			if recorder.Code != test.status || recorder.Body.String() != test.body {
				t.Fatalf("response = %d %q; want %d %q", recorder.Code, recorder.Body.String(), test.status, test.body)
			}

			if test.status == http.StatusOK && recorder.Header().Get("Cache-Control") != "no-store" {
				t.Fatal("liveness must not be cached")
			}
		})
	}
}

func TestRun(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		address   string
		cancel    bool
		wantError bool
	}{
		{name: "invalid address fails startup", address: "127.0.0.1:invalid", cancel: false, wantError: true},
		{name: "canceled host closes listener", address: "127.0.0.1:0", cancel: true, wantError: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			if test.cancel {
				cancel()
			}

			err := Run(ctx, test.address)
			if (err != nil) != test.wantError {
				t.Fatalf("Run error = %v; wantError = %v", err, test.wantError)
			}
		})
	}
}

func TestServe(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		acceptError bool
		closeError  bool
		wantError   bool
	}{
		{name: "health and graceful shutdown", acceptError: false, closeError: false, wantError: false},
		{name: "listener failure is returned", acceptError: true, closeError: false, wantError: true},
		{name: "shutdown failure is returned", acceptError: false, closeError: true, wantError: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			listenConfig := new(net.ListenConfig)

			listener, err := listenConfig.Listen(t.Context(), "tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}

			t.Cleanup(func() { _ = listener.Close() })

			wrapped := &testListener{Listener: listener, acceptError: test.acceptError, closeError: test.closeError}

			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			done := make(chan error, 1)

			go func() { done <- serve(ctx, wrapped) }()

			if !test.acceptError {
				checkHealthy(t, listener.Addr().String())
				cancel()
			}

			select {
			case serveErr := <-done:
				if (serveErr != nil) != test.wantError {
					t.Fatalf("serve error = %v; wantError = %v", serveErr, test.wantError)
				}
			case <-t.Context().Done():
				t.Fatal("host did not terminate")
			}

			// Serve owns the listener on both normal and failed paths.
			_, acceptErr := listener.Accept()
			if !errors.Is(acceptErr, net.ErrClosed) {
				t.Fatalf("listener was not closed: %v", acceptErr)
			}
		})
	}
}

func checkHealthy(t *testing.T, address string) {
	t.Helper()

	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://"+address+healthPath, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Avoid inheriting proxy routing for this local listener.
	transport := new(http.Transport)
	t.Cleanup(transport.CloseIdleConnections)

	client := new(http.Client)
	client.Transport = transport

	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}

	body, readErr := io.ReadAll(response.Body)
	_ = response.Body.Close()

	if readErr != nil || response.StatusCode != http.StatusOK || string(body) != healthBody {
		t.Fatalf("health = %d %q, error = %v", response.StatusCode, body, readErr)
	}
}

type testListener struct {
	net.Listener

	acceptError bool
	closeError  bool
}

func (listener *testListener) Accept() (net.Conn, error) {
	if listener.acceptError {
		return nil, io.ErrUnexpectedEOF
	}

	return listener.Listener.Accept()
}

func (listener *testListener) Close() error {
	closeErr := listener.Listener.Close()
	if listener.closeError {
		return errors.Join(closeErr, io.ErrClosedPipe)
	}

	return closeErr
}
