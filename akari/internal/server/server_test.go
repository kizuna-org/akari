package server

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/kizuna-org/akari/internal/config"
	"go.uber.org/fx/fxtest"
	"google.golang.org/protobuf/types/known/emptypb"
)

const (
	testPersona = "akari"
	testTicks   = 7
	testMoments = 3
	blockPath   = "/block"
)

var errWriteFailed = errors.New("client hung up")

// stubPersona stands in for a running persona.
type stubPersona struct {
	name    string
	ticks   int
	moments int
}

func (s stubPersona) Name() string { return s.name }
func (s stubPersona) Ticks() int   { return s.ticks }
func (s stubPersona) Moments() int { return s.moments }

func testMux() *http.ServeMux {
	return NewMux(stubPersona{name: testPersona, ticks: testTicks, moments: testMoments})
}

func testConfig(addr string) config.Config {
	return config.Config{
		Addr:     addr,
		Database: config.Database{Host: "", Port: 0, User: "", Password: "", Name: "", SSLMode: ""},
		Persona:  config.Persona{Name: "", Seed: 0, Interval: 0, Nightly: 0},
	}
}

// freeAddr reserves an address and releases it, so a server can be told exactly
// where to listen.
func freeAddr(t *testing.T) string {
	t.Helper()

	var listenConfig net.ListenConfig

	listener, err := listenConfig.Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen() error = %v", err)
	}

	addr := listener.Addr().String()

	err = listener.Close()
	if err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	return addr
}

// fetch performs a GET and returns the status and body, closing the response.
func fetch(t *testing.T, client *http.Client, url string) (int, []byte, error) {
	t.Helper()

	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
	if err != nil {
		return 0, nil, err
	}

	response, err := client.Do(request)
	if err != nil {
		return 0, nil, err
	}

	defer func() {
		err := response.Body.Close()
		if err != nil {
			t.Errorf("Close() error = %v", err)
		}
	}()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return 0, nil, err
	}

	return response.StatusCode, body, nil
}

func TestNewMux(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		path string
		want int
	}{
		{name: "health check", path: healthProcedure, want: http.StatusOK},
		{name: "unknown route", path: "/unknown", want: http.StatusNotFound},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(testMux())
			t.Cleanup(server.Close)

			client := connect.NewClient[emptypb.Empty, emptypb.Empty](
				server.Client(),
				server.URL+testCase.path,
			)

			_, err := client.CallUnary(t.Context(), connect.NewRequest(&emptypb.Empty{}))
			if testCase.want == http.StatusOK && err != nil {
				t.Fatalf("CallUnary() error = %v", err)
			}

			if testCase.want == http.StatusNotFound && err == nil {
				t.Fatal("CallUnary() error = nil, want error")
			}
		})
	}
}

func TestHealthEndpoint(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(testMux())
	t.Cleanup(server.Close)

	status, body, err := fetch(t, server.Client(), server.URL+healthPath)
	if err != nil {
		t.Fatalf("fetch() error = %v", err)
	}

	if status != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", status, http.StatusOK)
	}

	var health Health

	err = json.Unmarshal(body, &health)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	want := Health{Status: "ok", Persona: testPersona, Ticks: testTicks, Moments: testMoments}
	if health != want {
		t.Fatalf("health = %#v, want %#v", health, want)
	}
}

// TestHealthReportsAnIdlePersonaAsWell guards the thing most likely to be
// misread: a persona with nothing on its mind is healthy, not broken.
func TestHealthReportsAnIdlePersonaAsWell(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(NewMux(stubPersona{name: testPersona, ticks: 42, moments: 0}))
	t.Cleanup(server.Close)

	_, body, err := fetch(t, server.Client(), server.URL+healthPath)
	if err != nil {
		t.Fatalf("fetch() error = %v", err)
	}

	var health Health

	err = json.Unmarshal(body, &health)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if health.Status != "ok" {
		t.Fatalf("Status = %q, want ok even with nothing on its mind", health.Status)
	}

	if health.Ticks == 0 {
		t.Fatal("Ticks = 0, want the tick count that proves the loop is turning")
	}
}

// failingWriter accepts headers but refuses to write a body, standing in for a
// client that hung up mid-response.
type failingWriter struct {
	header http.Header
}

func (f *failingWriter) Header() http.Header {
	if f.header == nil {
		f.header = make(http.Header)
	}

	return f.header
}

func (f *failingWriter) Write([]byte) (int, error) { return 0, errWriteFailed }

func (f *failingWriter) WriteHeader(int) {}

func TestHealthHandlerSurvivesAWriteFailure(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, healthPath, nil)
	handler := healthHandler(stubPersona{name: testPersona, ticks: testTicks, moments: testMoments})

	// A response that cannot be written is logged and shrugged off, not fatal.
	handler(&failingWriter{header: nil}, request)
}

func TestNewHTTPServer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		addr string
	}{
		{name: "the configured address is used", addr: ":9999"},
		{name: "an empty address is left to the standard library", addr: ""},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			server := NewHTTPServer(testConfig(testCase.addr), testMux())
			if server.Addr != testCase.addr {
				t.Fatalf("Addr = %q, want %q", server.Addr, testCase.addr)
			}

			if server.ReadHeaderTimeout != readHeaderTimeout {
				t.Fatalf("ReadHeaderTimeout = %v, want %v", server.ReadHeaderTimeout, readHeaderTimeout)
			}

			if server.Handler == nil {
				t.Fatal("Handler = nil, want the mux")
			}
		})
	}
}

func TestRegisterLifecycleServesAndStops(t *testing.T) {
	t.Parallel()

	addr := freeAddr(t)
	lifecycle := fxtest.NewLifecycle(t)

	RegisterLifecycle(lifecycle, NewHTTPServer(testConfig(addr), testMux()))
	lifecycle.RequireStart()

	waitForListening(t, addr)

	// The persona is reachable while the lifecycle is running.
	status, _, err := fetch(t, http.DefaultClient, "http://"+addr+healthPath)
	if err != nil {
		lifecycle.RequireStop()
		t.Fatalf("the server never answered: %v", err)
	}

	if status != http.StatusOK {
		lifecycle.RequireStop()
		t.Fatalf("StatusCode = %d, want %d", status, http.StatusOK)
	}

	lifecycle.RequireStop()

	// And unreachable once it has stopped.
	_, _, err = fetch(t, http.DefaultClient, "http://"+addr+healthPath)
	if err == nil {
		t.Fatal("the server still answered after stopping, want it shut down")
	}
}

// signallingHandler closes a channel the first time it sees a record at or
// above the given level.
type signallingHandler struct {
	fired chan struct{}
	level slog.Level
	once  sync.Once
}

func (h *signallingHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

func (h *signallingHandler) Handle(context.Context, slog.Record) error {
	h.once.Do(func() { close(h.fired) })

	return nil
}

func (h *signallingHandler) WithAttrs([]slog.Attr) slog.Handler { return h }

func (h *signallingHandler) WithGroup(string) slog.Handler { return h }

// TestRegisterLifecycleReportsAServerThatCannotListen covers the case that
// matters in a container: the port is already taken, so the server never comes
// up. It must say so rather than fail silently.
//
//nolint:paralleltest // swaps the default logger, so it cannot run beside others
func TestRegisterLifecycleReportsAServerThatCannotListen(t *testing.T) {
	var listenConfig net.ListenConfig

	occupied, err := listenConfig.Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen() error = %v", err)
	}

	t.Cleanup(func() {
		err := occupied.Close()
		if err != nil {
			t.Errorf("Close() error = %v", err)
		}
	})

	logged := make(chan struct{})
	previous := slog.Default()

	slog.SetDefault(slog.New(&signallingHandler{
		fired: logged,
		level: slog.LevelError,
		once:  sync.Once{},
	}))
	t.Cleanup(func() { slog.SetDefault(previous) })

	lifecycle := fxtest.NewLifecycle(t)

	RegisterLifecycle(lifecycle, NewHTTPServer(testConfig(occupied.Addr().String()), testMux()))
	lifecycle.RequireStart()

	select {
	case <-logged:
	case <-time.After(2 * time.Second):
		lifecycle.RequireStop()
		t.Fatal("nothing was logged, want a server that cannot listen to say so")
	}

	lifecycle.RequireStop()
}

// TestRegisterLifecycleReportsAShutdownThatCouldNotFinish covers the other way
// stopping can go wrong: a request is still being served when the time allowed
// for shutting down has run out.
func TestRegisterLifecycleReportsAShutdownThatCouldNotFinish(t *testing.T) {
	t.Parallel()

	addr := freeAddr(t)
	serving := make(chan struct{})
	release := make(chan struct{})

	mux := http.NewServeMux()
	mux.HandleFunc(blockPath, func(http.ResponseWriter, *http.Request) {
		close(serving)
		<-release
	})

	server := NewHTTPServer(testConfig(addr), mux)
	lifecycle := fxtest.NewLifecycle(t)

	RegisterLifecycle(lifecycle, server)
	lifecycle.RequireStart()

	t.Cleanup(func() {
		close(release)

		err := server.Close()
		if err != nil {
			t.Errorf("Close() error = %v", err)
		}
	})

	waitForListening(t, addr)

	go func() {
		_, _, _ = fetch(t, http.DefaultClient, "http://"+addr+blockPath)
	}()

	select {
	case <-serving:
	case <-time.After(2 * time.Second):
		t.Fatal("the request never reached the handler")
	}

	// A request is still in flight, and shutting down is given almost no time to
	// wait for it. The deadline has to be real rather than already spent, or the
	// lifecycle would give up before ever reaching the hook.
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := lifecycle.Stop(ctx)
	if err == nil {
		t.Fatal("Stop() error = nil, want a shutdown that ran out of time to say so")
	}
}

// waitForListening blocks until something accepts connections on the address,
// so a test does not race the server's own goroutine getting started.
func waitForListening(t *testing.T, addr string) {
	t.Helper()

	var dialer net.Dialer

	dialer.Timeout = 100 * time.Millisecond

	for range 200 {
		conn, err := dialer.DialContext(t.Context(), "tcp", addr)
		if err == nil {
			err = conn.Close()
			if err != nil {
				t.Errorf("Close() error = %v", err)
			}

			return
		}

		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("nothing is listening on %s", addr)
}
