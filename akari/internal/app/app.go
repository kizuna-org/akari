package app

import (
	"github.com/kizuna-org/akari/internal/config"
	"github.com/kizuna-org/akari/internal/database"
	"github.com/kizuna-org/akari/internal/server"
	"go.uber.org/fx"
)

func New() *fx.App {
	return fx.New(
		fx.Provide(
			config.Load,
			database.NewClient,
			server.NewMux,
			server.NewHTTPServer,
		),
		fx.Invoke(
			database.RegisterLifecycle,
			server.RegisterLifecycle,
		),
	)
}
