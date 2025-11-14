package di

import (
	"context"
	"fmt"
	"log/slog"

	"entgo.io/ent/dialect"
	"github.com/kizuna-org/akari/gen/ent"
	internalDiscordUsecase "github.com/kizuna-org/akari/internal/app/usecase/discord"
	"github.com/kizuna-org/akari/pkg/config"
	databaseDomain "github.com/kizuna-org/akari/pkg/database/domain"
	"github.com/kizuna-org/akari/pkg/database/infrastructure/postgres"
	databaseInteractor "github.com/kizuna-org/akari/pkg/database/usecase/interactor"
	discordRepository "github.com/kizuna-org/akari/pkg/discord/adapter/repository"
	discordService "github.com/kizuna-org/akari/pkg/discord/domain/service"
	"github.com/kizuna-org/akari/pkg/discord/handler"
	discordInfra "github.com/kizuna-org/akari/pkg/discord/infrastructure"
	discordInteractor "github.com/kizuna-org/akari/pkg/discord/usecase/interactor"
	"github.com/kizuna-org/akari/pkg/llm/infrastructure/gemini"
	llmInteractor "github.com/kizuna-org/akari/pkg/llm/usecase/interactor"
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
			newEntClient,
			gemini.NewRepository,
			newPostgresClient,
			postgres.NewRepository,
			newDatabaseRepository,
			newSystemPromptRepository,
			newCharacterRepository,
			newDiscordClient,
		),

		// Usecase
		fx.Provide(
			llmInteractor.NewLLMInteractor,
			databaseInteractor.NewDatabaseInteractor,
			databaseInteractor.NewSystemPromptInteractor,
			databaseInteractor.NewCharacterInteractor,
			discordRepository.NewDiscordRepository,
			internalDiscordUsecase.NewDiscordMessageUsecase,
		),

		// Service
		fx.Provide(
			discordService.NewDiscordService,
		),

		// Interactor
		fx.Provide(
			discordInteractor.NewDiscordInteractor,
		),

		fx.Provide(
			handler.NewMessageHandler,
		),

		fx.Provide(
			slog.Default,
		),

		// Lifecycle hooks
		fx.Invoke(registerDatabaseHooks),
	)
}

func newEntClient(configRepo config.ConfigRepository) (*ent.Client, error) {
	cfg := configRepo.GetConfig()

	return ent.Open(dialect.Postgres, cfg.Database.BuildDSN())
}

func registerDatabaseHooks(
	lc fx.Lifecycle,
	client postgres.Client,
	repository postgres.Repository,
	logger *slog.Logger,
) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("verifying database connection")
			if err := repository.HealthCheck(ctx); err != nil {
				return fmt.Errorf("database health check failed: %w", err)
			}
			logger.Info("database connection verified successfully")

			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("disconnecting from database")
			if err := client.Close(); err != nil {
				return fmt.Errorf("failed to disconnect from database: %w", err)
			}
			logger.Info("database disconnected successfully")

			return nil
		},
	})
}

func newPostgresClient(configRepo config.ConfigRepository, logger *slog.Logger) (postgres.Client, error) {
	cfg := configRepo.GetConfig().Database

	client, err := postgres.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create database client: %w", err)
	}

	logger.Info("database client created",
		slog.String("host", cfg.Host),
		slog.Int("port", cfg.Port),
		slog.String("database", cfg.Database),
	)

	return client, nil
}

func newDatabaseRepository(repo postgres.Repository) databaseDomain.DatabaseRepository {
	return repo
}

func newSystemPromptRepository(repo postgres.Repository) databaseDomain.SystemPromptRepository {
	return repo
}

func newCharacterRepository(repo postgres.Repository) databaseDomain.CharacterRepository {
	return repo
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
