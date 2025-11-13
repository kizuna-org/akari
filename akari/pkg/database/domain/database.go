package domain

//go:generate go tool mockgen -package=mock -source=database.go -destination=mock/database.go

import (
	"context"

	"github.com/kizuna-org/akari/gen/ent"
)

type DatabaseRepository interface {
	WithTransaction(ctx context.Context, fn TxFunc) error
}

type (
	Tx = ent.Tx
)

type TxFunc func(ctx context.Context, tx *Tx) error
