package domain

import (
	"context"

	"github.com/google/uuid"
	"github.com/kizuna-org/akari/kiseki/pkg/memory/domain/entity"
)

// InputBufferRepository manages input buffer (sensory memory)
type InputBufferRepository interface {
	// Push adds a new item to the input buffer
	Push(ctx context.Context, fragment *entity.MemoryFragment) error

	// GetAll retrieves all items in the input buffer for a character
	GetAll(ctx context.Context, characterID uuid.UUID) ([]*entity.MemoryFragment, error)

	// GetRecent retrieves the N most recent items
	GetRecent(ctx context.Context, characterID uuid.UUID, limit int) ([]*entity.MemoryFragment, error)

	// Clear removes all items from the input buffer for a character
	Clear(ctx context.Context, characterID uuid.UUID) error
}

// ContextRepository manages context (short-term memory)
type ContextRepository interface {
	// Save saves the entire context snapshot
	Save(ctx context.Context, snapshot *entity.ContextSnapshot) error

	// Get retrieves the context snapshot for a character
	Get(ctx context.Context, characterID uuid.UUID) (*entity.ContextSnapshot, error)

	// Update updates specific fields in the context
	Update(ctx context.Context, characterID uuid.UUID, updates map[string]interface{}) error

	// Delete removes the context for a character
	Delete(ctx context.Context, characterID uuid.UUID) error

	// AddFragment adds a fragment to the context
	AddFragment(ctx context.Context, characterID uuid.UUID, fragment *entity.MemoryFragment) error

	// RemoveFragment removes a fragment from the context
	RemoveFragment(ctx context.Context, characterID uuid.UUID, fragmentID uuid.UUID) error
}
