package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/kizuna-org/akari/kiseki/pkg/character/domain"
	"github.com/kizuna-org/akari/kiseki/pkg/character/domain/entity"
	"github.com/redis/go-redis/v9"
)

// Repository implements the CharacterRepository interface using Redis
type Repository struct {
	client *redis.Client
}

// NewRepository creates a new character repository
func NewRepository(client *redis.Client) *Repository {
	return &Repository{
		client: client,
	}
}

var _ domain.CharacterRepository = (*Repository)(nil)

const (
	characterKeyPrefix = "character:"
	characterListKey   = "characters:list"
)

// getCharacterKey returns the Redis key for a character
func getCharacterKey(id uuid.UUID) string {
	return fmt.Sprintf("%s%s", characterKeyPrefix, id.String())
}

// characterData is the internal representation for JSON storage
type characterData struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// toCharacterData converts entity to storage format
func toCharacterData(character *entity.Character) *characterData {
	return &characterData{
		ID:        character.ID.String(),
		Name:      character.Name,
		CreatedAt: character.CreatedAt,
		UpdatedAt: character.UpdatedAt,
	}
}

// toEntity converts storage format to entity
func (cd *characterData) toEntity() (*entity.Character, error) {
	id, err := uuid.Parse(cd.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid character ID: %w", err)
	}

	return &entity.Character{
		ID:        id,
		Name:      cd.Name,
		CreatedAt: cd.CreatedAt,
		UpdatedAt: cd.UpdatedAt,
	}, nil
}

// Create creates a new character
func (r *Repository) Create(ctx context.Context, character *entity.Character) error {
	data := toCharacterData(character)
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal character: %w", err)
	}

	key := getCharacterKey(character.ID)

	// Use pipeline for atomic operations
	pipe := r.client.Pipeline()
	pipe.Set(ctx, key, jsonData, 0)
	pipe.SAdd(ctx, characterListKey, character.ID.String())

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create character: %w", err)
	}

	return nil
}

// Get retrieves a character by ID
func (r *Repository) Get(ctx context.Context, id uuid.UUID) (*entity.Character, error) {
	key := getCharacterKey(id)

	jsonData, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, fmt.Errorf("character not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get character: %w", err)
	}

	var data characterData
	if err := json.Unmarshal([]byte(jsonData), &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal character: %w", err)
	}

	return data.toEntity()
}

// List retrieves all characters
func (r *Repository) List(ctx context.Context) ([]*entity.Character, error) {
	// Get all character IDs from the set
	idStrings, err := r.client.SMembers(ctx, characterListKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to list character IDs: %w", err)
	}

	if len(idStrings) == 0 {
		return []*entity.Character{}, nil
	}

	// Get all characters using pipeline
	pipe := r.client.Pipeline()
	cmds := make(map[string]*redis.StringCmd)

	for _, idStr := range idStrings {
		id, err := uuid.Parse(idStr)
		if err != nil {
			continue // Skip invalid IDs
		}
		key := getCharacterKey(id)
		cmds[idStr] = pipe.Get(ctx, key)
	}

	_, err = pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("failed to fetch characters: %w", err)
	}

	// Parse results
	characters := make([]*entity.Character, 0, len(cmds))
	for _, cmd := range cmds {
		jsonData, err := cmd.Result()
		if err != nil {
			continue // Skip missing entries
		}

		var data characterData
		if err := json.Unmarshal([]byte(jsonData), &data); err != nil {
			continue // Skip invalid JSON
		}

		character, err := data.toEntity()
		if err != nil {
			continue // Skip invalid data
		}

		characters = append(characters, character)
	}

	return characters, nil
}

// Update updates an existing character
func (r *Repository) Update(ctx context.Context, character *entity.Character) error {
	// Check if character exists
	exists, err := r.Exists(ctx, character.ID)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("character not found")
	}

	data := toCharacterData(character)
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal character: %w", err)
	}

	key := getCharacterKey(character.ID)
	err = r.client.Set(ctx, key, jsonData, 0).Err()
	if err != nil {
		return fmt.Errorf("failed to update character: %w", err)
	}

	return nil
}

// Delete deletes a character by ID
func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	key := getCharacterKey(id)

	// Use pipeline for atomic operations
	pipe := r.client.Pipeline()
	pipe.Del(ctx, key)
	pipe.SRem(ctx, characterListKey, id.String())

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete character: %w", err)
	}

	return nil
}

// Exists checks if a character exists by ID
func (r *Repository) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	key := getCharacterKey(id)

	exists, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check character existence: %w", err)
	}

	return exists > 0, nil
}
