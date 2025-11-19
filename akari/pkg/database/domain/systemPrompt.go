package domain

//go:generate go tool mockgen -package=mock -source=systemPrompt.go -destination=mock/systemPrompt.go

import (
	"context"
	"time"

	"github.com/kizuna-org/akari/gen/ent"
)

type SystemPromptRepository interface {
	GetSystemPromptByID(ctx context.Context, id int) (*SystemPrompt, error)
}

type SystemPrompt struct {
	ID        int
	Title     string
	Purpose   string
	Prompt    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func FromEntSystemPrompt(entSystemPrompt *ent.SystemPrompt) *SystemPrompt {
	if entSystemPrompt == nil {
		return nil
	}

	return &SystemPrompt{
		ID:        entSystemPrompt.ID,
		Title:     entSystemPrompt.Title,
		Purpose:   string(entSystemPrompt.Purpose),
		Prompt:    entSystemPrompt.Prompt,
		CreatedAt: entSystemPrompt.CreatedAt,
		UpdatedAt: entSystemPrompt.UpdatedAt,
	}
}
