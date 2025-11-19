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

	entUser := &ent.AkariUser{ID: 11, CreatedAt: now, UpdatedAt: now}
	entDiscordMessage := &ent.DiscordMessage{ID: "22", CreatedAt: now}
	entConversationGroup := &ent.ConversationGroup{ID: 33, CreatedAt: now}

	entConversation := &ent.Conversation{
		ID:        5,
		CreatedAt: now,
		Edges: ent.ConversationEdges{
			User:              entUser,
			DiscordMessage:    entDiscordMessage,
			ConversationGroup: entConversationGroup,
		},
	}

	conv := domain.FromEntConversation(entConversation)
	if conv == nil {
		t.Fatalf("expected non-nil domain conversation")
	}

	if conv.ID != entConversation.ID {
		t.Fatalf("ID mismatch: got=%d want=%d", conv.ID, entConversation.ID)
	}

	if conv.User == nil {
		t.Fatalf("expected nested User to be converted")
	}

	if conv.User.ID != entUser.ID {
		t.Fatalf("User ID mismatch: got=%d want=%d", conv.User.ID, entUser.ID)
	}

	if conv.DiscordMessage == nil {
		t.Fatalf("expected nested DiscordMessage to be converted")
	}

	if conv.DiscordMessage.ID != entDiscordMessage.ID {
		t.Fatalf("DiscordMessage ID mismatch: got=%s want=%s", conv.DiscordMessage.ID, entDiscordMessage.ID)
	}

	if conv.ConversationGroup == nil {
		t.Fatalf("expected nested ConversationGroup to be converted")
	}

	if conv.ConversationGroup.ID != entConversationGroup.ID {
		t.Fatalf("ConversationGroup ID mismatch: got=%d want=%d", conv.ConversationGroup.ID, entConversationGroup.ID)
	}

	if !conv.CreatedAt.Equal(entConversation.CreatedAt) {
		t.Fatalf("CreatedAt mismatch: got=%v want=%v", conv.CreatedAt, entConversation.CreatedAt)
	}
}

func TestFromEntConversation_Nil(t *testing.T) {
	t.Parallel()

	if domain.FromEntConversation(nil) != nil {
		t.Fatalf("expected nil when input is nil")
	}
}
