package redis

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kizuna-org/akari/kiseki/pkg/character/domain/entity"
)

func TestGetCharacterKey(t *testing.T) {
	id := uuid.New()
	key := getCharacterKey(id)

	if key == "" {
		t.Error("getCharacterKey() returned empty string")
	}

	if len(key) < 10 {
		t.Errorf("getCharacterKey() returned too short key: %s", key)
	}

	// Should contain the prefix
	if key[:len(characterKeyPrefix)] != characterKeyPrefix {
		t.Errorf("getCharacterKey() prefix = %v, want %v", key[:len(characterKeyPrefix)], characterKeyPrefix)
	}
}

func TestCharacterDataConversion(t *testing.T) {
	tests := []struct {
		name      string
		character *entity.Character
	}{
		{
			name: "convert character",
			character: &entity.Character{
				ID:        uuid.New(),
				Name:      "Test Character",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert to data
			data := toCharacterData(tt.character)
			if data == nil {
				t.Fatal("toCharacterData() returned nil")
			}

			if data.Name != tt.character.Name {
				t.Errorf("Name = %v, want %v", data.Name, tt.character.Name)
			}

			// Convert back to entity
			converted, err := data.toEntity()
			if err != nil {
				t.Fatalf("toEntity() error = %v", err)
			}

			if converted.ID != tt.character.ID {
				t.Errorf("ID = %v, want %v", converted.ID, tt.character.ID)
			}

			if converted.Name != tt.character.Name {
				t.Errorf("Name = %v, want %v", converted.Name, tt.character.Name)
			}
		})
	}
}

func TestNewRepository(t *testing.T) {
	// This is a simple test to verify repository creation
	// Integration tests would require a real Redis connection
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Skip("Integration test - requires Redis instance")
}

func TestRepository_Create(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Skip("Integration test - requires Redis instance")
}

func TestRepository_Get(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Skip("Integration test - requires Redis instance")
}

func TestRepository_List(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Skip("Integration test - requires Redis instance")
}

func TestRepository_Update(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Skip("Integration test - requires Redis instance")
}

func TestRepository_Delete(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Skip("Integration test - requires Redis instance")
}
