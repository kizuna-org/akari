package entity_test

import (
	"testing"
	"time"

	"github.com/kizuna-org/akari/internal/message/domain/entity"
	databaseDomain "github.com/kizuna-org/akari/pkg/database/domain"
	"github.com/stretchr/testify/require"
)

func TestToDiscordChannel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		channel *databaseDomain.DiscordChannel
		want    *entity.DiscordChannel
	}{
		{
			name:    "nil channel",
			channel: nil,
			want:    nil,
		},
		{
			name: "valid channel",
			channel: &databaseDomain.DiscordChannel{
				ID:        "ch-123",
				Type:      "0",
				Name:      "general",
				GuildID:   "g-456",
				CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			want: &entity.DiscordChannel{
				ID:        "ch-123",
				Type:      "0",
				Name:      "general",
				GuildID:   "g-456",
				CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "empty channel",
			channel: &databaseDomain.DiscordChannel{
				ID:   "",
				Type: "",
				Name: "",
			},
			want: &entity.DiscordChannel{
				ID:   "",
				Type: "",
				Name: "",
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := entity.ToDiscordChannel(testCase.channel)
			require.Equal(t, testCase.want, got)
		})
	}
}

func TestDiscordChannelToDatabaseDiscordChannel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ch   *entity.DiscordChannel
		want databaseDomain.DiscordChannel
	}{
		{
			name: "convert channel",
			ch: &entity.DiscordChannel{
				ID:        "ch-123",
				Type:      "0",
				Name:      "general",
				GuildID:   "g-456",
				CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			want: databaseDomain.DiscordChannel{
				ID:        "ch-123",
				Type:      "0",
				Name:      "general",
				GuildID:   "g-456",
				CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "channel with type 1",
			ch: &entity.DiscordChannel{
				ID:   "ch-789",
				Type: "1",
				Name: "dm",
			},
			want: databaseDomain.DiscordChannel{
				ID:   "ch-789",
				Type: "1",
				Name: "dm",
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := testCase.ch.ToDatabaseDiscordChannel()
			require.Equal(t, testCase.want.ID, got.ID)
			require.Equal(t, testCase.want.Type, got.Type)
			require.Equal(t, testCase.want.Name, got.Name)
			require.Equal(t, testCase.want.GuildID, got.GuildID)
			require.Equal(t, testCase.want.CreatedAt, got.CreatedAt)
		})
	}
}
