package repository

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/kizuna-org/akari/pkg/database/domain"
	"github.com/kizuna-org/akari/pkg/database/infrastructure"
)

type Repository interface {
	domain.DatabaseRepository
	domain.AkariUserRepository
	domain.CharacterRepository
	domain.ConversationRepository
	domain.ConversationGroupRepository
	domain.DiscordUserRepository
	domain.DiscordMessageRepository
	domain.DiscordChannelRepository
	domain.DiscordGuildRepository
	domain.SystemPromptRepository
	HealthCheck(ctx context.Context) error
}

type repositoryImpl struct {
	client infrastructure.Client
	logger *slog.Logger
}

func NewRepository(client infrastructure.Client, logger *slog.Logger) Repository {
	return &repositoryImpl{
		client: client,
		logger: logger.With("component", "database_repository"),
	}
}

func (r *repositoryImpl) HealthCheck(ctx context.Context) error {
	if err := r.client.Ping(ctx); err != nil {
		return fmt.Errorf("database health check failed: %w", err)
	}

	return nil
}

func (r *repositoryImpl) WithTransaction(ctx context.Context, fn domain.TxFunc) error {
	return r.client.WithTx(ctx, fn)
}
