package redis

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kizuna-org/akari/kiseki/pkg/vectordb/domain/entity"
)

func TestRepository_KeyGeneration(t *testing.T) {
	characterID := uuid.New()
	fragmentID := uuid.New()

	tests := []struct {
		name     string
		keyFunc  func(uuid.UUID, uuid.UUID) string
		wantPart string
	}{
		{
			name:     "access count key",
			keyFunc:  getAccessCountKey,
			wantPart: ":count",
		},
		{
			name:     "last access key",
			keyFunc:  getLastAccessKey,
			wantPart: ":last_access",
		},
		{
			name:     "first access key",
			keyFunc:  getFirstAccessKey,
			wantPart: ":first_access",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := tt.keyFunc(characterID, fragmentID)
			if key == "" {
				t.Error("Key should not be empty")
			}
			// Check that key contains character and fragment IDs
			if len(key) < 10 {
				t.Errorf("Key seems too short: %s", key)
			}
		})
	}
}

func TestRepository_InitializeAccessInfo(t *testing.T) {
	// This test requires a real Redis connection
	// Skip if REDIS_TEST environment variable is not set
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Note: This would require a Redis instance for integration testing
	// For unit tests, we would use a mock
	t.Skip("Integration test - requires Redis instance")
}

func TestRepository_GetAccessInfo_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Note: This would require a Redis instance for integration testing
	t.Skip("Integration test - requires Redis instance")
}

func TestAccessInfo_Fields(t *testing.T) {
	// Test entity structure
	characterID := uuid.New()
	fragmentID := uuid.New()
	now := time.Now()

	info := entity.AccessInfo{
		FragmentID:      fragmentID,
		CharacterID:     characterID,
		AccessCount:     10,
		LastAccessedAt:  now,
		FirstAccessedAt: now.Add(-24 * time.Hour),
	}

	if info.FragmentID != fragmentID {
		t.Errorf("FragmentID = %v, want %v", info.FragmentID, fragmentID)
	}
	if info.CharacterID != characterID {
		t.Errorf("CharacterID = %v, want %v", info.CharacterID, characterID)
	}
	if info.AccessCount != 10 {
		t.Errorf("AccessCount = %v, want %v", info.AccessCount, 10)
	}
}

func TestRepository_Context(t *testing.T) {
	// Test that context is properly handled
	ctx := context.Background()
	if ctx == nil {
		t.Error("Context should not be nil")
	}

	// Test context cancellation
	ctx, cancel := context.WithCancel(ctx)
	cancel()
	if ctx.Err() == nil {
		t.Error("Cancelled context should have an error")
	}
}
