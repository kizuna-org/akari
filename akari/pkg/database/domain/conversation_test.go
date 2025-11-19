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

	valid := &ent.Conversation{
		ID:        1,
		CreatedAt: now,
		Edges: ent.ConversationEdges{
			User:              entUser,
			DiscordMessage:    entDiscordMessage,
			ConversationGroup: entConversationGroup,
		},
	}

	missingUser := &ent.Conversation{
		ID:        2,
		CreatedAt: now,
		Edges: ent.ConversationEdges{
			User:              nil,
			DiscordMessage:    entDiscordMessage,
			ConversationGroup: entConversationGroup,
		},
	}

	missingMessage := &ent.Conversation{
		ID:        3,
		CreatedAt: now,
		Edges: ent.ConversationEdges{
			User:              entUser,
			DiscordMessage:    nil,
			ConversationGroup: entConversationGroup,
		},
	}

	missingGroup := &ent.Conversation{
		ID:        4,
		CreatedAt: now,
		Edges: ent.ConversationEdges{
			User:              entUser,
			DiscordMessage:    entDiscordMessage,
			ConversationGroup: nil,
		},
	}

	tests := []struct {
		name    string
		input   *ent.Conversation
		wantErr bool
	}{
		{name: "valid conversation", input: valid, wantErr: false},
		{name: "nil input", input: nil, wantErr: true},
		{name: "missing user edge", input: missingUser, wantErr: true},
		{name: "missing discord message edge", input: missingMessage, wantErr: true},
		{name: "missing conversation group edge", input: missingGroup, wantErr: true},
	}

	runConversationCases(t, tests)
}

func runConversationCases(t *testing.T, tests []struct {
	name    string
	input   *ent.Conversation
	wantErr bool
}) {
	t.Helper()

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, err := domain.FromEntConversation(testCase.input)
			if (err != nil) != testCase.wantErr {
				t.Fatalf("unexpected error state: %v", err)
			}

			if testCase.wantErr {
				if got != nil {
					t.Fatalf("expected nil on error, got: %+v", got)
				}

				return
			}

			if got == nil {
				t.Fatalf("expected non-nil result")
			}

			validateConversationResult(t, got, testCase.input)
		})
	}
}

func validateConversationResult(t *testing.T, got *domain.Conversation, want *ent.Conversation) {
	t.Helper()

	if got.ID != want.ID {
		t.Fatalf("ID mismatch: got=%d want=%d", got.ID, want.ID)
	}

	if got.UserID != want.Edges.User.ID {
		t.Fatalf("User ID mismatch: got=%d want=%d", got.UserID, want.Edges.User.ID)
	}

	if got.DiscordMessageID != want.Edges.DiscordMessage.ID {
		t.Fatalf("DiscordMessage ID mismatch: got=%s want=%s", got.DiscordMessageID, want.Edges.DiscordMessage.ID)
	}

	if got.ConversationGroupID != want.Edges.ConversationGroup.ID {
		expected := want.Edges.ConversationGroup.ID
		t.Fatalf("ConversationGroup ID mismatch: got=%d want=%d", got.ConversationGroupID, expected)
	}

	if !got.CreatedAt.Equal(want.CreatedAt) {
		t.Fatalf("CreatedAt mismatch: got=%v want=%v", got.CreatedAt, want.CreatedAt)
	}
}
