package redis

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/kizuna-org/akari/kiseki/pkg/vectordb/domain"
	"github.com/kizuna-org/akari/kiseki/pkg/vectordb/domain/entity"
	"github.com/redis/go-redis/v9"
)

// Repository implements the KVSRepository interface for Redis
type Repository struct {
	client *Client
}

// NewRepository creates a new Redis repository
func NewRepository(client *Client) *Repository {
	return &Repository{
		client: client,
	}
}

var _ domain.KVSRepository = (*Repository)(nil)

// Key formats for Redis
const (
	accessCountKey = "character:%s:fragment:%s:count"
	lastAccessKey  = "character:%s:fragment:%s:last_access"
	firstAccessKey = "character:%s:fragment:%s:first_access"
)

// getAccessCountKey returns the Redis key for access count
func getAccessCountKey(characterID, fragmentID uuid.UUID) string {
	return fmt.Sprintf(accessCountKey, characterID.String(), fragmentID.String())
}

// getLastAccessKey returns the Redis key for last access time
func getLastAccessKey(characterID, fragmentID uuid.UUID) string {
	return fmt.Sprintf(lastAccessKey, characterID.String(), fragmentID.String())
}

// getFirstAccessKey returns the Redis key for first access time
func getFirstAccessKey(characterID, fragmentID uuid.UUID) string {
	return fmt.Sprintf(firstAccessKey, characterID.String(), fragmentID.String())
}

// IncrementAccess increments the access count for a fragment
func (r *Repository) IncrementAccess(ctx context.Context, characterID uuid.UUID, fragmentID uuid.UUID) error {
	countKey := getAccessCountKey(characterID, fragmentID)
	lastKey := getLastAccessKey(characterID, fragmentID)

	pipe := r.client.GetClient().Pipeline()
	pipe.Incr(ctx, countKey)
	pipe.Set(ctx, lastKey, time.Now().Unix(), 0)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to increment access: %w", err)
	}

	return nil
}

// GetAccessInfo retrieves access information for a fragment
func (r *Repository) GetAccessInfo(ctx context.Context, characterID uuid.UUID, fragmentID uuid.UUID) (*entity.AccessInfo, error) {
	countKey := getAccessCountKey(characterID, fragmentID)
	lastKey := getLastAccessKey(characterID, fragmentID)
	firstKey := getFirstAccessKey(characterID, fragmentID)

	pipe := r.client.GetClient().Pipeline()
	countCmd := pipe.Get(ctx, countKey)
	lastCmd := pipe.Get(ctx, lastKey)
	firstCmd := pipe.Get(ctx, firstKey)

	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("failed to get access info: %w", err)
	}

	info := &entity.AccessInfo{
		FragmentID:  fragmentID,
		CharacterID: characterID,
	}

	// Parse count
	if countStr, err := countCmd.Result(); err == nil {
		if count, err := strconv.ParseInt(countStr, 10, 64); err == nil {
			info.AccessCount = count
		}
	}

	// Parse last access time
	if lastStr, err := lastCmd.Result(); err == nil {
		if lastUnix, err := strconv.ParseInt(lastStr, 10, 64); err == nil {
			info.LastAccessedAt = time.Unix(lastUnix, 0)
		}
	}

	// Parse first access time
	if firstStr, err := firstCmd.Result(); err == nil {
		if firstUnix, err := strconv.ParseInt(firstStr, 10, 64); err == nil {
			info.FirstAccessedAt = time.Unix(firstUnix, 0)
		}
	}

	return info, nil
}

// GetBatchAccessInfo retrieves access information for multiple fragments
func (r *Repository) GetBatchAccessInfo(ctx context.Context, characterID uuid.UUID, fragmentIDs []uuid.UUID) (map[uuid.UUID]*entity.AccessInfo, error) {
	if len(fragmentIDs) == 0 {
		return make(map[uuid.UUID]*entity.AccessInfo), nil
	}

	pipe := r.client.GetClient().Pipeline()

	// Map to store commands for each fragment
	countCmds := make(map[uuid.UUID]*redis.StringCmd)
	lastCmds := make(map[uuid.UUID]*redis.StringCmd)
	firstCmds := make(map[uuid.UUID]*redis.StringCmd)

	for _, fragmentID := range fragmentIDs {
		countKey := getAccessCountKey(characterID, fragmentID)
		lastKey := getLastAccessKey(characterID, fragmentID)
		firstKey := getFirstAccessKey(characterID, fragmentID)

		countCmds[fragmentID] = pipe.Get(ctx, countKey)
		lastCmds[fragmentID] = pipe.Get(ctx, lastKey)
		firstCmds[fragmentID] = pipe.Get(ctx, firstKey)
	}

	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("failed to get batch access info: %w", err)
	}

	result := make(map[uuid.UUID]*entity.AccessInfo)
	for _, fragmentID := range fragmentIDs {
		info := &entity.AccessInfo{
			FragmentID:  fragmentID,
			CharacterID: characterID,
		}

		// Parse count
		if countStr, err := countCmds[fragmentID].Result(); err == nil {
			if count, err := strconv.ParseInt(countStr, 10, 64); err == nil {
				info.AccessCount = count
			}
		}

		// Parse last access time
		if lastStr, err := lastCmds[fragmentID].Result(); err == nil {
			if lastUnix, err := strconv.ParseInt(lastStr, 10, 64); err == nil {
				info.LastAccessedAt = time.Unix(lastUnix, 0)
			}
		}

		// Parse first access time
		if firstStr, err := firstCmds[fragmentID].Result(); err == nil {
			if firstUnix, err := strconv.ParseInt(firstStr, 10, 64); err == nil {
				info.FirstAccessedAt = time.Unix(firstUnix, 0)
			}
		}

		result[fragmentID] = info
	}

	return result, nil
}

// UpdateAccessTime updates the last access time for a fragment
func (r *Repository) UpdateAccessTime(ctx context.Context, characterID uuid.UUID, fragmentID uuid.UUID) error {
	lastKey := getLastAccessKey(characterID, fragmentID)

	err := r.client.GetClient().Set(ctx, lastKey, time.Now().Unix(), 0).Err()
	if err != nil {
		return fmt.Errorf("failed to update access time: %w", err)
	}

	return nil
}

// InitializeAccessInfo initializes access information for a new fragment
func (r *Repository) InitializeAccessInfo(ctx context.Context, info entity.AccessInfo) error {
	countKey := getAccessCountKey(info.CharacterID, info.FragmentID)
	lastKey := getLastAccessKey(info.CharacterID, info.FragmentID)
	firstKey := getFirstAccessKey(info.CharacterID, info.FragmentID)

	now := time.Now().Unix()
	pipe := r.client.GetClient().Pipeline()
	pipe.Set(ctx, countKey, 0, 0)
	pipe.Set(ctx, lastKey, now, 0)
	pipe.Set(ctx, firstKey, now, 0)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to initialize access info: %w", err)
	}

	return nil
}
