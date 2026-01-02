package repository_test

import (
	"context"
	"testing"

	"github.com/kizuna-org/akari/pkg/database/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepository_CreateDiscordMessage_Integration(t *testing.T) {
	t.Parallel()

	_, repo, _ := setupTestDB(t)
	ctx := context.Background()

	tests := []struct {
		name     string
		setup    func() domain.DiscordMessage
		validate func(t *testing.T, got *domain.DiscordMessage, expected domain.DiscordMessage)
	}{
		{
			name: "success",
			setup: func() domain.DiscordMessage {
				discordUser := RandomDiscordUser()
				createdUser, err := repo.CreateDiscordUser(ctx, discordUser)
				require.NoError(t, err)

				guild := RandomDiscordGuild()
				createdGuild, err := repo.CreateDiscordGuild(ctx, guild)
				require.NoError(t, err)

				channel := RandomDiscordChannel(createdGuild.ID)
				createdChannel, err := repo.CreateDiscordChannel(ctx, channel)
				require.NoError(t, err)

				return RandomDiscordMessage(createdUser.ID, createdChannel.ID)
			},
			validate: func(t *testing.T, got *domain.DiscordMessage, expected domain.DiscordMessage) {
				assert.Equal(t, expected.ID, got.ID)
				assert.Equal(t, expected.AuthorID, got.AuthorID)
				assert.Equal(t, expected.ChannelID, got.ChannelID)
				assert.Equal(t, expected.Content, got.Content)
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			params := testCase.setup()

			got, err := repo.CreateDiscordMessage(ctx, params)
			require.NoError(t, err)

			if testCase.validate != nil {
				testCase.validate(t, got, params)
			}
		})
	}
}

func TestRepository_GetDiscordMessageByID_Integration(t *testing.T) {
	t.Parallel()

	_, repo, _ := setupTestDB(t)
	ctx := context.Background()

	tests := []struct {
		name     string
		setup    func() string
		wantErr  bool
		errMsg   string
		validate func(t *testing.T, got *domain.DiscordMessage, expectedID string)
	}{
		{
			name: "not found",
			setup: func() string {
				return RandomDiscordID()
			},
			wantErr: true,
			errMsg:  "failed to get discord message by id",
		},
		{
			name: "success",
			setup: func() string {
				discordUser := RandomDiscordUser()
				createdUser, err := repo.CreateDiscordUser(ctx, discordUser)
				require.NoError(t, err)

				guild := RandomDiscordGuild()
				createdGuild, err := repo.CreateDiscordGuild(ctx, guild)
				require.NoError(t, err)

				channel := RandomDiscordChannel(createdGuild.ID)
				createdChannel, err := repo.CreateDiscordChannel(ctx, channel)
				require.NoError(t, err)

				message := RandomDiscordMessage(createdUser.ID, createdChannel.ID)
				created, err := repo.CreateDiscordMessage(ctx, message)
				require.NoError(t, err)
				return created.ID
			},
			wantErr: false,
			validate: func(t *testing.T, got *domain.DiscordMessage, expectedID string) {
				assert.Equal(t, expectedID, got.ID)
				assert.NotEmpty(t, got.AuthorID)
				assert.NotEmpty(t, got.ChannelID)
				assert.NotEmpty(t, got.Content)
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			messageID := testCase.setup()

			got, err := repo.GetDiscordMessageByID(ctx, messageID)

			if testCase.wantErr {
				require.Error(t, err)
				if testCase.errMsg != "" {
					assert.Contains(t, err.Error(), testCase.errMsg)
				}
			} else {
				require.NoError(t, err)
				if testCase.validate != nil {
					testCase.validate(t, got, messageID)
				}
			}
		})
	}
}

func TestRepository_DeleteDiscordMessage_Integration(t *testing.T) {
	t.Parallel()

	_, repo, _ := setupTestDB(t)
	ctx := context.Background()

	tests := []struct {
		name    string
		setup   func() string
		wantErr bool
		errMsg  string
	}{
		{
			name: "success",
			setup: func() string {
				discordUser := RandomDiscordUser()
				createdUser, err := repo.CreateDiscordUser(ctx, discordUser)
				require.NoError(t, err)

				guild := RandomDiscordGuild()
				createdGuild, err := repo.CreateDiscordGuild(ctx, guild)
				require.NoError(t, err)

				channel := RandomDiscordChannel(createdGuild.ID)
				createdChannel, err := repo.CreateDiscordChannel(ctx, channel)
				require.NoError(t, err)

				message := RandomDiscordMessage(createdUser.ID, createdChannel.ID)
				created, err := repo.CreateDiscordMessage(ctx, message)
				require.NoError(t, err)
				return created.ID
			},
			wantErr: false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			messageID := testCase.setup()

			err := repo.DeleteDiscordMessage(ctx, messageID)

			if testCase.wantErr {
				require.Error(t, err)
				if testCase.errMsg != "" {
					assert.Contains(t, err.Error(), testCase.errMsg)
				}
			} else {
				require.NoError(t, err)

				_, err = repo.GetDiscordMessageByID(ctx, messageID)
				require.Error(t, err)
				assert.Contains(t, err.Error(), "failed to get discord message by id")
			}
		})
	}
}
