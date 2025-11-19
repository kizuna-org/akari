package domain_test

import (
	"testing"
	"time"

	"github.com/kizuna-org/akari/gen/ent"
	"github.com/kizuna-org/akari/pkg/database/domain"
)

func TestFromEntAkariUser_Converts(t *testing.T) {
	t.Parallel()

	now := time.Now()
	entUser := &ent.AkariUser{
		ID:        1,
		CreatedAt: now,
		UpdatedAt: now,
	}

	user, err := domain.FromEntAkariUser(entUser)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if user == nil {
		t.Fatalf("expected non-nil domain user")
	}

	if user.ID != entUser.ID {
		t.Fatalf("ID mismatch: got=%d want=%d", user.ID, entUser.ID)
	}

	if !user.CreatedAt.Equal(entUser.CreatedAt) {
		t.Fatalf("CreatedAt mismatch: got=%v want=%v", user.CreatedAt, entUser.CreatedAt)
	}

	if !user.UpdatedAt.Equal(entUser.UpdatedAt) {
		t.Fatalf("UpdatedAt mismatch: got=%v want=%v", user.UpdatedAt, entUser.UpdatedAt)
	}
}

func TestFromEntAkariUser_Nil(t *testing.T) {
	t.Parallel()

	user, err := domain.FromEntAkariUser(nil)
	if err == nil {
		t.Fatalf("expected error for nil input, got nil")
	}

	if user != nil {
		t.Fatalf("expected nil user for nil input, got non-nil")
	}
}
