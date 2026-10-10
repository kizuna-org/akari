// Package host provides process liveness. It does not implement Akari's mind.
package host

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"
)

const (
	headerTimeout   = 5 * time.Second
	shutdownTimeout = 5 * time.Second
	healthBody      = "{\"status\":\"alive\",\"mode\":\"foundation\"}\n"
)

type Service interface {
	Close(ctx context.Context) error
}

type Starter func(context.Context) (Service, error)

// Run binds before reporting success and serves until cancellation or failure.
func Run(ctx context.Context, address string) error {
	return RunWith(ctx, address, nil)
}

// RunWith restores a service before serving liveness, and finalizes it before closing the listener.
func RunWith(ctx context.Context, address string, start Starter) error {
	listenConfig := new(net.ListenConfig)

	listener, err := listenConfig.Listen(ctx, "tcp", address)
	if err != nil {
		return fmt.Errorf("bind foundation host: %w", err)
	}

	var service Service

	if start != nil {
		service, err = start(ctx)
		if err != nil {
			_ = listener.Close()

			return fmt.Errorf("restore foundation state: %w", err)
		}
	}

	slog.Info("foundation host listening", "address", listener.Addr().String(), "mode", "foundation")

	return serveManaged(ctx, listener, service)
}

func serve(ctx context.Context, listener net.Listener) error {
	return serveManaged(ctx, listener, nil)
}

func serveManaged(ctx context.Context, listener net.Listener, service Service) error {
	server := new(http.Server)
	server.Handler = NewHandler()
	server.ReadHeaderTimeout = headerTimeout
	server.WriteTimeout = headerTimeout
	server.IdleTimeout = headerTimeout

	done := make(chan error, 1)

	go func() {
		done <- server.Serve(listener)
	}()

	var serveErr error

	finished := false

	select {
	case serveErr = <-done:
		finished = true
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownTimeout)
	defer cancel()

	var stateErr error

	if service != nil {
		stateErr = service.Close(shutdownCtx)
	}

	shutdownErr := server.Shutdown(shutdownCtx)
	closeErr := server.Close()

	if !finished {
		serveErr = <-done
	}

	if errors.Is(serveErr, http.ErrServerClosed) {
		serveErr = nil
	}

	err := errors.Join(stateErr, shutdownErr, closeErr, serveErr)
	if err != nil {
		return fmt.Errorf("stop foundation host: %w", err)
	}

	return nil
}

// NewHandler reports host liveness without claiming mind readiness.
func NewHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writer.Header().Set("Cache-Control", "no-store")

		_, _ = writer.Write([]byte(healthBody))
	})

	return mux
}
