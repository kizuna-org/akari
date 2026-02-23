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
	inputBufferKeyPrefix = "kiseki:input_buffer:"
	inputBufferTTL       = 10 * time.Second
	inputBufferMaxSize   = 100 // Keep last 100 items
)

// InputBufferRepository implements domain.InputBufferRepository using Redis
type InputBufferRepository struct {
	client *redis.Client
}

// NewInputBufferRepository creates a new input buffer repository
func NewInputBufferRepository(client *redis.Client) *InputBufferRepository {
	return &InputBufferRepository{
		client: client,
	}
}

// Push adds a new item to the input buffer
func (r *InputBufferRepository) Push(ctx context.Context, fragment *entity.MemoryFragment) error {
	key := inputBufferKey(fragment.CharacterID)

	// Serialize fragment
	data, err := json.Marshal(fragment)
	if err != nil {
		return fmt.Errorf("failed to marshal fragment: %w", err)
	}

	// Push to list (left push for FIFO)
	if err := r.client.LPush(ctx, key, data).Err(); err != nil {
		return fmt.Errorf("failed to push to input buffer: %w", err)
	}

	// Trim to max size
	if err := r.client.LTrim(ctx, key, 0, int64(inputBufferMaxSize-1)).Err(); err != nil {
		return fmt.Errorf("failed to trim input buffer: %w", err)
	}

	// Set expiration
	if err := r.client.Expire(ctx, key, inputBufferTTL).Err(); err != nil {
		return fmt.Errorf("failed to set expiration: %w", err)
	}

	return nil
}

// GetAll retrieves all items in the input buffer for a character
func (r *InputBufferRepository) GetAll(ctx context.Context, characterID uuid.UUID) ([]*entity.MemoryFragment, error) {
	key := inputBufferKey(characterID)

	// Get all items from list
	items, err := r.client.LRange(ctx, key, 0, -1).Result()
	if err != nil {
		if err == redis.Nil {
			return []*entity.MemoryFragment{}, nil
		}
		return nil, fmt.Errorf("failed to get input buffer: %w", err)
	}

	// Deserialize fragments
	fragments := make([]*entity.MemoryFragment, 0, len(items))
	for _, item := range items {
		var fragment entity.MemoryFragment
		if err := json.Unmarshal([]byte(item), &fragment); err != nil {
			continue // Skip invalid items
		}

		// Filter out expired items
		if !fragment.IsExpired() {
			fragments = append(fragments, &fragment)
		}
	}

	return fragments, nil
}

// GetRecent retrieves the N most recent items
func (r *InputBufferRepository) GetRecent(ctx context.Context, characterID uuid.UUID, limit int) ([]*entity.MemoryFragment, error) {
	key := inputBufferKey(characterID)

	// Get recent items (0 to limit-1)
	items, err := r.client.LRange(ctx, key, 0, int64(limit-1)).Result()
	if err != nil {
		if err == redis.Nil {
			return []*entity.MemoryFragment{}, nil
		}
		return nil, fmt.Errorf("failed to get recent items: %w", err)
	}

	// Deserialize fragments
	fragments := make([]*entity.MemoryFragment, 0, len(items))
	for _, item := range items {
		var fragment entity.MemoryFragment
		if err := json.Unmarshal([]byte(item), &fragment); err != nil {
			continue
		}

		if !fragment.IsExpired() {
			fragments = append(fragments, &fragment)
		}
	}

	return fragments, nil
}

// Clear removes all items from the input buffer for a character
func (r *InputBufferRepository) Clear(ctx context.Context, characterID uuid.UUID) error {
	key := inputBufferKey(characterID)

	if err := r.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to clear input buffer: %w", err)
	}

	return nil
}

// Helper functions
func inputBufferKey(characterID uuid.UUID) string {
	return fmt.Sprintf("%s%s", inputBufferKeyPrefix, characterID.String())
}
