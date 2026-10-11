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

// Run binds before reporting success and serves until cancellation or failure.
func Run(ctx context.Context, address string) error {
	listenConfig := new(net.ListenConfig)

	listener, err := listenConfig.Listen(ctx, "tcp", address)
	if err != nil {
		return fmt.Errorf("bind foundation host: %w", err)
	}

	slog.Info("foundation host listening", "address", listener.Addr().String(), "mode", "foundation")

	return serve(ctx, listener)
}

func serve(ctx context.Context, listener net.Listener) error {
	server := new(http.Server)
	server.Handler = NewHandler()
	server.ReadHeaderTimeout = headerTimeout
	server.WriteTimeout = headerTimeout
	server.IdleTimeout = headerTimeout

	done := make(chan error, 1)

	go func() {
		done <- server.Serve(listener)
	}()

	select {
	case err := <-done:
		return fmt.Errorf("serve foundation host: %w", err)
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownTimeout)
	defer cancel()

	shutdownErr := server.Shutdown(shutdownCtx)
	closeErr := server.Close()
	serveErr := <-done

	if errors.Is(serveErr, http.ErrServerClosed) {
		serveErr = nil
	}

	err := errors.Join(shutdownErr, closeErr, serveErr)
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
