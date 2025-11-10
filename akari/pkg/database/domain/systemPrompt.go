package domain

import (
	"context"

	"github.com/kizuna-org/akari/gen/ent"
	"github.com/kizuna-org/akari/gen/ent/systemprompt"
)

type SystemPromptRepository interface {
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

type (
	SystemPrompt        = ent.SystemPrompt
	SystemPromptPurpose = systemprompt.Purpose
)
