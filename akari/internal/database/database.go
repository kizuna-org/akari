package database

import (
	"context"
	"fmt"
	"log/slog"

	"entgo.io/ent/dialect"
	"github.com/kizuna-org/akari/gen/ent"
	"github.com/kizuna-org/akari/internal/config"
	_ "github.com/lib/pq"
	"go.uber.org/fx"
)

func NewClient(cfg config.Config) (*ent.Client, error) {
	client, err := ent.Open(dialect.Postgres, cfg.Database.DSN())
	if err != nil {
		return nil, fmt.Errorf("open ent client: %w", err)
	}

	return client, nil
}

func RegisterLifecycle(lc fx.Lifecycle, client *ent.Client) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := client.Schema.Create(ctx); err != nil {
				return fmt.Errorf("create database schema: %w", err)
			}

			slog.Info("database connected")

			return nil
		},
		OnStop: func(ctx context.Context) error {
			if err := client.Close(); err != nil {
				return fmt.Errorf("close ent client: %w", err)
			}

			slog.Info("database disconnected")

			return nil
		},
	})
}
