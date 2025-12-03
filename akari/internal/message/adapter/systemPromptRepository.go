package adapter

import (
	"context"
	"fmt"

	"github.com/kizuna-org/akari/internal/message/domain"
	databaseInteractor "github.com/kizuna-org/akari/pkg/database/usecase/interactor"
)

type systemPromptRepository struct {
	interactor databaseInteractor.SystemPromptInteractor
}

func NewSystemPromptRepository(interactor databaseInteractor.SystemPromptInteractor) domain.SystemPromptRepository {
	return &systemPromptRepository{
		interactor: interactor,
	}
}

func (r *systemPromptRepository) GetSystemPromptByID(ctx context.Context, id int) (*domain.SystemPrompt, error) {
	systemPrompt, err := r.interactor.GetSystemPromptByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get system prompt by id: %w", err)
	}

	return &domain.SystemPrompt{
		ID:     systemPrompt.ID,
		Prompt: systemPrompt.Prompt,
	}, nil
}
