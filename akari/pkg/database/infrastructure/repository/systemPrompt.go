package repository

import (
	"context"
	"fmt"

	"github.com/kizuna-org/akari/pkg/database/domain"
)

func (r *repositoryImpl) GetSystemPromptByID(ctx context.Context, id int) (*domain.SystemPrompt, error) {
	systemPrompt, err := r.client.SystemPromptClient().Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get system prompt by id: %w", err)
	}

	return domain.FromEntSystemPrompt(systemPrompt), nil
}
