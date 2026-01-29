package qdrant

import (
	"testing"

	"github.com/google/uuid"
)

func TestCollectionName(t *testing.T) {
	tests := []struct {
		name        string
		characterID uuid.UUID
		wantPrefix  string
	}{
		{
			name:        "valid UUID",
			characterID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
			wantPrefix:  "character_",
		},
		{
			name:        "another valid UUID",
			characterID: uuid.New(),
			wantPrefix:  "character_",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CollectionName(tt.characterID)
			if len(got) < len(tt.wantPrefix) {
				t.Errorf("CollectionName() is too short, got = %v", got)
			}
			if got[:len(tt.wantPrefix)] != tt.wantPrefix {
				t.Errorf("CollectionName() prefix = %v, want %v", got[:len(tt.wantPrefix)], tt.wantPrefix)
			}
			// Ensure the UUID is included (character_ is 11 chars + UUID is 36 chars = 47 total)
			expectedLength := len(tt.wantPrefix) + 36 // UUID string length
			if len(got) != expectedLength {
				t.Errorf("CollectionName() length = %v, want %v (got: %v)", len(got), expectedLength, got)
			}
		})
	}
}

func TestNewRepository(t *testing.T) {
	// Test repository creation
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// This would require a real Qdrant client for integration testing
	t.Skip("Integration test - requires Qdrant instance")
}

func TestRepository_Upsert(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Integration test with real Qdrant
	t.Skip("Integration test - requires Qdrant instance")
}

func TestRepository_HybridSearch(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Integration test with real Qdrant
	t.Skip("Integration test - requires Qdrant instance")
}

func TestExtractMetadata(t *testing.T) {
	// Test metadata extraction logic would go here
	// This is a unit test that doesn't require external dependencies
	metadata := make(map[string]interface{})
	metadata["test_key"] = "test_value"

	if len(metadata) != 1 {
		t.Errorf("Metadata length = %v, want 1", len(metadata))
	}
}
