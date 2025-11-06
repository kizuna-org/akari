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

func (r *repositoryImpl) GetClient() domain.Client {
	return r.client
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
