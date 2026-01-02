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

func TestRepository_DiscordGuild_Integration(t *testing.T) {
	t.Parallel()

	_, repo, _ := setupTestDB(t)
	ctx := context.Background()

	t.Run("CreateDiscordGuild", func(t *testing.T) {
		t.Parallel()

		params := RandomDiscordGuild()
		guild, err := repo.CreateDiscordGuild(ctx, params)
		require.NoError(t, err)
		assert.Equal(t, params.ID, guild.ID)
		assert.Equal(t, params.Name, guild.Name)
	})

	t.Run("GetDiscordGuildByID", func(t *testing.T) {
		t.Parallel()

		params := RandomDiscordGuild()
		created, err := repo.CreateDiscordGuild(ctx, params)
		require.NoError(t, err)

		got, err := repo.GetDiscordGuildByID(ctx, created.ID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, got.ID)
		assert.Equal(t, created.Name, got.Name)
	})

	t.Run("GetDiscordGuildByID - not found", func(t *testing.T) {
		t.Parallel()

		_, err := repo.GetDiscordGuildByID(ctx, RandomDiscordID())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get discord guild by id")
	})

	t.Run("ListDiscordGuilds", func(t *testing.T) {
		t.Parallel()

		guild1 := RandomDiscordGuild()
		created1, err := repo.CreateDiscordGuild(ctx, guild1)
		require.NoError(t, err)

		guild2 := RandomDiscordGuild()
		created2, err := repo.CreateDiscordGuild(ctx, guild2)
		require.NoError(t, err)

		guilds, err := repo.ListDiscordGuilds(ctx)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(guilds), 2)

		found1 := false
		found2 := false
		for _, g := range guilds {
			if g.ID == created1.ID {
				found1 = true
			}
			if g.ID == created2.ID {
				found2 = true
			}
		}
		assert.True(t, found1, "guild1 should be in the list")
		assert.True(t, found2, "guild2 should be in the list")
	})

	t.Run("DeleteDiscordGuild", func(t *testing.T) {
		t.Parallel()

		params := RandomDiscordGuild()
		created, err := repo.CreateDiscordGuild(ctx, params)
		require.NoError(t, err)

		err = repo.DeleteDiscordGuild(ctx, created.ID)
		require.NoError(t, err)

		_, err = repo.GetDiscordGuildByID(ctx, created.ID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get discord guild by id")
	})
}

func TestRepository_DiscordChannel_Integration(t *testing.T) {
	t.Parallel()

	_, repo, _ := setupTestDB(t)
	ctx := context.Background()

	t.Run("CreateDiscordChannel", func(t *testing.T) {
		t.Parallel()

		guild := RandomDiscordGuild()
		createdGuild, err := repo.CreateDiscordGuild(ctx, guild)
		require.NoError(t, err)

		params := RandomDiscordChannel(createdGuild.ID)
		channel, err := repo.CreateDiscordChannel(ctx, params)
		require.NoError(t, err)
		assert.Equal(t, params.ID, channel.ID)
		assert.Equal(t, params.Type, channel.Type)
		assert.Equal(t, params.Name, channel.Name)
		assert.Equal(t, params.GuildID, channel.GuildID)
	})

	t.Run("GetDiscordChannelByID", func(t *testing.T) {
		t.Parallel()

		guild := RandomDiscordGuild()
		createdGuild, err := repo.CreateDiscordGuild(ctx, guild)
		require.NoError(t, err)

		params := RandomDiscordChannel(createdGuild.ID)
		created, err := repo.CreateDiscordChannel(ctx, params)
		require.NoError(t, err)

		got, err := repo.GetDiscordChannelByID(ctx, created.ID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, got.ID)
		assert.Equal(t, created.Type, got.Type)
		assert.Equal(t, created.Name, got.Name)
		assert.Equal(t, created.GuildID, got.GuildID)
	})

	t.Run("GetDiscordChannelByID - not found", func(t *testing.T) {
		t.Parallel()

		_, err := repo.GetDiscordChannelByID(ctx, RandomDiscordID())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get discord channel by id")
	})

	t.Run("GetDiscordChannelsByGuildID", func(t *testing.T) {
		t.Parallel()

		guild := RandomDiscordGuild()
		createdGuild, err := repo.CreateDiscordGuild(ctx, guild)
		require.NoError(t, err)

		channel1 := RandomDiscordChannel(createdGuild.ID)
		created1, err := repo.CreateDiscordChannel(ctx, channel1)
		require.NoError(t, err)

		channel2 := RandomDiscordChannel(createdGuild.ID)
		created2, err := repo.CreateDiscordChannel(ctx, channel2)
		require.NoError(t, err)

		channels, err := repo.GetDiscordChannelsByGuildID(ctx, createdGuild.ID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(channels), 2)

		found1 := false
		found2 := false
		for _, c := range channels {
			if c.ID == created1.ID {
				found1 = true
			}
			if c.ID == created2.ID {
				found2 = true
			}
		}
		assert.True(t, found1, "channel1 should be in the list")
		assert.True(t, found2, "channel2 should be in the list")
	})

	t.Run("DeleteDiscordChannel", func(t *testing.T) {
		t.Parallel()

		guild := RandomDiscordGuild()
		createdGuild, err := repo.CreateDiscordGuild(ctx, guild)
		require.NoError(t, err)

		params := RandomDiscordChannel(createdGuild.ID)
		created, err := repo.CreateDiscordChannel(ctx, params)
		require.NoError(t, err)

		err = repo.DeleteDiscordChannel(ctx, created.ID)
		require.NoError(t, err)

		_, err = repo.GetDiscordChannelByID(ctx, created.ID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get discord channel by id")
	})
}

func TestRepository_DiscordMessage_Integration(t *testing.T) {
	t.Parallel()

	_, repo, _ := setupTestDB(t)
	ctx := context.Background()

	t.Run("CreateDiscordMessage", func(t *testing.T) {
		t.Parallel()

		// Setup dependencies
		discordUser := RandomDiscordUser()
		createdUser, err := repo.CreateDiscordUser(ctx, discordUser)
		require.NoError(t, err)

		guild := RandomDiscordGuild()
		createdGuild, err := repo.CreateDiscordGuild(ctx, guild)
		require.NoError(t, err)

		channel := RandomDiscordChannel(createdGuild.ID)
		createdChannel, err := repo.CreateDiscordChannel(ctx, channel)
		require.NoError(t, err)

		params := RandomDiscordMessage(createdUser.ID, createdChannel.ID)
		message, err := repo.CreateDiscordMessage(ctx, params)
		require.NoError(t, err)
		assert.Equal(t, params.ID, message.ID)
		assert.Equal(t, params.AuthorID, message.AuthorID)
		assert.Equal(t, params.ChannelID, message.ChannelID)
		assert.Equal(t, params.Content, message.Content)
	})

	t.Run("GetDiscordMessageByID", func(t *testing.T) {
		t.Parallel()

		// Setup dependencies
		discordUser := RandomDiscordUser()
		createdUser, err := repo.CreateDiscordUser(ctx, discordUser)
		require.NoError(t, err)

		guild := RandomDiscordGuild()
		createdGuild, err := repo.CreateDiscordGuild(ctx, guild)
		require.NoError(t, err)

		channel := RandomDiscordChannel(createdGuild.ID)
		createdChannel, err := repo.CreateDiscordChannel(ctx, channel)
		require.NoError(t, err)

		params := RandomDiscordMessage(createdUser.ID, createdChannel.ID)
		created, err := repo.CreateDiscordMessage(ctx, params)
		require.NoError(t, err)

		got, err := repo.GetDiscordMessageByID(ctx, created.ID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, got.ID)
		assert.Equal(t, created.AuthorID, got.AuthorID)
		assert.Equal(t, created.ChannelID, got.ChannelID)
		assert.Equal(t, created.Content, got.Content)
	})

	t.Run("GetDiscordMessageByID - not found", func(t *testing.T) {
		t.Parallel()

		_, err := repo.GetDiscordMessageByID(ctx, RandomDiscordID())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get discord message by id")
	})

	t.Run("DeleteDiscordMessage", func(t *testing.T) {
		t.Parallel()

		// Setup dependencies
		discordUser := RandomDiscordUser()
		createdUser, err := repo.CreateDiscordUser(ctx, discordUser)
		require.NoError(t, err)

		guild := RandomDiscordGuild()
		createdGuild, err := repo.CreateDiscordGuild(ctx, guild)
		require.NoError(t, err)

		channel := RandomDiscordChannel(createdGuild.ID)
		createdChannel, err := repo.CreateDiscordChannel(ctx, channel)
		require.NoError(t, err)

		params := RandomDiscordMessage(createdUser.ID, createdChannel.ID)
		created, err := repo.CreateDiscordMessage(ctx, params)
		require.NoError(t, err)

		err = repo.DeleteDiscordMessage(ctx, created.ID)
		require.NoError(t, err)

		_, err = repo.GetDiscordMessageByID(ctx, created.ID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get discord message by id")
	})
}

func TestRepository_ConversationGroup_Integration(t *testing.T) {
	t.Parallel()

	_, repo, entClient := setupTestDB(t)
	ctx := context.Background()

	t.Run("CreateConversationGroup", func(t *testing.T) {
		t.Parallel()

		gofakeit.Seed(time.Now().UnixNano())

		// Create Character with Config
		config, err := entClient.CharacterConfig.Create().
			SetDefaultSystemPrompt(gofakeit.Sentence(10)).
			Save(ctx)
		require.NoError(t, err)

		systemPrompt, err := entClient.SystemPrompt.Create().
			SetTitle(gofakeit.Word()).
			SetPurpose("text_chat").
			SetPrompt(gofakeit.Paragraph(3, 5, 10, "\n")).
			Save(ctx)
		require.NoError(t, err)

		character, err := entClient.Character.Create().
			SetName(gofakeit.Name()).
			SetConfig(config).
			AddSystemPrompts(systemPrompt).
			Save(ctx)
		require.NoError(t, err)

		group, err := repo.CreateConversationGroup(ctx, character.ID)
		require.NoError(t, err)
		assert.Greater(t, group.ID, 0)
		assert.Equal(t, character.ID, group.CharacterID)
		assert.NotZero(t, group.CreatedAt)
	})

	t.Run("GetConversationGroupByID", func(t *testing.T) {
		t.Parallel()

		gofakeit.Seed(time.Now().UnixNano())

		config, err := entClient.CharacterConfig.Create().
			SetDefaultSystemPrompt(gofakeit.Sentence(10)).
			Save(ctx)
		require.NoError(t, err)

		systemPrompt, err := entClient.SystemPrompt.Create().
			SetTitle(gofakeit.Word()).
			SetPurpose("text_chat").
			SetPrompt(gofakeit.Paragraph(3, 5, 10, "\n")).
			Save(ctx)
		require.NoError(t, err)

		character, err := entClient.Character.Create().
			SetName(gofakeit.Name()).
			SetConfig(config).
			AddSystemPrompts(systemPrompt).
			Save(ctx)
		require.NoError(t, err)

		created, err := repo.CreateConversationGroup(ctx, character.ID)
		require.NoError(t, err)

		got, err := repo.GetConversationGroupByID(ctx, created.ID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, got.ID)
		assert.Equal(t, created.CharacterID, got.CharacterID)
	})

	t.Run("GetConversationGroupByID - not found", func(t *testing.T) {
		t.Parallel()

		_, err := repo.GetConversationGroupByID(ctx, 99999)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get conversation group by id")
	})

	t.Run("ListConversationGroups", func(t *testing.T) {
		t.Parallel()

		gofakeit.Seed(time.Now().UnixNano())

		config1, err := entClient.CharacterConfig.Create().
			SetDefaultSystemPrompt(gofakeit.Sentence(10)).
			Save(ctx)
		require.NoError(t, err)

		systemPrompt1, err := entClient.SystemPrompt.Create().
			SetTitle(gofakeit.Word()).
			SetPurpose("text_chat").
			SetPrompt(gofakeit.Paragraph(3, 5, 10, "\n")).
			Save(ctx)
		require.NoError(t, err)

		character1, err := entClient.Character.Create().
			SetName(gofakeit.Name()).
			SetConfig(config1).
			AddSystemPrompts(systemPrompt1).
			Save(ctx)
		require.NoError(t, err)

		group1, err := repo.CreateConversationGroup(ctx, character1.ID)
		require.NoError(t, err)

		config2, err := entClient.CharacterConfig.Create().
			SetDefaultSystemPrompt(gofakeit.Sentence(10)).
			Save(ctx)
		require.NoError(t, err)

		systemPrompt2, err := entClient.SystemPrompt.Create().
			SetTitle(gofakeit.Word()).
			SetPurpose("text_chat").
			SetPrompt(gofakeit.Paragraph(3, 5, 10, "\n")).
			Save(ctx)
		require.NoError(t, err)

		character2, err := entClient.Character.Create().
			SetName(gofakeit.Name()).
			SetConfig(config2).
			AddSystemPrompts(systemPrompt2).
			Save(ctx)
		require.NoError(t, err)

		group2, err := repo.CreateConversationGroup(ctx, character2.ID)
		require.NoError(t, err)

		groups, err := repo.ListConversationGroups(ctx)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(groups), 2)

		found1 := false
		found2 := false
		for _, g := range groups {
			if g.ID == group1.ID {
				found1 = true
			}
			if g.ID == group2.ID {
				found2 = true
			}
		}
		assert.True(t, found1, "group1 should be in the list")
		assert.True(t, found2, "group2 should be in the list")
	})

	t.Run("DeleteConversationGroup", func(t *testing.T) {
		t.Parallel()

		gofakeit.Seed(time.Now().UnixNano())

		config, err := entClient.CharacterConfig.Create().
			SetDefaultSystemPrompt(gofakeit.Sentence(10)).
			Save(ctx)
		require.NoError(t, err)

		systemPrompt, err := entClient.SystemPrompt.Create().
			SetTitle(gofakeit.Word()).
			SetPurpose("text_chat").
			SetPrompt(gofakeit.Paragraph(3, 5, 10, "\n")).
			Save(ctx)
		require.NoError(t, err)

		character, err := entClient.Character.Create().
			SetName(gofakeit.Name()).
			SetConfig(config).
			AddSystemPrompts(systemPrompt).
			Save(ctx)
		require.NoError(t, err)

		created, err := repo.CreateConversationGroup(ctx, character.ID)
		require.NoError(t, err)

		err = repo.DeleteConversationGroup(ctx, created.ID)
		require.NoError(t, err)

		_, err = repo.GetConversationGroupByID(ctx, created.ID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get conversation group by id")
	})
}

func TestRepository_Conversation_Integration(t *testing.T) {
	t.Parallel()

	_, repo, entClient := setupTestDB(t)
	ctx := context.Background()

	t.Run("CreateConversation", func(t *testing.T) {
		t.Parallel()

		// Setup dependencies
		akariUser, err := repo.CreateAkariUser(ctx)
		require.NoError(t, err)

		discordUser := RandomDiscordUser()
		createdDiscordUser, err := repo.CreateDiscordUser(ctx, discordUser)
		require.NoError(t, err)

		guild := RandomDiscordGuild()
		createdGuild, err := repo.CreateDiscordGuild(ctx, guild)
		require.NoError(t, err)

		channel := RandomDiscordChannel(createdGuild.ID)
		createdChannel, err := repo.CreateDiscordChannel(ctx, channel)
		require.NoError(t, err)

		message := RandomDiscordMessage(createdDiscordUser.ID, createdChannel.ID)
		createdMessage, err := repo.CreateDiscordMessage(ctx, message)
		require.NoError(t, err)

		gofakeit.Seed(time.Now().UnixNano())
		config, err := entClient.CharacterConfig.Create().
			SetDefaultSystemPrompt(gofakeit.Sentence(10)).
			Save(ctx)
		require.NoError(t, err)

		systemPrompt, err := entClient.SystemPrompt.Create().
			SetTitle(gofakeit.Word()).
			SetPurpose("text_chat").
			SetPrompt(gofakeit.Paragraph(3, 5, 10, "\n")).
			Save(ctx)
		require.NoError(t, err)

		character, err := entClient.Character.Create().
			SetName(gofakeit.Name()).
			SetConfig(config).
			AddSystemPrompts(systemPrompt).
			Save(ctx)
		require.NoError(t, err)

		conversationGroup, err := repo.CreateConversationGroup(ctx, character.ID)
		require.NoError(t, err)

		params := RandomConversation(akariUser.ID, createdMessage.ID, conversationGroup.ID)
		conversation, err := repo.CreateConversation(ctx, params)
		require.NoError(t, err)
		assert.Greater(t, conversation.ID, 0)
		assert.Equal(t, params.UserID, conversation.UserID)
		assert.Equal(t, params.DiscordMessageID, conversation.DiscordMessageID)
		assert.Equal(t, params.ConversationGroupID, conversation.ConversationGroupID)
	})

	t.Run("GetConversationByID", func(t *testing.T) {
		t.Parallel()

		// Setup dependencies
		akariUser, err := repo.CreateAkariUser(ctx)
		require.NoError(t, err)

		discordUser := RandomDiscordUser()
		createdDiscordUser, err := repo.CreateDiscordUser(ctx, discordUser)
		require.NoError(t, err)

		guild := RandomDiscordGuild()
		createdGuild, err := repo.CreateDiscordGuild(ctx, guild)
		require.NoError(t, err)

		channel := RandomDiscordChannel(createdGuild.ID)
		createdChannel, err := repo.CreateDiscordChannel(ctx, channel)
		require.NoError(t, err)

		message := RandomDiscordMessage(createdDiscordUser.ID, createdChannel.ID)
		createdMessage, err := repo.CreateDiscordMessage(ctx, message)
		require.NoError(t, err)

		gofakeit.Seed(time.Now().UnixNano())
		config, err := entClient.CharacterConfig.Create().
			SetDefaultSystemPrompt(gofakeit.Sentence(10)).
			Save(ctx)
		require.NoError(t, err)

		systemPrompt, err := entClient.SystemPrompt.Create().
			SetTitle(gofakeit.Word()).
			SetPurpose("text_chat").
			SetPrompt(gofakeit.Paragraph(3, 5, 10, "\n")).
			Save(ctx)
		require.NoError(t, err)

		character, err := entClient.Character.Create().
			SetName(gofakeit.Name()).
			SetConfig(config).
			AddSystemPrompts(systemPrompt).
			Save(ctx)
		require.NoError(t, err)

		conversationGroup, err := repo.CreateConversationGroup(ctx, character.ID)
		require.NoError(t, err)

		params := RandomConversation(akariUser.ID, createdMessage.ID, conversationGroup.ID)
		created, err := repo.CreateConversation(ctx, params)
		require.NoError(t, err)

		got, err := repo.GetConversationByID(ctx, created.ID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, got.ID)
		assert.Equal(t, created.UserID, got.UserID)
		assert.Equal(t, created.DiscordMessageID, got.DiscordMessageID)
		assert.Equal(t, created.ConversationGroupID, got.ConversationGroupID)
	})

	t.Run("GetConversationByID - not found", func(t *testing.T) {
		t.Parallel()

		_, err := repo.GetConversationByID(ctx, 99999)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get conversation by id")
	})

	t.Run("ListConversations", func(t *testing.T) {
		t.Parallel()

		// Setup dependencies
		akariUser, err := repo.CreateAkariUser(ctx)
		require.NoError(t, err)

		discordUser := RandomDiscordUser()
		createdDiscordUser, err := repo.CreateDiscordUser(ctx, discordUser)
		require.NoError(t, err)

		guild := RandomDiscordGuild()
		createdGuild, err := repo.CreateDiscordGuild(ctx, guild)
		require.NoError(t, err)

		channel := RandomDiscordChannel(createdGuild.ID)
		createdChannel, err := repo.CreateDiscordChannel(ctx, channel)
		require.NoError(t, err)

		gofakeit.Seed(time.Now().UnixNano())
		config, err := entClient.CharacterConfig.Create().
			SetDefaultSystemPrompt(gofakeit.Sentence(10)).
			Save(ctx)
		require.NoError(t, err)

		systemPrompt, err := entClient.SystemPrompt.Create().
			SetTitle(gofakeit.Word()).
			SetPurpose("text_chat").
			SetPrompt(gofakeit.Paragraph(3, 5, 10, "\n")).
			Save(ctx)
		require.NoError(t, err)

		character, err := entClient.Character.Create().
			SetName(gofakeit.Name()).
			SetConfig(config).
			AddSystemPrompts(systemPrompt).
			Save(ctx)
		require.NoError(t, err)

		conversationGroup, err := repo.CreateConversationGroup(ctx, character.ID)
		require.NoError(t, err)

		message1 := RandomDiscordMessage(createdDiscordUser.ID, createdChannel.ID)
		createdMessage1, err := repo.CreateDiscordMessage(ctx, message1)
		require.NoError(t, err)

		params1 := RandomConversation(akariUser.ID, createdMessage1.ID, conversationGroup.ID)
		conv1, err := repo.CreateConversation(ctx, params1)
		require.NoError(t, err)

		message2 := RandomDiscordMessage(createdDiscordUser.ID, createdChannel.ID)
		createdMessage2, err := repo.CreateDiscordMessage(ctx, message2)
		require.NoError(t, err)

		params2 := RandomConversation(akariUser.ID, createdMessage2.ID, conversationGroup.ID)
		conv2, err := repo.CreateConversation(ctx, params2)
		require.NoError(t, err)

		conversations, err := repo.ListConversations(ctx)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(conversations), 2)

		found1 := false
		found2 := false
		for _, c := range conversations {
			if c.ID == conv1.ID {
				found1 = true
			}
			if c.ID == conv2.ID {
				found2 = true
			}
		}
		assert.True(t, found1, "conv1 should be in the list")
		assert.True(t, found2, "conv2 should be in the list")
	})

	t.Run("DeleteConversation", func(t *testing.T) {
		t.Parallel()

		// Setup dependencies
		akariUser, err := repo.CreateAkariUser(ctx)
		require.NoError(t, err)

		discordUser := RandomDiscordUser()
		createdDiscordUser, err := repo.CreateDiscordUser(ctx, discordUser)
		require.NoError(t, err)

		guild := RandomDiscordGuild()
		createdGuild, err := repo.CreateDiscordGuild(ctx, guild)
		require.NoError(t, err)

		channel := RandomDiscordChannel(createdGuild.ID)
		createdChannel, err := repo.CreateDiscordChannel(ctx, channel)
		require.NoError(t, err)

		message := RandomDiscordMessage(createdDiscordUser.ID, createdChannel.ID)
		createdMessage, err := repo.CreateDiscordMessage(ctx, message)
		require.NoError(t, err)

		gofakeit.Seed(time.Now().UnixNano())
		config, err := entClient.CharacterConfig.Create().
			SetDefaultSystemPrompt(gofakeit.Sentence(10)).
			Save(ctx)
		require.NoError(t, err)

		systemPrompt, err := entClient.SystemPrompt.Create().
			SetTitle(gofakeit.Word()).
			SetPurpose("text_chat").
			SetPrompt(gofakeit.Paragraph(3, 5, 10, "\n")).
			Save(ctx)
		require.NoError(t, err)

		character, err := entClient.Character.Create().
			SetName(gofakeit.Name()).
			SetConfig(config).
			AddSystemPrompts(systemPrompt).
			Save(ctx)
		require.NoError(t, err)

		conversationGroup, err := repo.CreateConversationGroup(ctx, character.ID)
		require.NoError(t, err)

		params := RandomConversation(akariUser.ID, createdMessage.ID, conversationGroup.ID)
		created, err := repo.CreateConversation(ctx, params)
		require.NoError(t, err)

		err = repo.DeleteConversation(ctx, created.ID)
		require.NoError(t, err)

		_, err = repo.GetConversationByID(ctx, created.ID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get conversation by id")
	})
}

func TestRepository_SystemPrompt_Integration(t *testing.T) {
	t.Parallel()

	_, repo, entClient := setupTestDB(t)
	ctx := context.Background()

	t.Run("GetSystemPromptByID", func(t *testing.T) {
		t.Parallel()

		gofakeit.Seed(time.Now().UnixNano())

		systemPrompt, err := entClient.SystemPrompt.Create().
			SetTitle(gofakeit.Word()).
			SetPurpose("text_chat").
			SetPrompt(gofakeit.Paragraph(3, 5, 10, "\n")).
			Save(ctx)
		require.NoError(t, err)

		got, err := repo.GetSystemPromptByID(ctx, systemPrompt.ID)
		require.NoError(t, err)
		assert.Equal(t, systemPrompt.ID, got.ID)
		assert.Equal(t, systemPrompt.Title, got.Title)
		assert.Equal(t, string(systemPrompt.Purpose), got.Purpose)
		assert.Equal(t, systemPrompt.Prompt, got.Prompt)
	})

	t.Run("GetSystemPromptByID - not found", func(t *testing.T) {
		t.Parallel()

		_, err := repo.GetSystemPromptByID(ctx, 99999)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get system prompt by id")
	})
}

func TestRepository_Transactions_Integration(t *testing.T) {
	t.Parallel()

	_, repo, _ := setupTestDB(t)
	ctx := context.Background()

	t.Run("successful transaction commit", func(t *testing.T) {
		t.Parallel()

		err := repo.WithTransaction(ctx, func(ctx context.Context, tx *domain.Tx) error {
			// Create multiple entities in a transaction
			user1, err := repo.CreateAkariUser(ctx)
			if err != nil {
				return err
			}

			user2, err := repo.CreateAkariUser(ctx)
			if err != nil {
				return err
			}

			// Verify both users are created
			_, err = repo.GetAkariUserByID(ctx, user1.ID)
			if err != nil {
				return err
			}

			_, err = repo.GetAkariUserByID(ctx, user2.ID)
			if err != nil {
				return err
			}

			return nil
		})

		require.NoError(t, err)
	})

	t.Run("transaction rollback on error", func(t *testing.T) {
		t.Parallel()

		// Create a user before transaction
		userBefore, err := repo.CreateAkariUser(ctx)
		require.NoError(t, err)

		err = repo.WithTransaction(ctx, func(ctx context.Context, tx *domain.Tx) error {
			// Try to create another user
			_, createErr := repo.CreateAkariUser(ctx)
			if createErr != nil {
				return createErr
			}

			// Return an error to trigger rollback
			return errors.New("transaction error")
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "transaction error")

		// Verify user created before transaction still exists
		_, err = repo.GetAkariUserByID(ctx, userBefore.ID)
		require.NoError(t, err)
	})

	t.Run("multiple entities in transaction", func(t *testing.T) {
		t.Parallel()

		gofakeit.Seed(time.Now().UnixNano())

		err := repo.WithTransaction(ctx, func(ctx context.Context, tx *domain.Tx) error {
			// Create DiscordUser
			discordUser := RandomDiscordUser()
			createdUser, err := repo.CreateDiscordUser(ctx, discordUser)
			if err != nil {
				return err
			}

			// Create DiscordGuild
			guild := RandomDiscordGuild()
			createdGuild, err := repo.CreateDiscordGuild(ctx, guild)
			if err != nil {
				return err
			}

			// Create DiscordChannel
			channel := RandomDiscordChannel(createdGuild.ID)
			createdChannel, err := repo.CreateDiscordChannel(ctx, channel)
			if err != nil {
				return err
			}

			// Create DiscordMessage
			message := RandomDiscordMessage(createdUser.ID, createdChannel.ID)
			createdMessage, err := repo.CreateDiscordMessage(ctx, message)
			if err != nil {
				return err
			}

			// Verify all entities are created
			_, err = repo.GetDiscordUserByID(ctx, createdUser.ID)
			if err != nil {
				return err
			}

			_, err = repo.GetDiscordGuildByID(ctx, createdGuild.ID)
			if err != nil {
				return err
			}

			_, err = repo.GetDiscordChannelByID(ctx, createdChannel.ID)
			if err != nil {
				return err
			}

			_, err = repo.GetDiscordMessageByID(ctx, createdMessage.ID)
			if err != nil {
				return err
			}

			return nil
		})

		require.NoError(t, err)
	})

	t.Run("transaction rollback with multiple entities", func(t *testing.T) {
		t.Parallel()

		gofakeit.Seed(time.Now().UnixNano())

		// Create entities before transaction
		discordUserBefore := RandomDiscordUser()
		createdUserBefore, err := repo.CreateDiscordUser(ctx, discordUserBefore)
		require.NoError(t, err)

		err = repo.WithTransaction(ctx, func(ctx context.Context, tx *domain.Tx) error {
			// Create multiple entities in transaction
			guild := RandomDiscordGuild()
			_, createErr := repo.CreateDiscordGuild(ctx, guild)
			if createErr != nil {
				return createErr
			}

			channel := RandomDiscordChannel(guild.ID)
			_, createErr = repo.CreateDiscordChannel(ctx, channel)
			if createErr != nil {
				return createErr
			}

			// Return error to trigger rollback
			return errors.New("transaction rollback test")
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "transaction rollback test")

		// Verify entity created before transaction still exists
		_, err = repo.GetDiscordUserByID(ctx, createdUserBefore.ID)
		require.NoError(t, err)
	})
}
