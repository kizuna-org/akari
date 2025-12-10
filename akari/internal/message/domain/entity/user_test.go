package entity_test

import (
	"testing"

	"github.com/kizuna-org/akari/internal/message/domain/entity"
	databaseDomain "github.com/kizuna-org/akari/pkg/database/domain"
	discordEntity "github.com/kizuna-org/akari/pkg/discord/domain/entity"
	"github.com/stretchr/testify/require"
)

func TestToUser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		user *discordEntity.User
		want *entity.User
	}{
		{
			name: "nil user",
			user: nil,
			want: nil,
		},
		{
			name: "valid user",
			user: &discordEntity.User{ID: "g-123", Username: "testuser", Bot: false},
			want: &entity.User{ID: "g-123", Username: "testuser", Bot: false},
		},
		{
			name: "empty user",
			user: &discordEntity.User{ID: ""},
			want: &entity.User{ID: ""},
		},
		{
			name: "bot user",
			user: &discordEntity.User{ID: "g-456", Username: "botname", Bot: true},
			want: &entity.User{ID: "g-456", Username: "botname", Bot: true},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := entity.ToUser(testCase.user)
			require.Equal(t, testCase.want, got)
		})
	}
}

func TestUserToDatabaseUser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		g    *entity.User
		want databaseDomain.DiscordUser
	}{
		{
			name: "convert user",
			g:    &entity.User{ID: "g-123", Username: "testuser", Bot: false},
			want: databaseDomain.DiscordUser{
				ID:       "g-123",
				Username: "testuser",
				Bot:      false,
			},
		},
		{
			name: "user with empty ID",
			g:    &entity.User{ID: ""},
			want: databaseDomain.DiscordUser{ID: ""},
		},
		{
			name: "user with username and bot flag",
			g:    &entity.User{ID: "g-456", Username: "botuser", Bot: true},
			want: databaseDomain.DiscordUser{
				ID:       "g-456",
				Username: "botuser",
				Bot:      true,
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := testCase.g.ToDatabaseUser()
			require.Equal(t, testCase.want.ID, got.ID)
			require.Equal(t, testCase.want.Username, got.Username)
			require.Equal(t, testCase.want.Bot, got.Bot)
			require.NotZero(t, got.CreatedAt)
			require.NotZero(t, got.UpdatedAt)
		})
	}
}
