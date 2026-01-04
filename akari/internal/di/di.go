package di

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"time"

	"entgo.io/ent/dialect"
	"github.com/kizuna-org/akari/gen/ent"
	"github.com/kizuna-org/akari/internal/infrastructure/logger"
	messageAdapter "github.com/kizuna-org/akari/internal/message/adapter"
	messageDomain "github.com/kizuna-org/akari/internal/message/domain"
	messageUsecase "github.com/kizuna-org/akari/internal/message/usecase"
	"github.com/kizuna-org/akari/pkg/config"
	databaseDomain "github.com/kizuna-org/akari/pkg/database/domain"
	databaseInfra "github.com/kizuna-org/akari/pkg/database/infrastructure"
	databaseRepo "github.com/kizuna-org/akari/pkg/database/infrastructure/repository"
	databaseInteractor "github.com/kizuna-org/akari/pkg/database/usecase/interactor"
	discordAdapter "github.com/kizuna-org/akari/pkg/discord/adapter"
	discordRepository "github.com/kizuna-org/akari/pkg/discord/adapter/repository"
	discordDomain "github.com/kizuna-org/akari/pkg/discord/domain/repository"
	discordService "github.com/kizuna-org/akari/pkg/discord/domain/service"
	"github.com/kizuna-org/akari/pkg/discord/handler"
	discordInfra "github.com/kizuna-org/akari/pkg/discord/infrastructure"
	discordInteractor "github.com/kizuna-org/akari/pkg/discord/usecase/interactor"
	"github.com/kizuna-org/akari/pkg/llm/infrastructure/gemini"
	llmInteractor "github.com/kizuna-org/akari/pkg/llm/usecase/interactor"
	"go.uber.org/fx"
)

type (
	defaultCharacterID int
	defaultPromptIndex int
)

const (
	characterIDValue int = 1
	promptIndexValue int = 0
)

func newInfrastructureProviders() fx.Option {
	return fx.Provide(
		newEntClient,
		gemini.NewRepository,
		newDatabaseClient,
		databaseRepo.NewRepository,
		newDatabaseRepository,
		newSystemPromptRepository,
		newCharacterRepository,
		newAkariUserRepository,
		newDiscordUserRepository,
		newDiscordMessageRepository,
		newDiscordChannelRepository,
		newDiscordGuildRepository,
		newDiscordClient,
	)
}

func newUsecaseProviders() fx.Option {
	return fx.Provide(
		databaseInteractor.NewDatabaseInteractor,
		databaseInteractor.NewCharacterInteractor,
		databaseInteractor.NewAkariUserInteractor,
		databaseInteractor.NewDiscordUserInteractor,
		databaseInteractor.NewDiscordMessageInteractor,
		databaseInteractor.NewDiscordChannelInteractor,
		databaseInteractor.NewDiscordGuildInteractor,
		newDiscordRepository,
		llmInteractor.NewLLMInteractor,
		databaseInteractor.NewSystemPromptInteractor,
	)
}

func newMessagePackageProviders() fx.Option {
	return fx.Options(
		fx.Provide(
			messageAdapter.NewCharacterRepository,
			messageAdapter.NewDiscordRepository,
			messageAdapter.NewDiscordUserRepository,
			messageAdapter.NewDiscordMessageRepository,
			messageAdapter.NewDiscordChannelRepository,
			messageAdapter.NewDiscordGuildRepository,
			messageAdapter.NewLLMRepository,
			messageAdapter.NewSystemPromptRepository,
			messageAdapter.NewValidationRepository,
			newHandleMessageInteractor,
		),
		fx.Supply(
			defaultCharacterID(characterIDValue),
			defaultPromptIndex(promptIndexValue),
		),
	)
}

func newHandleMessageInteractor(
	characterRepo messageDomain.CharacterRepository,
	discordRepo messageDomain.DiscordRepository,
	discordUserRepo messageDomain.DiscordUserRepository,
	discordMessageRepo messageDomain.DiscordMessageRepository,
	discordChannelRepo messageDomain.DiscordChannelRepository,
	discordGuildRepo messageDomain.DiscordGuildRepository,
	llmRepo messageDomain.LLMRepository,
	systemPromptRepo messageDomain.SystemPromptRepository,
	validationRepo messageDomain.ValidationRepository,
	characterID defaultCharacterID,
	promptIdx defaultPromptIndex,
	configRepo config.ConfigRepository,
) (discordService.HandleMessageInteractor, error) {
	cfg := configRepo.GetConfig()

	botNameRegex, err := regexp.Compile(cfg.Discord.BotNameRegExp)
	if err != nil {
		return nil, fmt.Errorf("di: invalid bot name regex pattern: %w", err)
	}

	return messageUsecase.NewHandleMessageInteractor(
		messageUsecase.HandleMessageConfig{
			CharacterRepo:       characterRepo,
			DiscordRepo:         discordRepo,
			DiscordUserRepo:     discordUserRepo,
			DiscordMessageRepo:  discordMessageRepo,
			DiscordChannelRepo:  discordChannelRepo,
			DiscordGuildRepo:    discordGuildRepo,
			LLMRepo:             llmRepo,
			SystemPromptRepo:    systemPromptRepo,
			ValidationRepo:      validationRepo,
			DefaultCharacterID:  int(characterID),
			DefaultPromptIndex:  int(promptIdx),
			BotNamePatternRegex: botNameRegex,
		},
	), nil
}

func newServiceAndInteractorProviders() fx.Option {
	return fx.Provide(
		discordService.NewDiscordService,
		discordInteractor.NewDiscordInteractor,
		handler.NewMessageHandler,
		discordAdapter.NewBotRunner,
		logger.NewLogger,
	)
}

func NewModule() fx.Option {
	return fx.Module("akari",
		// Configuration
		fx.Provide(
			config.NewConfigRepository,
		),

		// Infrastructure
		newInfrastructureProviders(),

		// Usecase
		newUsecaseProviders(),

		// Message Package
		newMessagePackageProviders(),

		// Service and Interactor
		newServiceAndInteractorProviders(),

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
	client databaseInfra.Client,
	repository databaseRepo.Repository,
	logger *slog.Logger,
) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("Verifying database connection")
			if err := repository.HealthCheck(ctx); err != nil {
				return fmt.Errorf("database health check failed: %w", err)
			}
			logger.Info("Database connection verified successfully")

			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Disconnecting from database")
			if err := client.Close(); err != nil {
				return fmt.Errorf("failed to disconnect from database: %w", err)
			}
			logger.Info("Database disconnected successfully")

			return nil
		},
	})
}

func newDatabaseClient(configRepo config.ConfigRepository, logger *slog.Logger) (databaseInfra.Client, error) {
	cfg := configRepo.GetConfig().Database

	client, err := databaseInfra.NewClient(cfg)
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

func newDatabaseRepository(repo databaseRepo.Repository) databaseDomain.DatabaseRepository {
	return repo
}

func newCharacterRepository(repo databaseRepo.Repository) databaseDomain.CharacterRepository {
	return repo
}

func newDiscordRepository(
	client *discordInfra.DiscordClient,
	configRepo config.ConfigRepository,
) discordDomain.DiscordRepository {
	cfg := configRepo.GetConfig()

	return discordRepository.NewDiscordRepository(client, time.Duration(cfg.Discord.ReadyTimeout)*time.Second)
}

func newSystemPromptRepository(repo databaseRepo.Repository) databaseDomain.SystemPromptRepository {
	return repo
}

func newDiscordUserRepository(repo databaseRepo.Repository) databaseDomain.DiscordUserRepository {
	return repo
}

func newDiscordMessageRepository(repo databaseRepo.Repository) databaseDomain.DiscordMessageRepository {
	return repo
}

func newDiscordChannelRepository(repo databaseRepo.Repository) databaseDomain.DiscordChannelRepository {
	return repo
}

func newDiscordGuildRepository(repo databaseRepo.Repository) databaseDomain.DiscordGuildRepository {
	return repo
}

func newAkariUserRepository(repo databaseRepo.Repository) databaseDomain.AkariUserRepository {
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
