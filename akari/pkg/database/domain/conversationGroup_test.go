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

	conversationGroup := domain.FromEntConversationGroup(entConversationGroup)
	if conversationGroup == nil {
		t.Fatalf("expected non-nil domain conversation group")
	}

	if conversationGroup.ID != entConversationGroup.ID {
		t.Fatalf("ID mismatch: %d", conversationGroup.ID)
	}

	if conversationGroup.Character == nil {
		t.Fatalf("Character not converted: %+v", conversationGroup.Character)
	}

	if conversationGroup.Character.ID != entCharacter.ID || conversationGroup.Character.Name != entCharacter.Name {
		t.Fatalf("Character fields incorrect: %+v", conversationGroup.Character)
	}
}

func TestFromEntConversationGroup_Nil(t *testing.T) {
	t.Parallel()

	if domain.FromEntConversationGroup(nil) != nil {
		t.Fatalf("expected nil when input is nil")
	}
}
