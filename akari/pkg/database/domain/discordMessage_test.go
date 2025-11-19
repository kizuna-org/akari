package domain_test

import (
	"testing"
	"time"

	"github.com/kizuna-org/akari/gen/ent"
	"github.com/kizuna-org/akari/pkg/database/domain"
)

func TestToDomainDiscordMessage_HandlesNilEdges(t *testing.T) {
	t.Parallel()

	now := time.Now()
	entDiscordMessage := &ent.DiscordMessage{ID: "message-id", Content: "hi", Timestamp: now, CreatedAt: now}

	discordMessage, err := domain.FromEntDiscordMessage(entDiscordMessage)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if discordMessage == nil {
		t.Fatalf("expected non-nil domain message")
	}
}

func TestFromEntDiscordMessage_Nil(t *testing.T) {
	t.Parallel()

	discordMessage, err := domain.FromEntDiscordMessage(nil)
	if err == nil {
		t.Fatalf("expected error when input is nil")
	}

	if discordMessage != nil {
		t.Fatalf("expected nil domain message when input is nil")
	}
}

func TestFromEntDiscordMessage_IncludesChannelAndAuthor(t *testing.T) {
	t.Parallel()

	now := time.Now()
	entChannel := &ent.DiscordChannel{ID: "channel-id", Name: "channel-name", CreatedAt: now}
	entAuthor := &ent.DiscordUser{ID: "u1", CreatedAt: now}
	entDiscordMessage := &ent.DiscordMessage{
		ID: "message-id", Content: "hello", Timestamp: now, CreatedAt: now,
		Edges: ent.DiscordMessageEdges{Channel: entChannel, Author: entAuthor},
	}

	discordMessage, err := domain.FromEntDiscordMessage(entDiscordMessage)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if discordMessage == nil {
		t.Fatalf("expected non-nil domain message")
	}

	if discordMessage.ChannelID != entChannel.ID {
		t.Fatalf("Channel ID mismatch: got=%s want id=%s",
			discordMessage.ChannelID, entChannel.ID,
		)
	}

	if discordMessage.AuthorID != entAuthor.ID {
		t.Fatalf("Author ID mismatch: got=%s want=%s", discordMessage.AuthorID, entAuthor.ID)
	}
}
