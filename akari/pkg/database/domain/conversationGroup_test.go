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
	entCharacter := &ent.Character{ID: 5, Name: "character", CreatedAt: now, UpdatedAt: now}
	entConversationGroup := &ent.ConversationGroup{
		ID:        3,
		CreatedAt: now,
		Edges: ent.ConversationGroupEdges{
			Character:     entCharacter,
			Conversations: []*ent.Conversation{{ID: 100}, {ID: 101}},
		},
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

	if len(conversationGroup.Conversations) != len(entConversationGroup.Edges.Conversations) {
		t.Fatalf("Conversations length incorrect: %+v", conversationGroup.Conversations)
	}

	if conversationGroup.Conversations[0].ID != entConversationGroup.Edges.Conversations[0].ID ||
		conversationGroup.Conversations[1].ID != entConversationGroup.Edges.Conversations[1].ID {
		t.Fatalf("Conversations incorrect: %+v", conversationGroup.Conversations)
	}
}

func TestFromEntConversationGroup_Nil(t *testing.T) {
	t.Parallel()

	if domain.FromEntConversationGroup(nil) != nil {
		t.Fatalf("expected nil when input is nil")
	}
}
