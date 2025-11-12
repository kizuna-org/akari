package interactor

import (
	"context"

	"github.com/kizuna-org/akari/pkg/database/domain"
)

type CharacterConfigInteractor interface {
	CreateCharacterConfig(
		ctx context.Context,
		characterID *int,
		nameRegexp *string,
		defaultSystemPrompt string,
	) (*domain.CharacterConfig, error)
	GetCharacterConfigByID(ctx context.Context, id int) (*domain.CharacterConfig, error)
	GetCharacterConfigByCharacterID(ctx context.Context, characterID int) (*domain.CharacterConfig, error)
	UpdateCharacterConfig(
		ctx context.Context,
		id int,
		characterID *int,
		nameRegexp *string,
		defaultSystemPrompt *string,
	) (*domain.CharacterConfig, error)
	DeleteCharacterConfig(ctx context.Context, id int) error
}

type characterConfigInteractorImpl struct {
	repository domain.CharacterConfigRepository
}

func NewCharacterConfigInteractor(repository domain.CharacterConfigRepository) CharacterConfigInteractor {
	return &characterConfigInteractorImpl{repository: repository}
}

func (i *characterConfigInteractorImpl) CreateCharacterConfig(
	ctx context.Context,
	characterID *int,
	nameRegexp *string,
	defaultSystemPrompt string,
) (*domain.CharacterConfig, error) {
	return i.repository.CreateCharacterConfig(ctx, characterID, nameRegexp, defaultSystemPrompt)
}

func (i *characterConfigInteractorImpl) GetCharacterConfigByID(
	ctx context.Context,
	id int,
) (*domain.CharacterConfig, error) {
	return i.repository.GetCharacterConfigByID(ctx, id)
}

func (i *characterConfigInteractorImpl) GetCharacterConfigByCharacterID(
	ctx context.Context,
	characterID int,
) (*domain.CharacterConfig, error) {
	return i.repository.GetCharacterConfigByCharacterID(ctx, characterID)
}

func (i *characterConfigInteractorImpl) UpdateCharacterConfig(
	ctx context.Context,
	id int,
	characterID *int,
	nameRegexp *string,
	defaultSystemPrompt *string,
) (*domain.CharacterConfig, error) {
	return i.repository.UpdateCharacterConfig(ctx, id, characterID, nameRegexp, defaultSystemPrompt)
}

func (i *characterConfigInteractorImpl) DeleteCharacterConfig(
	ctx context.Context,
	id int,
) error {
	return i.repository.DeleteCharacterConfig(ctx, id)
}
