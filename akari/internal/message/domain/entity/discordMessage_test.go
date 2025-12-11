package entity_test

import (
	"testing"
	"time"

	"github.com/kizuna-org/akari/internal/message/domain/entity"
	databaseDomain "github.com/kizuna-org/akari/pkg/database/domain"
	discordEntity "github.com/kizuna-org/akari/pkg/discord/domain/entity"
	"github.com/stretchr/testify/require"
)

func TestToDiscordMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		msg  *discordEntity.Message
		want *entity.DiscordMessage
	}{
		{
			name: "nil message",
			msg:  nil,
			want: nil,
		},
		{
			name: "valid message",
			msg: &discordEntity.Message{
				ID:        "123",
				ChannelID: "ch-456",
				GuildID:   "g-789",
				AuthorID:  "au-123",
				Content:   "hello",
				Timestamp: time.Date(2025, 12, 10, 12, 0, 0, 0, time.UTC),
				IsBot:     true,
				Mentions:  []string{"user1", "user2"},
			},
			want: &entity.DiscordMessage{
				ID:        "123",
				ChannelID: "ch-456",
				GuildID:   "g-789",
				AuthorID:  "au-123",
				Content:   "hello",
				Timestamp: time.Date(2025, 12, 10, 12, 0, 0, 0, time.UTC),
				IsBot:     true,
				Mentions:  []string{"user1", "user2"},
			},
		},
		{
			name: "empty message",
			msg: &discordEntity.Message{
				ID:       "",
				Content:  "",
				Mentions: nil,
			},
			want: &entity.DiscordMessage{
				ID:       "",
				Content:  "",
				Mentions: nil,
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := entity.ToDiscordMessage(testCase.msg)
			require.Equal(t, testCase.want, got)
		})
	}
}

func TestDiscordMessageToDatabaseDiscordMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		msg  *entity.DiscordMessage
		want databaseDomain.DiscordMessage
	}{
		{
			name: "convert message",
			msg: &entity.DiscordMessage{
				ID:        "msg-123",
				ChannelID: "ch-456",
				AuthorID:  "au-789",
				Content:   "test content",
				Timestamp: time.Date(2025, 12, 10, 12, 0, 0, 0, time.UTC),
				IsBot:     false,
			},
			want: databaseDomain.DiscordMessage{
				ID:        "msg-123",
				ChannelID: "ch-456",
				AuthorID:  "au-789",
				Content:   "test content",
				Timestamp: time.Date(2025, 12, 10, 12, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "empty message",
			msg: &entity.DiscordMessage{
				ID: "",
			},
			want: databaseDomain.DiscordMessage{
				ID: "",
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := testCase.msg.ToDatabaseDiscordMessage()
			require.Equal(t, testCase.want.ID, got.ID)
			require.Equal(t, testCase.want.ChannelID, got.ChannelID)
			require.Equal(t, testCase.want.AuthorID, got.AuthorID)
			require.Equal(t, testCase.want.Content, got.Content)
			require.Equal(t, testCase.want.Timestamp, got.Timestamp)
		})
	}
}
