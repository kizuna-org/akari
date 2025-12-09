package interactor

//go:generate go tool mockgen -package=mock -source=systemPrompt.go -destination=mock/systemPrompt.go

import (
	"context"

	"github.com/kizuna-org/akari/pkg/database/domain"
)

type SystemPromptInteractor interface {
	GetSystemPromptByID(ctx context.Context, id int) (*domain.SystemPrompt, error)
}

type systemPromptInteractorImpl struct {
	repository domain.SystemPromptRepository
}

func NewSystemPromptInteractor(repository domain.SystemPromptRepository) SystemPromptInteractor {
	return &systemPromptInteractorImpl{
		repository: repository,
	}
}

func (d *systemPromptInteractorImpl) GetSystemPromptByID(
	ctx context.Context,
	id int,
) (*domain.SystemPrompt, error) {
	return d.repository.GetSystemPromptByID(ctx, id)
}
