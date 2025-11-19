package domain_test

import (
	"testing"
	"time"

	"github.com/kizuna-org/akari/gen/ent"
	"github.com/kizuna-org/akari/pkg/database/domain"
)

func TestFromEntConversation_ConvertsNestedEdges(t *testing.T) {
	t.Parallel()

	now := time.Now()

	entUser := &ent.AkariUser{ID: 1, CreatedAt: now, UpdatedAt: now}
	entDiscordMessage := &ent.DiscordMessage{ID: "message-id", CreatedAt: now}
	entConversationGroup := &ent.ConversationGroup{ID: 1, CreatedAt: now}

	entConversation := &ent.Conversation{
		ID:        1,
		CreatedAt: now,
		Edges: ent.ConversationEdges{
			User:              entUser,
			DiscordMessage:    entDiscordMessage,
			ConversationGroup: entConversationGroup,
		},
	}

	conversation, err := domain.FromEntConversation(entConversation)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if conversation == nil {
		t.Fatalf("expected non-nil domain conversation")
	}

	if conversation.ID != entConversation.ID {
		t.Fatalf("ID mismatch: got=%d want=%d", conversation.ID, entConversation.ID)
	}

	if conversation.UserID != entUser.ID {
		t.Fatalf("User ID mismatch: got=%d want=%d", conversation.UserID, entUser.ID)
	}

	if conversation.DiscordMessageID != entDiscordMessage.ID {
		t.Fatalf("DiscordMessage ID mismatch: got=%s want=%s", conversation.DiscordMessageID, entDiscordMessage.ID)
	}

	if conversation.ConversationGroupID != entConversationGroup.ID {
		t.Fatalf("ConversationGroup ID mismatch: got=%d want=%d", conversation.ConversationGroupID, entConversationGroup.ID)
	}

	if !conversation.CreatedAt.Equal(entConversation.CreatedAt) {
		t.Fatalf("CreatedAt mismatch: got=%v want=%v", conversation.CreatedAt, entConversation.CreatedAt)
	}
}

func TestFromEntConversation_Nil(t *testing.T) {
	t.Parallel()

	conversation, err := domain.FromEntConversation(nil)
	if err == nil {
		t.Fatalf("expected error when input is nil")
	}

	if conversation != nil {
		t.Fatalf("expected nil conversation when input is nil")
	}
}
