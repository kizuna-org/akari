package interactor

import (
	"context"

	"github.com/kizuna-org/akari/pkg/database/domain"
)

type DatabaseInteractor interface {
	Close() error
	HealthCheck(ctx context.Context) error
	WithTransaction(ctx context.Context, fn domain.TxFunc) error
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

type databaseInteractorImpl struct {
	repository domain.DatabaseRepository
}

func NewDatabaseInteractor(repository domain.DatabaseRepository) DatabaseInteractor {
	return &databaseInteractorImpl{
		repository: repository,
	}
}

func (d *databaseInteractorImpl) Close() error {
	return d.repository.Close()
}

func (d *databaseInteractorImpl) HealthCheck(ctx context.Context) error {
	return d.repository.HealthCheck(ctx)
}

func (d *databaseInteractorImpl) WithTransaction(ctx context.Context, fn domain.TxFunc) error {
	return d.repository.WithTransaction(ctx, fn)
}

func (d *databaseInteractorImpl) CreateSystemPrompt(
	ctx context.Context,
	title, prompt string,
	purpose domain.SystemPromptPurpose,
) (*domain.SystemPrompt, error) {
	return d.repository.CreateSystemPrompt(ctx, title, prompt, purpose)
}

func (d *databaseInteractorImpl) GetSystemPromptByID(
	ctx context.Context,
	id int,
) (*domain.SystemPrompt, error) {
	return d.repository.GetSystemPromptByID(ctx, id)
}

func (d *databaseInteractorImpl) UpdateSystemPrompt(
	ctx context.Context,
	id int,
	title, prompt *string,
	purpose *domain.SystemPromptPurpose,
) (*domain.SystemPrompt, error) {
	return d.repository.UpdateSystemPrompt(ctx, id, title, prompt, purpose)
}

func (d *databaseInteractorImpl) DeleteSystemPrompt(ctx context.Context, id int) error {
	return d.repository.DeleteSystemPrompt(ctx, id)
}
