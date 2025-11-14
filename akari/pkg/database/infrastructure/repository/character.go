package repository

import (
	"context"
	"fmt"

	"github.com/kizuna-org/akari/gen/ent/character"
	"github.com/kizuna-org/akari/pkg/database/domain"
)

func (r *repositoryImpl) GetCharacterByID(
	ctx context.Context,
	characterID int,
) (*domain.Character, error) {
	character, err := r.client.CharacterClient().
		Query().
		Where(character.IDEQ(characterID)).
		WithConfig().
		WithSystemPrompts().
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get character with system prompt: %w", err)
	}

	return character, nil
}

func (r *repositoryImpl) ListCharacters(ctx context.Context) ([]*domain.Character, error) {
	characters, err := r.client.CharacterClient().
		Query().
		WithConfig().
		WithSystemPrompts().
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list characters: %w", err)
	}

	return characters, nil
}
