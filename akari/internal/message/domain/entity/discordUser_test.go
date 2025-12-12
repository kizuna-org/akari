package entity_test

import (
	"testing"

	"github.com/kizuna-org/akari/internal/message/domain/entity"
	databaseDomain "github.com/kizuna-org/akari/pkg/database/domain"
	"github.com/stretchr/testify/require"
)

func TestToDiscordUser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		user *databaseDomain.DiscordUser
		want *entity.DiscordUser
	}{
		{
			name: "nil user",
			user: nil,
			want: nil,
		},
		{
			name: "valid user",
			user: &databaseDomain.DiscordUser{ID: "g-123", Username: "testuser", Bot: false},
			want: &entity.DiscordUser{ID: "g-123", Username: "testuser", Bot: false},
		},
		{
			name: "empty user",
			user: &databaseDomain.DiscordUser{ID: ""},
			want: &entity.DiscordUser{ID: ""},
		},
		{
			name: "bot user",
			user: &databaseDomain.DiscordUser{ID: "g-456", Username: "botname", Bot: true},
			want: &entity.DiscordUser{ID: "g-456", Username: "botname", Bot: true},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := entity.ToDiscordUser(testCase.user)
			require.Equal(t, testCase.want, got)
		})
	}
}

func TestDiscordUserToDatabaseDiscordUser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		g    *entity.DiscordUser
		want databaseDomain.DiscordUser
	}{
		{
			name: "convert user",
			g:    &entity.DiscordUser{ID: "g-123", Username: "testuser", Bot: false},
			want: databaseDomain.DiscordUser{
				ID:       "g-123",
				Username: "testuser",
				Bot:      false,
			},
		},
		{
			name: "user with empty ID",
			g:    &entity.DiscordUser{ID: ""},
			want: databaseDomain.DiscordUser{ID: ""},
		},
		{
			name: "user with username and bot flag",
			g:    &entity.DiscordUser{ID: "g-456", Username: "botuser", Bot: true},
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

			got := testCase.g.ToDatabaseDiscordUser()
			require.Equal(t, testCase.want.ID, got.ID)
			require.Equal(t, testCase.want.Username, got.Username)
			require.Equal(t, testCase.want.Bot, got.Bot)
			require.NotZero(t, got.CreatedAt)
			require.NotZero(t, got.UpdatedAt)
		})
	}
}
