package entity

import (
	"testing"
	"time"
)

func TestNewCharacter(t *testing.T) {
	tests := []struct {
		name     string
		charName string
	}{
		{
			name:     "create character with name",
			charName: "Test Character",
		},
		{
			name:     "create character with empty name",
			charName: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			char := NewCharacter(tt.charName)

			if char == nil {
				t.Fatal("NewCharacter() returned nil")
			}

			if char.Name != tt.charName {
				t.Errorf("Name = %v, want %v", char.Name, tt.charName)
			}

			if char.ID.String() == "" {
				t.Error("ID should not be empty")
			}

			if char.CreatedAt.IsZero() {
				t.Error("CreatedAt should not be zero")
			}

			if char.UpdatedAt.IsZero() {
				t.Error("UpdatedAt should not be zero")
			}

			if !char.CreatedAt.Equal(char.UpdatedAt) {
				t.Error("CreatedAt and UpdatedAt should be equal for new character")
			}
		})
	}
}

func TestCharacter_Update(t *testing.T) {
	tests := []struct {
		name        string
		initialName string
		newName     string
	}{
		{
			name:        "update character name",
			initialName: "Original Name",
			newName:     "Updated Name",
		},
		{
			name:        "update to empty name",
			initialName: "Original Name",
			newName:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			char := NewCharacter(tt.initialName)
			originalUpdatedAt := char.UpdatedAt

			// Wait a bit to ensure time difference
			time.Sleep(10 * time.Millisecond)

			char.Update(tt.newName)

			if char.Name != tt.newName {
				t.Errorf("Name = %v, want %v", char.Name, tt.newName)
			}

			if !char.UpdatedAt.After(originalUpdatedAt) {
				t.Error("UpdatedAt should be updated")
			}

			if !char.CreatedAt.Before(char.UpdatedAt) && !char.CreatedAt.Equal(char.UpdatedAt) {
				t.Error("CreatedAt should be before or equal to UpdatedAt")
			}
		})
	}
}
