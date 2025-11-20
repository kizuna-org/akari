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
	if err == nil || discordMessage != nil {
		t.Fatalf("expected error for missing edges; got err=%v dm=%v", err, discordMessage)
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

	missingChannel := &ent.DiscordMessage{
		ID:        "m2",
		Content:   "c",
		Timestamp: now,
		CreatedAt: now,
		Edges:     ent.DiscordMessageEdges{Channel: nil, Author: entAuthor},
	}

	missingAuthor := &ent.DiscordMessage{
		ID:        "m3",
		Content:   "c",
		Timestamp: now,
		CreatedAt: now,
		Edges:     ent.DiscordMessageEdges{Channel: entChannel, Author: nil},
	}

	tests := []struct {
		name    string
		input   *ent.DiscordMessage
		wantErr bool
	}{
		{name: "valid message", input: entDiscordMessage, wantErr: false},
		{name: "missing channel edge", input: missingChannel, wantErr: true},
		{name: "missing author edge", input: missingAuthor, wantErr: true},
		{name: "nil input", input: nil, wantErr: true},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, err := domain.FromEntDiscordMessage(testCase.input)
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

			if got.ChannelID != testCase.input.Edges.Channel.ID {
				t.Fatalf("Channel ID mismatch: got=%s want id=%s", got.ChannelID, testCase.input.Edges.Channel.ID)
			}

			if got.AuthorID != testCase.input.Edges.Author.ID {
				t.Fatalf("Author ID mismatch: got=%s want=%s", got.AuthorID, testCase.input.Edges.Author.ID)
			}
		})
	}
}
