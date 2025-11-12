package interactor

import (
	"context"

	"github.com/kizuna-org/akari/pkg/database/domain"
)

type CharacterInteractor interface {
	CreateCharacter(ctx context.Context, name string) (*domain.Character, error)
	GetCharacterByID(ctx context.Context, characterID int) (*domain.Character, error)
	GetCharacterWithSystemPromptByID(ctx context.Context, characterID int) (*domain.Character, error)
	ListCharacters(ctx context.Context, activeOnly bool) ([]*domain.Character, error)
	UpdateCharacter(ctx context.Context, characterID int, name *string, isActive *bool) (*domain.Character, error)
	DeleteCharacter(ctx context.Context, characterID int) error
}

type characterInteractorImpl struct {
	repository domain.CharacterRepository
}

func NewCharacterInteractor(repository domain.CharacterRepository) CharacterInteractor {
	return &characterInteractorImpl{
		repository: repository,
	}
}

func (c *characterInteractorImpl) CreateCharacter(
	ctx context.Context,
	name string,
) (*domain.Character, error) {
	return c.repository.CreateCharacter(ctx, name)
}

func (c *characterInteractorImpl) GetCharacterByID(
	ctx context.Context,
	characterID int,
) (*domain.Character, error) {
	return c.repository.GetCharacterByID(ctx, characterID)
}

func (c *characterInteractorImpl) GetCharacterWithSystemPromptByID(
	ctx context.Context,
	characterID int,
) (*domain.Character, error) {
	return c.repository.GetCharacterWithSystemPromptByID(ctx, characterID)
}

func (c *characterInteractorImpl) ListCharacters(
	ctx context.Context,
	activeOnly bool,
) ([]*domain.Character, error) {
	return c.repository.ListCharacters(ctx, activeOnly)
}

func (c *characterInteractorImpl) UpdateCharacter(
	ctx context.Context,
	characterID int,
	name *string,
	isActive *bool,
) (*domain.Character, error) {
	return c.repository.UpdateCharacter(ctx, characterID, name, isActive)
}

func (c *characterInteractorImpl) DeleteCharacter(ctx context.Context, characterID int) error {
	return c.repository.DeleteCharacter(ctx, characterID)
}
