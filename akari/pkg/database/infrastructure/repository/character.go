package repository

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/kizuna-org/akari/gen/ent/character"
	"github.com/kizuna-org/akari/pkg/database/domain"
)

func (r *repositoryImpl) CreateCharacter(
	ctx context.Context,
	name string,
) (*domain.Character, error) {
	character, err := r.client.CharacterClient().
		Create().
		SetName(name).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create character: %w", err)
	}

	r.logger.Info("character created",
		slog.Int("id", character.ID),
		slog.String("name", name),
	)

	return character, nil
}

func (r *repositoryImpl) GetCharacterByID(ctx context.Context, characterID int) (*domain.Character, error) {
	char, err := r.client.CharacterClient().Get(ctx, characterID)
	if err != nil {
		return nil, fmt.Errorf("failed to get character by id: %w", err)
	}

	return char, nil
}

func (r *repositoryImpl) GetCharacterWithEdgesByID(
	ctx context.Context,
	characterID int,
) (*domain.Character, error) {
	char, err := r.client.CharacterClient().
		Query().
		Where(character.IDEQ(characterID)).
		WithConfig().
		WithSystemPrompts().
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get character with system prompt: %w", err)
	}

	return char, nil
}

func (r *repositoryImpl) ListCharacters(ctx context.Context) ([]*domain.Character, error) {
	query := r.client.CharacterClient().Query()

	characters, err := query.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list characters: %w", err)
	}

	return characters, nil
}

func (r *repositoryImpl) UpdateCharacter(
	ctx context.Context,
	characterID int,
	name *string,
) (*domain.Character, error) {
	updater := r.client.CharacterClient().UpdateOneID(characterID)

	if name != nil {
		updater = updater.SetName(*name)
	}

	char, err := updater.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to update character: %w", err)
	}

	logAttrs := []any{slog.Int("id", characterID)}
	if name != nil {
		logAttrs = append(logAttrs, slog.String("name", *name))
	}

	r.logger.Info("character updated", logAttrs...)

	return char, nil
}

func (r *repositoryImpl) DeleteCharacter(ctx context.Context, characterID int) error {
	if err := r.client.CharacterClient().DeleteOneID(characterID).Exec(ctx); err != nil {
		return fmt.Errorf("failed to delete character: %w", err)
	}

	r.logger.Info("character deleted",
		slog.Int("id", characterID),
	)

	return nil
}
