package adapter

import (
	"context"
	"fmt"

	"github.com/kizuna-org/akari/internal/message/domain"
	databaseDomain "github.com/kizuna-org/akari/pkg/database/domain"
)

type systemPromptRepository struct {
	repository databaseDomain.SystemPromptRepository
}

func NewSystemPromptRepository(repository databaseDomain.SystemPromptRepository) domain.SystemPromptRepository {
	return &systemPromptRepository{
		repository: repository,
	}
}

func (r *systemPromptRepository) Get(ctx context.Context, id int) (*domain.SystemPrompt, error) {
	systemPrompt, err := r.repository.GetSystemPromptByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get system prompt by id: %w", err)
	}

	return &domain.SystemPrompt{
		ID:     systemPrompt.ID,
		Prompt: systemPrompt.Prompt,
	}, nil
}
