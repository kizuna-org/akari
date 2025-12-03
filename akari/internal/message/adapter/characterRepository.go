package adapter

import (
	"context"
	"fmt"

	"github.com/kizuna-org/akari/internal/message/domain"
	databaseInteractor "github.com/kizuna-org/akari/pkg/database/usecase/interactor"
)

type characterRepository struct {
	interactor databaseInteractor.CharacterInteractor
}

func NewCharacterRepository(interactor databaseInteractor.CharacterInteractor) domain.CharacterRepository {
	return &characterRepository{
		interactor: interactor,
	}
}

func (r *characterRepository) GetCharacterByID(ctx context.Context, characterID int) (*domain.Character, error) {
	character, err := r.interactor.GetCharacterByID(ctx, characterID)
	if err != nil {
		return nil, fmt.Errorf("failed to get character by id: %w", err)
	}

	return &domain.Character{
		ID:              character.ID,
		Name:            character.Name,
		SystemPromptIDs: character.SystemPromptIDs,
	}, nil
}
