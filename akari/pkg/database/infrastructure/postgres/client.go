package postgres

import (
	"context"
	"fmt"

	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql"
	"github.com/kizuna-org/akari/gen/ent"
	"github.com/kizuna-org/akari/pkg/database/domain"
	_ "github.com/lib/pq"
)

type Client interface {
	Ping(ctx context.Context) error
	Close() error
	WithTx(ctx context.Context, txFunc domain.TxFunc) error
	SystemPromptClient() *ent.SystemPromptClient
}

type client struct {
	*ent.Client
	driver *sql.Driver
}

func NewClient(cfg Config) (Client, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host,
		cfg.Port,
		cfg.User,
		cfg.Password,
		cfg.Database,
		cfg.SSLMode,
	)

	drv, err := sql.Open(dialect.Postgres, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	database := drv.DB()
	if cfg.MaxOpenConns > 0 {
		database.SetMaxOpenConns(cfg.MaxOpenConns)
	}

	if cfg.MaxIdleConns > 0 {
		database.SetMaxIdleConns(cfg.MaxIdleConns)
	}

	if cfg.ConnMaxLifetime > 0 {
		database.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	}

	if cfg.ConnMaxIdleTime > 0 {
		database.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)
	}

	opts := []ent.Option{ent.Driver(drv)}
	if cfg.Debug {
		opts = append(opts, ent.Debug())
	}

	return &client{Client: ent.NewClient(opts...), driver: drv}, nil
}

func (c *client) Ping(ctx context.Context) error {
	return c.driver.DB().PingContext(ctx)
}

func (c *client) Close() error {
	return c.Client.Close()
}

func (c *client) SystemPromptClient() *ent.SystemPromptClient {
	return c.SystemPrompt
}
