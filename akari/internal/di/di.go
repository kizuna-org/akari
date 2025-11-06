package di

import (
	"log/slog"

	"github.com/kizuna-org/akari/pkg/config"
	discordRepository "github.com/kizuna-org/akari/pkg/discord/adapter/repository"
	discordService "github.com/kizuna-org/akari/pkg/discord/domain/service"
	"github.com/kizuna-org/akari/pkg/discord/handler"
	discordInfra "github.com/kizuna-org/akari/pkg/discord/infrastructure"
	discordInteractor "github.com/kizuna-org/akari/pkg/discord/usecase/interactor"
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
            newDiscordClient,
        ),

        // Repository
        fx.Provide(
            discordRepository.NewDiscordRepository,
        ),

        // Service
        fx.Provide(
            discordService.NewDiscordService,
		),

		// Usecase
		fx.Provide(
			interactor.NewLLMInteractor,
			discordInteractor.NewDiscordInteractor,
		),

		// Handler
		fx.Provide(
			handler.NewMessageHandler,
		),

		// Logger
		fx.Provide(
			slog.Default,
		),
	)
}

func newDiscordClient(configRepo config.ConfigRepository) (*discordInfra.DiscordClient, error) {
	cfg := configRepo.GetConfig()
	return discordInfra.NewDiscordClient(cfg.Discord.Token)
}

func NewApp() *fx.App {
	return fx.New(
		NewModule(),
		fx.NopLogger,
	)
}
