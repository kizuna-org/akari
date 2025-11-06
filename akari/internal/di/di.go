package di

import (
	"log/slog"

	"github.com/kizuna-org/akari/pkg/config"
	"github.com/kizuna-org/akari/pkg/llm/infrastructure/gemini"
	"github.com/kizuna-org/akari/pkg/llm/usecase/interactor"
	"go.uber.org/fx"
)

func NewModule() fx.Option {
	return fx.Module("akari",
		// Configuration
		fx.Provide(
			config.NewConfigRepository,
		),

		// Infrastructure
		fx.Provide(
			gemini.NewRepository,
		),

		// Usecase
		fx.Provide(
			interactor.NewLLMInteractor,
		),

		// Logger
		fx.Provide(
			slog.Default,
		),
	)
}

func NewApp() *fx.App {
	return fx.New(
		NewModule(),
		fx.NopLogger,
	)
}
