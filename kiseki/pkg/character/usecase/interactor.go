package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/kizuna-org/akari/kiseki/pkg/character/domain"
	"github.com/kizuna-org/akari/kiseki/pkg/character/domain/entity"
)

// CharacterInteractor handles character use cases
type CharacterInteractor struct {
	repo domain.CharacterRepository
}

// NewCharacterInteractor creates a new character interactor
func NewCharacterInteractor(repo domain.CharacterRepository) *CharacterInteractor {
	return &CharacterInteractor{
		repo: repo,
	}
}

// CreateCharacterInput represents the input for creating a character
type CreateCharacterInput struct {
	Name string
}

// CreateCharacterOutput represents the output for creating a character
type CreateCharacterOutput struct {
	Character *entity.Character
}

// CreateCharacter creates a new character
func (i *CharacterInteractor) CreateCharacter(ctx context.Context, input CreateCharacterInput) (*CreateCharacterOutput, error) {
	if input.Name == "" {
		return nil, fmt.Errorf("character name is required")
	}

	character := entity.NewCharacter(input.Name)

	if err := i.repo.Create(ctx, character); err != nil {
		return nil, fmt.Errorf("failed to create character: %w", err)
	}

	return &CreateCharacterOutput{
		Character: character,
	}, nil
}

// GetCharacterInput represents the input for getting a character
type GetCharacterInput struct {
	ID uuid.UUID
}

// GetCharacterOutput represents the output for getting a character
type GetCharacterOutput struct {
	Character *entity.Character
}

// GetCharacter retrieves a character by ID
func (i *CharacterInteractor) GetCharacter(ctx context.Context, input GetCharacterInput) (*GetCharacterOutput, error) {
	character, err := i.repo.Get(ctx, input.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get character: %w", err)
	}

	return &GetCharacterOutput{
		Character: character,
	}, nil
}

// ListCharactersOutput represents the output for listing characters
type ListCharactersOutput struct {
	Characters []*entity.Character
}

// ListCharacters retrieves all characters
func (i *CharacterInteractor) ListCharacters(ctx context.Context) (*ListCharactersOutput, error) {
	characters, err := i.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list characters: %w", err)
	}

	return &ListCharactersOutput{
		Characters: characters,
	}, nil
}

// UpdateCharacterInput represents the input for updating a character
type UpdateCharacterInput struct {
	ID   uuid.UUID
	Name string
}

// UpdateCharacterOutput represents the output for updating a character
type UpdateCharacterOutput struct {
	Character *entity.Character
}

// UpdateCharacter updates an existing character
func (i *CharacterInteractor) UpdateCharacter(ctx context.Context, input UpdateCharacterInput) (*UpdateCharacterOutput, error) {
	if input.Name == "" {
		return nil, fmt.Errorf("character name is required")
	}

	character, err := i.repo.Get(ctx, input.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get character: %w", err)
	}

	character.Update(input.Name)

	if err := i.repo.Update(ctx, character); err != nil {
		return nil, fmt.Errorf("failed to update character: %w", err)
	}

	return &UpdateCharacterOutput{
		Character: character,
	}, nil
}

// DeleteCharacterInput represents the input for deleting a character
type DeleteCharacterInput struct {
	ID uuid.UUID
}

// DeleteCharacter deletes a character by ID
func (i *CharacterInteractor) DeleteCharacter(ctx context.Context, input DeleteCharacterInput) error {
	exists, err := i.repo.Exists(ctx, input.ID)
	if err != nil {
		return fmt.Errorf("failed to check character existence: %w", err)
	}
	if !exists {
		return fmt.Errorf("character not found")
	}

	if err := i.repo.Delete(ctx, input.ID); err != nil {
		return fmt.Errorf("failed to delete character: %w", err)
	}

	return nil
}
