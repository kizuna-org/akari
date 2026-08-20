// Package persona provides the persona and its running loop to the application.
//
// It is the seam between the domain packages, which know nothing about how the
// program is wired, and the fx graph that starts and stops things.
package persona

import (
	"context"
	"log/slog"

	"github.com/kizuna-org/akari/internal/config"
	"github.com/kizuna-org/akari/internal/loop"
	"github.com/kizuna-org/akari/internal/memory"
	"github.com/kizuna-org/akari/internal/mind"
	"github.com/kizuna-org/akari/internal/workspace"
	"go.uber.org/fx"
)

// New builds the persona the program runs.
//
// Its attention wanders a little: the chooser is weighted rather than
// strongest-wins, so the persona does not always attend to the largest number.
// The seed comes from configuration so a run can be replayed exactly.
func New(cfg config.Config) *mind.Mind {
	settings := mind.DefaultPersona(cfg.Persona.Name)

	return mind.New(
		settings,
		workspace.NewWeightedChooser(cfg.Persona.Seed),
		memory.TokenOverlap{},
	)
}

// NewLoop builds the loop that keeps the persona running.
func NewLoop(persona *mind.Mind, cfg config.Config) *loop.Loop {
	return loop.New(
		persona,
		loop.SystemClock{},
		loop.Config{Interval: cfg.Persona.Interval, Nightly: cfg.Persona.Nightly},
		slog.Default(),
	)
}

// RegisterLifecycle starts the persona with the program and rests it on the way
// down.
//
// The loop runs in its own goroutine because it is not serving anything: it
// keeps going whether or not a request ever arrives, which is the point of an
// always-on persona (docs/07-autonomy.md 7.1).
func RegisterLifecycle(lc fx.Lifecycle, runner *loop.Loop) {
	ctx, rest := context.WithCancel(context.Background())

	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			go runner.Run(ctx)

			return nil
		},
		OnStop: func(context.Context) error {
			rest()
			slog.Info("persona stopped", "ticks", runner.Ticks(), "moments", runner.Moments())

			return nil
		},
	})
}
