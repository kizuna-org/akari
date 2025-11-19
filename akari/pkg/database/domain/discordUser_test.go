package domain_test

import (
	"testing"
	"time"

	"github.com/kizuna-org/akari/gen/ent"
	"github.com/kizuna-org/akari/pkg/database/domain"
)

func TestFromEntDiscordUser_NilAndFields(t *testing.T) {
	t.Parallel()

	now := time.Now()

	if domain.FromEntDiscordUser(nil) != nil {
		t.Fatalf("expected nil for nil input")
	}

	entUser := &ent.DiscordUser{ID: "u-1", Username: "user1", Bot: false, CreatedAt: now, UpdatedAt: now}

	duser := domain.FromEntDiscordUser(entUser)
	if duser == nil {
		t.Fatalf("expected non-nil domain user")
	}

	if duser.ID != entUser.ID {
		t.Fatalf("ID mismatch: got=%v want=%v", duser.ID, entUser.ID)
	}

	if duser.Username != entUser.Username {
		t.Fatalf("Username mismatch: got=%v want=%v", duser.Username, entUser.Username)
	}
}

func TestFromEntDiscordUser_Nil(t *testing.T) {
	t.Parallel()

	if domain.FromEntDiscordUser(nil) != nil {
		t.Fatalf("expected nil when input is nil")
	}
}
