package interactor

import (
	"context"

	"github.com/kizuna-org/akari/pkg/database/domain"
)

type SystemPromptInteractor interface {
	CreateSystemPrompt(
		ctx context.Context,
		title, prompt string,
		purpose domain.SystemPromptPurpose,
	) (*domain.SystemPrompt, error)
	GetSystemPromptByID(ctx context.Context, id int) (*domain.SystemPrompt, error)
	UpdateSystemPrompt(
		ctx context.Context,
		id int,
		title, prompt *string,
		purpose *domain.SystemPromptPurpose,
	) (*domain.SystemPrompt, error)
	DeleteSystemPrompt(ctx context.Context, id int) error
}

type systemPromptInteractorImpl struct {
	repository domain.SystemPromptRepository
}

func NewSystemPromptInteractor(repository domain.SystemPromptRepository) SystemPromptInteractor {
	return &systemPromptInteractorImpl{
		repository: repository,
	}
}

func (d *systemPromptInteractorImpl) CreateSystemPrompt(
	ctx context.Context,
	title, prompt string,
	purpose domain.SystemPromptPurpose,
) (*domain.SystemPrompt, error) {
	return d.repository.CreateSystemPrompt(ctx, title, prompt, purpose)
}

func (d *systemPromptInteractorImpl) GetSystemPromptByID(
	ctx context.Context,
	id int,
) (*domain.SystemPrompt, error) {
	return d.repository.GetSystemPromptByID(ctx, id)
}

func (d *systemPromptInteractorImpl) UpdateSystemPrompt(
	ctx context.Context,
	id int,
	title, prompt *string,
	purpose *domain.SystemPromptPurpose,
) (*domain.SystemPrompt, error) {
	return d.repository.UpdateSystemPrompt(ctx, id, title, prompt, purpose)
}

func (d *systemPromptInteractorImpl) DeleteSystemPrompt(ctx context.Context, id int) error {
	return d.repository.DeleteSystemPrompt(ctx, id)
}
