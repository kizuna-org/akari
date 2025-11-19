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

	discordMessage := domain.FromEntDiscordMessage(entDiscordMessage)
	if discordMessage == nil {
		t.Fatalf("expected non-nil domain message")
	}

	if discordMessage.Channel != nil {
		t.Fatalf("expected nil Channel when edges missing, got %+v", discordMessage.Channel)
	}

	if discordMessage.Author != nil {
		t.Fatalf("expected nil Author when edges missing, got %+v", discordMessage.Author)
	}
}

func TestFromEntDiscordMessage_Nil(t *testing.T) {
	t.Parallel()

	if domain.FromEntDiscordMessage(nil) != nil {
		t.Fatalf("expected nil when input is nil")
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

	discordMessage := domain.FromEntDiscordMessage(entDiscordMessage)
	if discordMessage == nil {
		t.Fatalf("expected non-nil domain message")
	}

	if discordMessage.Channel == nil {
		t.Fatalf("expected Channel to be converted")
	}

	if discordMessage.Channel.ID != entChannel.ID || discordMessage.Channel.Name != entChannel.Name {
		t.Fatalf("Channel fields mismatch: got=%+v want id=%s name=%s",
			discordMessage.Channel, entChannel.ID, entChannel.Name,
		)
	}

	if discordMessage.Author == nil {
		t.Fatalf("expected Author to be converted")
	}

	if discordMessage.Author.ID != entAuthor.ID {
		t.Fatalf("Author ID mismatch: got=%s want=%s", discordMessage.Author.ID, entAuthor.ID)
	}
}
