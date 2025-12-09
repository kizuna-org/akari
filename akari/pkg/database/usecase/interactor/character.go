package interactor

//go:generate go tool mockgen -package=mock -source=character.go -destination=mock/character.go

import (
	"context"

	"github.com/kizuna-org/akari/pkg/database/domain"
)

type CharacterInteractor interface {
	GetCharacterByID(ctx context.Context, characterID int) (*domain.Character, error)
	ListCharacters(ctx context.Context) ([]*domain.Character, error)
}

type characterInteractorImpl struct {
	repository domain.CharacterRepository
}

func NewCharacterInteractor(repository domain.CharacterRepository) CharacterInteractor {
	return &characterInteractorImpl{
		repository: repository,
	}
}

func (c *characterInteractorImpl) GetCharacterByID(
	ctx context.Context,
	characterID int,
) (*domain.Character, error) {
	return c.repository.GetCharacterByID(ctx, characterID)
}

func (c *characterInteractorImpl) ListCharacters(
	ctx context.Context,
) ([]*domain.Character, error) {
	return c.repository.ListCharacters(ctx)
}
