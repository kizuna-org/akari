//go:build integration

package repository_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/kizuna-org/akari/gen/ent"
	"github.com/kizuna-org/akari/pkg/database/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepository_HealthCheck_Integration(t *testing.T) {
	t.Parallel()

	_, repo, _ := setupTestDB(t)

	ctx := context.Background()

	err := repo.HealthCheck(ctx)
	assert.NoError(t, err)
}

func TestRepository_WithTransaction_Integration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		fn      func(ctx context.Context, tx *domain.Tx) error
		wantErr bool
	}{
		{
			name: "success",
			fn: func(ctx context.Context, tx *domain.Tx) error {
				return nil
			},
			wantErr: false,
		},
		{
			name: "failure with rollback",
			fn: func(ctx context.Context, tx *domain.Tx) error {
				return errors.New("transaction error")
			},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			_, repo, _ := setupTestDB(t)

			ctx := context.Background()

			err := repo.WithTransaction(ctx, testCase.fn)

			if testCase.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "transaction error")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRepository_Character_Integration(t *testing.T) {
	t.Parallel()

	_, repo, entClient := setupTestDB(t)
	ctx := context.Background()

	t.Run("GetCharacterByID - not found", func(t *testing.T) {
		t.Parallel()

		_, err := repo.GetCharacterByID(ctx, 99999)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get character")
	})

	t.Run("ListCharacters - empty", func(t *testing.T) {
		t.Parallel()

		characters, err := repo.ListCharacters(ctx)
		require.NoError(t, err)
		assert.Empty(t, characters)
	})

	t.Run("GetCharacterByID and ListCharacters - with data", func(t *testing.T) {
		t.Parallel()

		gofakeit.Seed(time.Now().UnixNano())

		// Create CharacterConfig
		config, err := entClient.CharacterConfig.Create().
			SetDefaultSystemPrompt(gofakeit.Sentence(10)).
			Save(ctx)
		require.NoError(t, err)

		// Create SystemPrompt
		systemPrompt, err := entClient.SystemPrompt.Create().
			SetTitle(gofakeit.Word()).
			SetPurpose("text_chat").
			SetPrompt(gofakeit.Paragraph(3, 5, 10, "\n")).
			Save(ctx)
		require.NoError(t, err)

		// Create Character with Config and SystemPrompt
		character, err := entClient.Character.Create().
			SetName(gofakeit.Name()).
			SetConfig(config).
			AddSystemPrompts(systemPrompt).
			Save(ctx)
		require.NoError(t, err)

		// Test GetCharacterByID
		got, err := repo.GetCharacterByID(ctx, character.ID)
		require.NoError(t, err)
		assert.Equal(t, character.ID, got.ID)
		assert.Equal(t, character.Name, got.Name)
		assert.NotNil(t, got.Config)
		assert.Len(t, got.SystemPromptIDs, 1)
		assert.Equal(t, systemPrompt.ID, got.SystemPromptIDs[0])

		// Test ListCharacters
		characters, err := repo.ListCharacters(ctx)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(characters), 1)

		found := false
		for _, c := range characters {
			if c.ID == character.ID {
				found = true
				assert.Equal(t, character.Name, c.Name)
				break
			}
		}
		assert.True(t, found, "created character should be in the list")
	})
}

func TestRepository_AkariUser_Integration(t *testing.T) {
	t.Parallel()

	_, repo, entClient := setupTestDB(t)
	ctx := context.Background()

	t.Run("CreateAkariUser", func(t *testing.T) {
		t.Parallel()

		user, err := repo.CreateAkariUser(ctx)
		require.NoError(t, err)
		assert.Greater(t, user.ID, 0)
		assert.NotZero(t, user.CreatedAt)
		assert.NotZero(t, user.UpdatedAt)
	})

	t.Run("GetAkariUserByID", func(t *testing.T) {
		t.Parallel()

		created, err := repo.CreateAkariUser(ctx)
		require.NoError(t, err)

		got, err := repo.GetAkariUserByID(ctx, created.ID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, got.ID)
		assert.Equal(t, created.CreatedAt, got.CreatedAt)
	})

	t.Run("GetAkariUserByID - not found", func(t *testing.T) {
		t.Parallel()

		_, err := repo.GetAkariUserByID(ctx, 99999)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get akari user")
	})

	t.Run("GetAkariUserByDiscordUserID", func(t *testing.T) {
		t.Parallel()

		gofakeit.Seed(time.Now().UnixNano())

		// Create DiscordUser
		discordUser, err := entClient.DiscordUser.Create().
			SetID(RandomDiscordID()).
			SetUsername(RandomDiscordUsername()).
			SetBot(gofakeit.Bool()).
			Save(ctx)
		require.NoError(t, err)

		// Create AkariUser with DiscordUser
		akariUser, err := entClient.AkariUser.Create().
			SetDiscordUser(discordUser).
			Save(ctx)
		require.NoError(t, err)

		got, err := repo.GetAkariUserByDiscordUserID(ctx, discordUser.ID)
		require.NoError(t, err)
		assert.Equal(t, akariUser.ID, got.ID)
	})

	t.Run("GetAkariUserByDiscordUserID - not found", func(t *testing.T) {
		t.Parallel()

		_, err := repo.GetAkariUserByDiscordUserID(ctx, RandomDiscordID())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get akari user by discord id")
	})

	t.Run("ListAkariUsers", func(t *testing.T) {
		t.Parallel()

		// Create multiple users
		user1, err := repo.CreateAkariUser(ctx)
		require.NoError(t, err)

		user2, err := repo.CreateAkariUser(ctx)
		require.NoError(t, err)

		users, err := repo.ListAkariUsers(ctx)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(users), 2)

		found1 := false
		found2 := false
		for _, u := range users {
			if u.ID == user1.ID {
				found1 = true
			}
			if u.ID == user2.ID {
				found2 = true
			}
		}
		assert.True(t, found1, "user1 should be in the list")
		assert.True(t, found2, "user2 should be in the list")
	})

	t.Run("DeleteAkariUser", func(t *testing.T) {
		t.Parallel()

		user, err := repo.CreateAkariUser(ctx)
		require.NoError(t, err)

		err = repo.DeleteAkariUser(ctx, user.ID)
		require.NoError(t, err)

		_, err = repo.GetAkariUserByID(ctx, user.ID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get akari user")
	})
}

func TestRepository_DiscordUser_Integration(t *testing.T) {
	t.Parallel()

	_, repo, _ := setupTestDB(t)
	ctx := context.Background()

	t.Run("CreateDiscordUser", func(t *testing.T) {
		t.Parallel()

		params := RandomDiscordUser()
		user, err := repo.CreateDiscordUser(ctx, params)
		require.NoError(t, err)
		assert.Equal(t, params.ID, user.ID)
		assert.Equal(t, params.Username, user.Username)
		assert.Equal(t, params.Bot, user.Bot)
		assert.NotZero(t, user.CreatedAt)
	})

	t.Run("GetDiscordUserByID", func(t *testing.T) {
		t.Parallel()

		params := RandomDiscordUser()
		created, err := repo.CreateDiscordUser(ctx, params)
		require.NoError(t, err)

		got, err := repo.GetDiscordUserByID(ctx, created.ID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, got.ID)
		assert.Equal(t, created.Username, got.Username)
		assert.Equal(t, created.Bot, got.Bot)
	})

	t.Run("GetDiscordUserByID - not found", func(t *testing.T) {
		t.Parallel()

		_, err := repo.GetDiscordUserByID(ctx, RandomDiscordID())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get discord user by id")
	})

	t.Run("ListDiscordUsers", func(t *testing.T) {
		t.Parallel()

		user1 := RandomDiscordUser()
		created1, err := repo.CreateDiscordUser(ctx, user1)
		require.NoError(t, err)

		user2 := RandomDiscordUser()
		created2, err := repo.CreateDiscordUser(ctx, user2)
		require.NoError(t, err)

		users, err := repo.ListDiscordUsers(ctx)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(users), 2)

		found1 := false
		found2 := false
		for _, u := range users {
			if u.ID == created1.ID {
				found1 = true
			}
			if u.ID == created2.ID {
				found2 = true
			}
		}
		assert.True(t, found1, "user1 should be in the list")
		assert.True(t, found2, "user2 should be in the list")
	})

	t.Run("DeleteDiscordUser", func(t *testing.T) {
		t.Parallel()

		params := RandomDiscordUser()
		created, err := repo.CreateDiscordUser(ctx, params)
		require.NoError(t, err)

		err = repo.DeleteDiscordUser(ctx, created.ID)
		require.NoError(t, err)

		_, err = repo.GetDiscordUserByID(ctx, created.ID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get discord user by id")
	})
}
