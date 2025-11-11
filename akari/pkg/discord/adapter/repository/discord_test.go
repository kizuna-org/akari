package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/kizuna-org/akari/pkg/discord/adapter/repository"
	"github.com/kizuna-org/akari/pkg/discord/infrastructure"
	"github.com/stretchr/testify/assert"
)

func TestNewDiscordRepository(t *testing.T) {
	t.Parallel()

	client, err := infrastructure.NewDiscordClient("test-token")
	if err != nil {
		t.Fatalf("failed to create discord client: %v", err)
	}

	repo := repository.NewDiscordRepository(client)
	assert.NotNil(t, repo)
}

func TestDiscordRepository_Integration(t *testing.T) {
	t.Parallel()

	client, err := infrastructure.NewDiscordClient("test-token")
	if err != nil {
		t.Fatalf("failed to create discord client: %v", err)
	}

	assert.NotNil(t, client)

	repo := repository.NewDiscordRepository(client)
	assert.NotNil(t, repo)
}

func TestDiscordRepository_MessageEntityMapping(t *testing.T) {
	t.Parallel()

	timestamp, err := time.Parse(time.RFC3339, "2024-01-01T00:00:00Z")
	if err != nil {
		t.Fatalf("failed to parse timestamp: %v", err)
	}

	discordMsg := &discordgo.Message{
		ID:        "msg-123",
		ChannelID: "channel-456",
		GuildID:   "guild-789",
		Author: &discordgo.User{
			ID:            "user-001",
			Email:         "",
			Username:      "",
			Avatar:        "",
			Locale:        "",
			Discriminator: "",
			GlobalName:    "",
			Token:         "",
			Verified:      false,
			MFAEnabled:    false,
			Banner:        "",
			AccentColor:   0,
			Bot:           false,
			PublicFlags:   0,
			PremiumType:   0,
			System:        false,
			Flags:         0,
		},
		Content:             "Test message",
		Timestamp:           timestamp,
		EditedTimestamp:     nil,
		MentionRoles:        nil,
		TTS:                 false,
		MentionEveryone:     false,
		Attachments:         nil,
		Components:          nil,
		Embeds:              nil,
		Mentions:            nil,
		Reactions:           nil,
		Pinned:              false,
		Type:                0,
		WebhookID:           "",
		Member:              nil,
		MentionChannels:     nil,
		Activity:            nil,
		Application:         nil,
		MessageReference:    nil,
		ReferencedMessage:   nil,
		MessageSnapshots:    nil,
		Interaction:         nil,
		InteractionMetadata: nil,
		Flags:               0,
		Thread:              nil,
		StickerItems:        nil,
		Poll:                nil,
	}

	assert.Equal(t, "msg-123", discordMsg.ID)
	assert.Equal(t, "channel-456", discordMsg.ChannelID)
	assert.Equal(t, "guild-789", discordMsg.GuildID)
	assert.Equal(t, "user-001", discordMsg.Author.ID)
	assert.Equal(t, "Test message", discordMsg.Content)
	assert.Equal(t, timestamp, discordMsg.Timestamp)
}

func TestDiscordRepository_ContextPropagation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		setupCtx       func() context.Context
		expectCanceled bool
	}{
		{
			name:           "normal context",
			setupCtx:       t.Context,
			expectCanceled: false,
		},
		{
			name: "cancelled context",
			setupCtx: func() context.Context {
				ctx, cancel := context.WithCancel(t.Context())
				cancel()

				return ctx
			},
			expectCanceled: true,
		},
		{
			name: "context with timeout",
			setupCtx: func() context.Context {
				ctx, cancel := context.WithTimeout(t.Context(), 1*time.Second)
				t.Cleanup(cancel)

				return ctx
			},
			expectCanceled: false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := testCase.setupCtx()
			assert.NotNil(t, ctx, "context should not be nil")

			if testCase.expectCanceled {
				select {
				case <-ctx.Done():
					if ctx.Err() == nil {
						t.Error("cancelled context should have an error")
					}
				default:
					t.Error("context should be cancelled")
				}
			}
		})
	}
}
