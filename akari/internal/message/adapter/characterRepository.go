package adapter

import (
	"context"
	"fmt"

	"github.com/kizuna-org/akari/internal/message/domain"
	databaseDomain "github.com/kizuna-org/akari/pkg/database/domain"
)

type characterRepository struct {
	repository databaseDomain.CharacterRepository
}

func NewCharacterRepository(repository databaseDomain.CharacterRepository) domain.CharacterRepository {
	return &characterRepository{
		repository: repository,
	}
}

func (r *characterRepository) Get(ctx context.Context, characterID int) (*domain.Character, error) {
	character, err := r.repository.GetCharacterByID(ctx, characterID)
	if err != nil {
		return nil, fmt.Errorf("adapter: failed to get character by id: %w", err)
	}

	return &domain.Character{
		ID:              character.ID,
		Name:            character.Name,
		SystemPromptIDs: character.SystemPromptIDs,
	}, nil
}
