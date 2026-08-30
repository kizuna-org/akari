package app

import (
	"github.com/kizuna-org/akari/internal/config"
	"github.com/kizuna-org/akari/internal/database"
	"github.com/kizuna-org/akari/internal/loop"
	"github.com/kizuna-org/akari/internal/persona"
	"github.com/kizuna-org/akari/internal/server"
	"go.uber.org/fx"
)

// runningPersona lets the HTTP server report on the running persona without
// depending on how one is built or run. Joining the two is this package's job,
// because this is where the program is composed.
func runningPersona(runner *loop.Loop) server.Persona {
	return runner
}

// Options is everything the program is made of.
//
// It is kept separate from New so the wiring can be checked without starting
// anything. A dependency graph that does not resolve is a failure at boot, and
// in a container that means a crash loop rather than a test failure.
func Options() fx.Option {
	return fx.Options(
		fx.Provide(
			config.Load,
			database.NewClient,
			persona.New,
			persona.NewLoop,
			runningPersona,
			server.NewMux,
			server.NewHTTPServer,
		),
		fx.Invoke(
			database.RegisterLifecycle,
			persona.RegisterLifecycle,
			server.RegisterLifecycle,
		),
	)
}

// New builds the application.
func New() *fx.App {
	return fx.New(Options())
}
