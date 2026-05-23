package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"github.com/kizuna-org/akari/internal/config"
	"go.uber.org/fx"
	"google.golang.org/protobuf/types/known/emptypb"
)

const healthProcedure = "/akari.v1.HealthService/Check"

func NewMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle(healthProcedure, connect.NewUnaryHandler(
		healthProcedure,
		func(_ context.Context, _ *connect.Request[emptypb.Empty]) (*connect.Response[emptypb.Empty], error) {
			return connect.NewResponse(&emptypb.Empty{}), nil
		},
	))

	return mux
}

func NewHTTPServer(cfg config.Config, mux *http.ServeMux) *http.Server {
	return &http.Server{
		Addr:              cfg.Addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
}

func RegisterLifecycle(lc fx.Lifecycle, server *http.Server) {
	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			go func() {
				slog.Info("http server starting", "addr", server.Addr)
				if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
					slog.Error("http server stopped unexpectedly", "error", err)
				}
			}()

			return nil
		},
		OnStop: func(ctx context.Context) error {
			if err := server.Shutdown(ctx); err != nil {
				return fmt.Errorf("shutdown http server: %w", err)
			}

			slog.Info("http server stopped")

			return nil
		},
	})
}
