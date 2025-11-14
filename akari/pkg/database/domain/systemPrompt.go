package domain

//go:generate go tool mockgen -package=mock -source=systemPrompt.go -destination=mock/systemPrompt.go

import (
	"context"

	"github.com/kizuna-org/akari/gen/ent"
	"github.com/kizuna-org/akari/gen/ent/systemprompt"
)

type SystemPromptRepository interface {
	GetSystemPromptByID(ctx context.Context, id int) (*SystemPrompt, error)
}

type (
	SystemPrompt        = ent.SystemPrompt
	SystemPromptPurpose = systemprompt.Purpose
)
