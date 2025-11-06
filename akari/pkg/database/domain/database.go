package domain

import (
	"context"

	"github.com/kizuna-org/akari/gen/ent"
)

type DatabaseRepository interface {
	GetClient() Client
	Connect(ctx context.Context) error
	Disconnect() error
	HealthCheck(ctx context.Context) error
}

type Client interface {
	Unwrap() *ent.Client
	Ping(ctx context.Context) error
	Close() error
	WithTx(ctx context.Context, fn TxFunc) error
}

type (
	Tx = ent.Tx
)

type TxFunc func(ctx context.Context, tx *Tx) error
