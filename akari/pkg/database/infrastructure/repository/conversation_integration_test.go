package repository_test

import (
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/kizuna-org/akari/pkg/database/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepository_CreateConversation_Integration(t *testing.T) {
	t.Parallel()

	repo, entClient := setupTestDB(t)
	ctx := t.Context()

	tests := []struct {
		name     string
		setup    func() domain.Conversation
		validate func(t *testing.T, got *domain.Conversation, expected domain.Conversation)
	}{
		{
			name: "success",
			setup: func() domain.Conversation {
				akariUser, err := repo.CreateAkariUser(ctx)
				require.NoError(t, err)

				discordUser := RandomDiscordUser()
				discordUser.AkariUserID = &akariUser.ID
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

				_ = gofakeit.Seed(time.Now().UnixNano())
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

				return RandomConversation(akariUser.ID, createdMessage.ID, conversationGroup.ID)
			},
			validate: func(t *testing.T, got *domain.Conversation, expected domain.Conversation) {
				t.Helper()
				assert.Positive(t, got.ID)
				assert.Equal(t, expected.UserID, got.UserID)
				assert.Equal(t, expected.DiscordMessageID, got.DiscordMessageID)
				assert.Equal(t, expected.ConversationGroupID, got.ConversationGroupID)
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			params := testCase.setup()

			got, err := repo.CreateConversation(ctx, params)
			require.NoError(t, err)

			if testCase.validate != nil {
				testCase.validate(t, got, params)
			}
		})
	}
}

func TestRepository_GetConversationByID_Integration(t *testing.T) {
	t.Parallel()

	repo, entClient := setupTestDB(t)
	ctx := t.Context()

	tests := []struct {
		name     string
		setup    func() int
		wantErr  bool
		errMsg   string
		validate func(t *testing.T, got *domain.Conversation, expectedID int)
	}{
		{
			name: "not found",
			setup: func() int {
				return 99999
			},
			wantErr: true,
			errMsg:  "failed to get conversation by id",
		},
		{
			name: "success",
			setup: func() int {
				akariUser, err := repo.CreateAkariUser(ctx)
				require.NoError(t, err)

				discordUser := RandomDiscordUser()
				discordUser.AkariUserID = &akariUser.ID
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

				_ = gofakeit.Seed(time.Now().UnixNano())
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

				return created.ID
			},
			wantErr: false,
			validate: func(t *testing.T, got *domain.Conversation, expectedID int) {
				t.Helper()
				assert.Equal(t, expectedID, got.ID)
				assert.Positive(t, got.UserID)
				assert.NotEmpty(t, got.DiscordMessageID)
				assert.Positive(t, got.ConversationGroupID)
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			conversationID := testCase.setup()

			got, err := repo.GetConversationByID(ctx, conversationID)

			if testCase.wantErr {
				require.Error(t, err)

				if testCase.errMsg != "" {
					assert.Contains(t, err.Error(), testCase.errMsg)
				}
			} else {
				require.NoError(t, err)

				if testCase.validate != nil {
					testCase.validate(t, got, conversationID)
				}
			}
		})
	}
}

func TestRepository_ListConversations_Integration(t *testing.T) {
	t.Parallel()

	repo, entClient := setupTestDB(t)
	ctx := t.Context()

	tests := []struct {
		name     string
		setup    func() []int
		validate func(t *testing.T, got []*domain.Conversation, expectedIDs []int)
	}{
		{
			name: "with multiple conversations",
			setup: func() []int {
				akariUser, err := repo.CreateAkariUser(ctx)
				require.NoError(t, err)

				discordUser := RandomDiscordUser()
				discordUser.AkariUserID = &akariUser.ID
				createdDiscordUser, err := repo.CreateDiscordUser(ctx, discordUser)
				require.NoError(t, err)

				guild := RandomDiscordGuild()
				createdGuild, err := repo.CreateDiscordGuild(ctx, guild)
				require.NoError(t, err)

				channel := RandomDiscordChannel(createdGuild.ID)
				createdChannel, err := repo.CreateDiscordChannel(ctx, channel)
				require.NoError(t, err)

				_ = gofakeit.Seed(time.Now().UnixNano())
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

				var conversationIDs []int

				for range 2 {
					message := RandomDiscordMessage(createdDiscordUser.ID, createdChannel.ID)
					createdMessage, err := repo.CreateDiscordMessage(ctx, message)
					require.NoError(t, err)

					params := RandomConversation(akariUser.ID, createdMessage.ID, conversationGroup.ID)
					conv, err := repo.CreateConversation(ctx, params)
					require.NoError(t, err)

					conversationIDs = append(conversationIDs, conv.ID)
				}

				return conversationIDs
			},
			validate: func(t *testing.T, got []*domain.Conversation, expectedIDs []int) {
				t.Helper()
				found := make(map[int]bool)
				for _, id := range expectedIDs {
					found[id] = false
				}

				for _, c := range got {
					if _, exists := found[c.ID]; exists {
						found[c.ID] = true
					}
				}

				for id, wasFound := range found {
					assert.True(t, wasFound, "conversation %d should be in the list", id)
				}
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			expectedIDs := testCase.setup()

			got, err := repo.ListConversations(ctx)
			require.NoError(t, err)

			if testCase.validate != nil {
				testCase.validate(t, got, expectedIDs)
			}
		})
	}
}

func TestRepository_DeleteConversation_Integration(t *testing.T) {
	t.Parallel()

	repo, entClient := setupTestDB(t)
	ctx := t.Context()

	tests := []struct {
		name    string
		setup   func() int
		wantErr bool
		errMsg  string
	}{
		{
			name: "success",
			setup: func() int {
				akariUser, err := repo.CreateAkariUser(ctx)
				require.NoError(t, err)

				discordUser := RandomDiscordUser()
				discordUser.AkariUserID = &akariUser.ID
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

				_ = gofakeit.Seed(time.Now().UnixNano())
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

				return created.ID
			},
			wantErr: false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			conversationID := testCase.setup()

			err := repo.DeleteConversation(ctx, conversationID)

			if testCase.wantErr {
				require.Error(t, err)

				if testCase.errMsg != "" {
					assert.Contains(t, err.Error(), testCase.errMsg)
				}
			} else {
				require.NoError(t, err)

				_, err = repo.GetConversationByID(ctx, conversationID)
				require.Error(t, err)
				assert.Contains(t, err.Error(), "failed to get conversation by id")
			}
		})
	}
}
