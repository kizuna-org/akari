package postgres

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/kizuna-org/akari/pkg/config"
	"github.com/kizuna-org/akari/pkg/database/domain"
)

type repositoryImpl struct {
	client domain.Client
	logger *slog.Logger
}

func NewRepository(cfg config.ConfigRepository, logger *slog.Logger) (domain.DatabaseRepository, error) {
	config := NewConfig(cfg.GetConfig())

	client, err := NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create database client: %w", err)
	}

	logger.Info("database client created",
		slog.String("host", config.Host),
		slog.Int("port", config.Port),
		slog.String("database", config.Database),
	)

	return &repositoryImpl{
		client: client,
		logger: logger.With("component", "postgres_repository"),
	}, nil
}

func (r *repositoryImpl) Connect(ctx context.Context) error {
	if err := r.client.Ping(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	r.logger.Info("database connection verified")

	return nil
}

func (r *repositoryImpl) Disconnect() error {
	if err := r.client.Close(); err != nil {
		return fmt.Errorf("failed to close database connection: %w", err)
	}

	r.logger.Info("database connection closed")

	return nil
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

func (r *repositoryImpl) CreateSystemPrompt(
	ctx context.Context,
	title, prompt string,
	purpose domain.SystemPromptPurpose,
) (*domain.SystemPrompt, error) {
	systemPrompt, err := r.client.Unwrap().SystemPrompt.
		Create().
		SetTitle(title).
		SetPrompt(prompt).
		SetPurpose(purpose).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create system prompt: %w", err)
	}

	r.logger.Info("system prompt created",
		slog.Int("id", systemPrompt.ID),
		slog.String("title", title),
		slog.String("purpose", string(purpose)),
		slog.String("prompt", prompt),
	)

	return systemPrompt, nil
}

func (r *repositoryImpl) GetSystemPromptByID(ctx context.Context, id int) (*domain.SystemPrompt, error) {
	systemPrompt, err := r.client.Unwrap().SystemPrompt.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get system prompt by id: %w", err)
	}

	return systemPrompt, nil
}

func (r *repositoryImpl) UpdateSystemPrompt(
	ctx context.Context,
	promptID int,
	title, prompt *string,
	purpose *domain.SystemPromptPurpose,
) (*domain.SystemPrompt, error) {
	updater := r.client.Unwrap().SystemPrompt.UpdateOneID(promptID)
	if title != nil {
		updater = updater.SetTitle(*title)
	}

	if purpose != nil {
		updater = updater.SetPurpose(*purpose)
	}

	if prompt != nil {
		updater = updater.SetPrompt(*prompt)
	}

	systemPrompt, err := updater.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to update system prompt: %w", err)
	}

	r.logger.Info("system prompt updated",
		slog.Int("id", promptID),
		slog.String("title", *title),
		slog.String("purpose", string(*purpose)),
		slog.String("prompt", *prompt),
	)

	return systemPrompt, nil
}

func (r *repositoryImpl) DeleteSystemPrompt(ctx context.Context, promptID int) error {
	if err := r.client.Unwrap().SystemPrompt.DeleteOneID(promptID).Exec(ctx); err != nil {
		return fmt.Errorf("failed to delete system prompt: %w", err)
	}

	r.logger.Info("system prompt deleted",
		slog.Int("id", promptID),
	)

	return nil
}
