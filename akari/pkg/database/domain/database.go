package domain

import (
	"context"

	"github.com/kizuna-org/akari/gen/ent"
	"github.com/kizuna-org/akari/gen/ent/systemprompt"
)

type DatabaseRepository interface {
	Close() error
	HealthCheck(ctx context.Context) error
	WithTransaction(ctx context.Context, fn TxFunc) error
	CreateSystemPrompt(ctx context.Context, title, prompt string, purpose SystemPromptPurpose) (*SystemPrompt, error)
	GetSystemPromptByID(ctx context.Context, id int) (*SystemPrompt, error)
	UpdateSystemPrompt(
		ctx context.Context,
		id int,
		title, prompt *string,
		purpose *SystemPromptPurpose,
	) (*SystemPrompt, error)
	DeleteSystemPrompt(ctx context.Context, id int) error
}

type Client interface {
	Unwrap() *ent.Client
	Ping(ctx context.Context) error
	Close() error
	WithTx(ctx context.Context, fn TxFunc) error
}

type (
	Tx                  = ent.Tx
	SystemPrompt        = ent.SystemPrompt
	SystemPromptPurpose = systemprompt.Purpose
)

type TxFunc func(ctx context.Context, tx *Tx) error
