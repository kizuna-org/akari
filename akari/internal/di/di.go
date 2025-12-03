package di

import (
	"context"
	"fmt"
	"log/slog"

	"entgo.io/ent/dialect"
	"github.com/kizuna-org/akari/gen/ent"
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
	discordService "github.com/kizuna-org/akari/pkg/discord/domain/service"
	"github.com/kizuna-org/akari/pkg/discord/handler"
	discordInfra "github.com/kizuna-org/akari/pkg/discord/infrastructure"
	discordInteractor "github.com/kizuna-org/akari/pkg/discord/usecase/interactor"
	"github.com/kizuna-org/akari/pkg/llm/infrastructure/gemini"
	llmInteractor "github.com/kizuna-org/akari/pkg/llm/usecase/interactor"
	"go.uber.org/fx"
)

const (
	defaultCharacterID = 1
	defaultPromptIndex = 0
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
		newDiscordMessageRepository,
		newDiscordClient,
	)
}

func newUsecaseProviders() fx.Option {
	return fx.Provide(
		llmInteractor.NewLLMInteractor,
		databaseInteractor.NewDatabaseInteractor,
		databaseInteractor.NewSystemPromptInteractor,
		databaseInteractor.NewCharacterInteractor,
		databaseInteractor.NewConversationInteractor,
		databaseInteractor.NewConversationGroupInteractor,
		databaseInteractor.NewDiscordUserInteractor,
		databaseInteractor.NewAkariUserInteractor,
		discordRepository.NewDiscordRepository,
	)
}

func newMessagePackageProviders() fx.Option {
	return fx.Options(
		fx.Provide(
			messageAdapter.NewMessageRepository,
			messageAdapter.NewResponseRepository,
			messageAdapter.NewLLMRepository,
			messageAdapter.NewDiscordRepository,
			messageAdapter.NewValidationRepository,
			messageAdapter.NewCharacterRepository,
			messageAdapter.NewSystemPromptRepository,
			messageAdapter.NewConversationRepository,
			messageAdapter.NewConversationGroupRepository,
			messageAdapter.NewDiscordUserRepository,
			messageAdapter.NewAkariUserRepository,
			newHandleMessageInteractor,
		),
		fx.Supply(
			defaultCharacterID,
			defaultPromptIndex,
		),
	)
}

func newHandleMessageInteractor(
	messageRepo messageDomain.MessageRepository,
	responseRepo messageDomain.ResponseRepository,
	llmRepo messageDomain.LLMRepository,
	discordRepo messageDomain.DiscordRepository,
	validationRepo messageDomain.ValidationRepository,
	characterRepo messageDomain.CharacterRepository,
	systemPromptRepo messageDomain.SystemPromptRepository,
	conversationRepo messageDomain.ConversationRepository,
	conversationGroupRepo messageDomain.ConversationGroupRepository,
	discordUserRepo messageDomain.DiscordUserRepository,
	defaultCharacterID int,
	defaultPromptIndex int,
	client *discordInfra.DiscordClient,
	configRepo config.ConfigRepository,
) messageUsecase.HandleMessageInteractor {
	cfg := configRepo.GetConfig()

	return messageUsecase.NewHandleMessageInteractor(
		messageUsecase.HandleMessageConfig{
			MessageRepo:           messageRepo,
			ResponseRepo:          responseRepo,
			LLMRepo:               llmRepo,
			DiscordRepo:           discordRepo,
			ValidationRepo:        validationRepo,
			CharacterRepo:         characterRepo,
			SystemPromptRepo:      systemPromptRepo,
			ConversationRepo:      conversationRepo,
			ConversationGroupRepo: conversationGroupRepo,
			DiscordUserRepo:       discordUserRepo,
			DefaultCharacterID:    defaultCharacterID,
			DefaultPromptIndex:    defaultPromptIndex,
			BotUserID:             client.Session.State.User.ID,
			BotNamePattern:        cfg.Discord.BotNameRegExp,
		},
	)
}

func newServiceAndInteractorProviders() fx.Option {
	return fx.Provide(
		discordService.NewDiscordService,
		discordInteractor.NewDiscordInteractor,
		handler.NewMessageHandler,
		discordAdapter.NewBotRunner,
		slog.Default,
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

func newSystemPromptRepository(repo databaseRepo.Repository) databaseDomain.SystemPromptRepository {
	return repo
}

func newCharacterRepository(repo databaseRepo.Repository) databaseDomain.CharacterRepository {
	return repo
}

func newDiscordMessageRepository(repo databaseRepo.Repository) databaseDomain.DiscordMessageRepository {
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
