package domain_test

import (
	"testing"
	"time"

	"github.com/kizuna-org/akari/gen/ent"
	"github.com/kizuna-org/akari/pkg/database/domain"
)

func TestFromEntConversationGroup_IncludesCharacter(t *testing.T) {
	t.Parallel()

	now := time.Now()
	entCharacter := &ent.Character{ID: 1, Name: "name", CreatedAt: now, UpdatedAt: now}
	entConversationGroup := &ent.ConversationGroup{
		ID:        3,
		CreatedAt: now,
		Edges:     ent.ConversationGroupEdges{Character: entCharacter},
	}

	conversationGroup, err := domain.FromEntConversationGroup(entConversationGroup)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if conversationGroup == nil {
		t.Fatalf("expected non-nil domain conversation group")
	}

	if conversationGroup.ID != entConversationGroup.ID {
		t.Fatalf("ID mismatch: %d", conversationGroup.ID)
	}

	if conversationGroup.CharacterID != entCharacter.ID {
		t.Fatalf("Character ID incorrect: %+v", conversationGroup.CharacterID)
	}
}

func TestFromEntConversationGroup_Nil(t *testing.T) {
	t.Parallel()

	conversationGroup, err := domain.FromEntConversationGroup(nil)

	if err == nil {
		t.Fatalf("expected error when input is nil")
	}

	if conversationGroup != nil {
		t.Fatalf("expected nil domain conversation group when input is nil")
	}
}
