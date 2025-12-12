package entity_test

import (
	"testing"
	"time"

	"github.com/kizuna-org/akari/internal/message/domain/entity"
	databaseDomain "github.com/kizuna-org/akari/pkg/database/domain"
	"github.com/stretchr/testify/require"
)

func TestToDiscordGuild(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		guild *databaseDomain.DiscordGuild
		want  *entity.DiscordGuild
	}{
		{
			name:  "nil guild",
			guild: nil,
			want:  nil,
		},
		{
			name: "valid guild",
			guild: &databaseDomain.DiscordGuild{
				ID:        "g-123",
				Name:      "test guild",
				CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			want: &entity.DiscordGuild{
				ID:        "g-123",
				Name:      "test guild",
				CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "empty guild",
			guild: &databaseDomain.DiscordGuild{
				ID:   "",
				Name: "",
			},
			want: &entity.DiscordGuild{
				ID:   "",
				Name: "",
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := entity.ToDiscordGuild(testCase.guild)
			require.Equal(t, testCase.want, got)
		})
	}
}

func TestDiscordGuildToDatabaseDiscordGuild(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		g    *entity.DiscordGuild
		want databaseDomain.DiscordGuild
	}{
		{
			name: "convert guild",
			g: &entity.DiscordGuild{
				ID:        "g-123",
				Name:      "test guild",
				CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			want: databaseDomain.DiscordGuild{
				ID:         "g-123",
				Name:       "test guild",
				ChannelIDs: []string{},
				CreatedAt:  time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "guild with empty name",
			g: &entity.DiscordGuild{
				ID:   "g-789",
				Name: "",
			},
			want: databaseDomain.DiscordGuild{
				ID:         "g-789",
				Name:       "",
				ChannelIDs: []string{},
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := testCase.g.ToDatabaseDiscordGuild()
			require.Equal(t, testCase.want.ID, got.ID)
			require.Equal(t, testCase.want.Name, got.Name)
			require.Equal(t, testCase.want.ChannelIDs, got.ChannelIDs)
			require.Equal(t, testCase.want.CreatedAt, got.CreatedAt)
		})
	}
}
