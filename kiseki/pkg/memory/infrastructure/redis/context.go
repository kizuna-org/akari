package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/kizuna-org/akari/kiseki/pkg/memory/domain/entity"
	"github.com/redis/go-redis/v9"
)

const (
	contextKeyPrefix = "kiseki:context:"
	contextTTL       = 1 * time.Hour
)

// ContextRepository implements domain.ContextRepository using Redis
type ContextRepository struct {
	client *redis.Client
}

// NewContextRepository creates a new context repository
func NewContextRepository(client *redis.Client) *ContextRepository {
	return &ContextRepository{
		client: client,
	}
}

// Save saves the entire context snapshot
func (r *ContextRepository) Save(ctx context.Context, snapshot *entity.ContextSnapshot) error {
	key := contextKey(snapshot.CharacterID)

	// Serialize snapshot
	data, err := json.Marshal(snapshot)
	if err != nil {
		return fmt.Errorf("failed to marshal context snapshot: %w", err)
	}

	// Save to Redis
	if err := r.client.Set(ctx, key, data, contextTTL).Err(); err != nil {
		return fmt.Errorf("failed to save context: %w", err)
	}

	return nil
}

// Get retrieves the context snapshot for a character
func (r *ContextRepository) Get(ctx context.Context, characterID uuid.UUID) (*entity.ContextSnapshot, error) {
	key := contextKey(characterID)

	// Get from Redis
	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			// Return empty context if not found
			return entity.NewContextSnapshot(characterID), nil
		}
		return nil, fmt.Errorf("failed to get context: %w", err)
	}

	// Deserialize snapshot
	var snapshot entity.ContextSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, fmt.Errorf("failed to unmarshal context: %w", err)
	}

	// Remove expired fragments
	snapshot.RemoveExpired()

	// Save back if any fragments were removed
	if len(snapshot.Fragments) > 0 {
		_ = r.Save(ctx, &snapshot)
	}

	return &snapshot, nil
}

// Update updates specific fields in the context
func (r *ContextRepository) Update(ctx context.Context, characterID uuid.UUID, updates map[string]interface{}) error {
	// Get current snapshot
	snapshot, err := r.Get(ctx, characterID)
	if err != nil {
		return fmt.Errorf("failed to get context for update: %w", err)
	}

	// Apply updates
	if summary, ok := updates["summary"].(string); ok {
		snapshot.Summary = summary
	}
	if metadata, ok := updates["metadata"].(map[string]interface{}); ok {
		snapshot.Metadata = metadata
	}

	snapshot.UpdatedAt = time.Now()

	// Save updated snapshot
	return r.Save(ctx, snapshot)
}

// Delete removes the context for a character
func (r *ContextRepository) Delete(ctx context.Context, characterID uuid.UUID) error {
	key := contextKey(characterID)

	if err := r.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete context: %w", err)
	}

	return nil
}

// AddFragment adds a fragment to the context
func (r *ContextRepository) AddFragment(ctx context.Context, characterID uuid.UUID, fragment *entity.MemoryFragment) error {
	// Get current snapshot
	snapshot, err := r.Get(ctx, characterID)
	if err != nil {
		return fmt.Errorf("failed to get context: %w", err)
	}

	// Add fragment
	snapshot.AddFragment(fragment)

	// Save updated snapshot
	return r.Save(ctx, snapshot)
}

// RemoveFragment removes a fragment from the context
func (r *ContextRepository) RemoveFragment(ctx context.Context, characterID uuid.UUID, fragmentID uuid.UUID) error {
	// Get current snapshot
	snapshot, err := r.Get(ctx, characterID)
	if err != nil {
		return fmt.Errorf("failed to get context: %w", err)
	}

	// Remove fragment
	fragments := make([]*entity.MemoryFragment, 0)
	for _, f := range snapshot.Fragments {
		if f.ID != fragmentID {
			fragments = append(fragments, f)
		}
	}
	snapshot.Fragments = fragments
	snapshot.UpdatedAt = time.Now()

	// Save updated snapshot
	return r.Save(ctx, snapshot)
}

// Helper functions
func contextKey(characterID uuid.UUID) string {
	return fmt.Sprintf("%s%s", contextKeyPrefix, characterID.String())
}
