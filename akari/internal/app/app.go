package app

import (
	"github.com/kizuna-org/akari/internal/config"
	"github.com/kizuna-org/akari/internal/database"
	"github.com/kizuna-org/akari/internal/persona"
	"github.com/kizuna-org/akari/internal/server"
	"go.uber.org/fx"
)

func New() *fx.App {
	return fx.New(
		fx.Provide(
			config.Load,
			database.NewClient,
			persona.New,
			persona.NewLoop,
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
