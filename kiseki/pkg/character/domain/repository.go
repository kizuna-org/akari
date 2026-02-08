package domain

import (
	"context"

	"github.com/google/uuid"
	"github.com/kizuna-org/akari/kiseki/pkg/character/domain/entity"
)

// CharacterRepository defines the interface for character repository operations
type CharacterRepository interface {
	// Create creates a new character
	Create(ctx context.Context, character *entity.Character) error

	// Get retrieves a character by ID
	Get(ctx context.Context, id uuid.UUID) (*entity.Character, error)

	// List retrieves all characters
	List(ctx context.Context) ([]*entity.Character, error)

	// Update updates an existing character
	Update(ctx context.Context, character *entity.Character) error

	// Delete deletes a character by ID
	Delete(ctx context.Context, id uuid.UUID) error

	// Exists checks if a character exists by ID
	Exists(ctx context.Context, id uuid.UUID) (bool, error)
}
