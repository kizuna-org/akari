package server

import (
	"context"
	"encoding/json"
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

const (
	healthProcedure   = "/akari.v1.HealthService/Check"
	healthPath        = "/healthz"
	readHeaderTimeout = 5 * time.Second
)

// Persona is the running persona, as far as the server needs to know it.
//
// It is an interface so the HTTP layer does not depend on how a persona is
// built or run: all it needs is enough to say whether one is alive.
type Persona interface {
	// Name reports which persona is running.
	Name() string
	// Ticks reports how many turns it has taken.
	Ticks() int
	// Moments reports how many things it has been conscious of.
	Moments() int
}

// Health is what /healthz reports.
type Health struct {
	// Status is "ok" while the persona is running.
	Status string `json:"status"`
	// Persona names the persona.
	Persona string `json:"persona"`
	// Ticks is how many turns it has taken since waking.
	Ticks int `json:"ticks"`
	// Moments is how many things it has been conscious of.
	//
	// Zero is not a fault. A persona with nothing on its mind does nothing, and
	// will report zero moments however long it has been up
	// (docs/07-autonomy.md 7.2).
	Moments int `json:"moments"`
}

func NewMux(persona Persona) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle(healthProcedure, connect.NewUnaryHandler(
		healthProcedure,
		func(_ context.Context, _ *connect.Request[emptypb.Empty]) (*connect.Response[emptypb.Empty], error) {
			return connect.NewResponse(&emptypb.Empty{}), nil
		},
	))
	mux.HandleFunc(healthPath, healthHandler(persona))

	return mux
}

// healthHandler reports that the persona is up, and how much it has stirred.
//
// The tick count is the useful part: it is what distinguishes a persona that is
// running from one whose loop has quietly stopped.
func healthHandler(persona Persona) http.HandlerFunc {
	return func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")

		health := Health{
			Status:  "ok",
			Persona: persona.Name(),
			Ticks:   persona.Ticks(),
			Moments: persona.Moments(),
		}

		err := json.NewEncoder(writer).Encode(health)
		if err != nil {
			slog.Error("write health response", "error", err)
		}
	}
}

func NewHTTPServer(cfg config.Config, mux *http.ServeMux) *http.Server {
	server := new(http.Server)
	server.Addr = cfg.Addr
	server.Handler = mux
	server.ReadHeaderTimeout = readHeaderTimeout

	return server
}

func RegisterLifecycle(lc fx.Lifecycle, server *http.Server) {
	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			go func() {
				slog.Info("http server starting", "addr", server.Addr)

				err := server.ListenAndServe()
				if err != nil && !errors.Is(err, http.ErrServerClosed) {
					slog.Error("http server stopped unexpectedly", "error", err)
				}
			}()

			return nil
		},
		OnStop: func(ctx context.Context) error {
			err := server.Shutdown(ctx)
			if err != nil {
				return fmt.Errorf("shutdown http server: %w", err)
			}

			slog.Info("http server stopped")

			return nil
		},
	})
}
