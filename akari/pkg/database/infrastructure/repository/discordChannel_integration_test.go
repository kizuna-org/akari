package repository_test

import (
	"testing"

	"github.com/kizuna-org/akari/pkg/database/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepository_CreateDiscordChannel_Integration(t *testing.T) {
	t.Parallel()

	repo, _ := setupTestDB(t)
	ctx := t.Context()

	tests := []struct {
		name     string
		setup    func() (domain.DiscordChannel, string)
		validate func(t *testing.T, got *domain.DiscordChannel, expected domain.DiscordChannel)
	}{
		{
			name: "success",
			setup: func() (domain.DiscordChannel, string) {
				guild := RandomDiscordGuild()
				createdGuild, err := repo.CreateDiscordGuild(ctx, guild)
				require.NoError(t, err)

				params := RandomDiscordChannel(createdGuild.ID)

				return params, createdGuild.ID
			},
			validate: func(t *testing.T, got *domain.DiscordChannel, expected domain.DiscordChannel) {
				t.Helper()
				assert.Equal(t, expected.ID, got.ID)
				assert.Equal(t, expected.Type, got.Type)
				assert.Equal(t, expected.Name, got.Name)
				assert.Equal(t, expected.GuildID, got.GuildID)
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			params, _ := testCase.setup()

			got, err := repo.CreateDiscordChannel(ctx, params)
			require.NoError(t, err)

			if testCase.validate != nil {
				testCase.validate(t, got, params)
			}
		})
	}
}

func TestRepository_GetDiscordChannelByID_Integration(t *testing.T) {
	t.Parallel()

	repo, _ := setupTestDB(t)
	ctx := t.Context()

	tests := []struct {
		name     string
		setup    func() string
		wantErr  bool
		errMsg   string
		validate func(t *testing.T, got *domain.DiscordChannel, expectedID string)
	}{
		{
			name:    "not found",
			setup:   RandomDiscordID,
			wantErr: true,
			errMsg:  "failed to get discord channel by id",
		},
		{
			name: "success",
			setup: func() string {
				guild := RandomDiscordGuild()
				createdGuild, err := repo.CreateDiscordGuild(ctx, guild)
				require.NoError(t, err)

				params := RandomDiscordChannel(createdGuild.ID)
				created, err := repo.CreateDiscordChannel(ctx, params)
				require.NoError(t, err)

				return created.ID
			},
			wantErr: false,
			validate: func(t *testing.T, got *domain.DiscordChannel, expectedID string) {
				t.Helper()
				assert.Equal(t, expectedID, got.ID)
				assert.NotEmpty(t, got.Name)
				assert.NotEmpty(t, got.GuildID)
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			channelID := testCase.setup()

			got, err := repo.GetDiscordChannelByID(ctx, channelID)

			if testCase.wantErr {
				require.Error(t, err)

				if testCase.errMsg != "" {
					assert.Contains(t, err.Error(), testCase.errMsg)
				}
			} else {
				require.NoError(t, err)

				if testCase.validate != nil {
					testCase.validate(t, got, channelID)
				}
			}
		})
	}
}

func TestRepository_GetDiscordChannelsByGuildID_Integration(t *testing.T) {
	t.Parallel()

	repo, _ := setupTestDB(t)
	ctx := t.Context()

	tests := []struct {
		name     string
		setup    func() (string, []string)
		validate func(t *testing.T, got []*domain.DiscordChannel, guildID string, expectedIDs []string)
	}{
		{
			name: "with multiple channels",
			setup: func() (string, []string) {
				guild := RandomDiscordGuild()
				createdGuild, err := repo.CreateDiscordGuild(ctx, guild)
				require.NoError(t, err)

				channel1 := RandomDiscordChannel(createdGuild.ID)
				created1, err := repo.CreateDiscordChannel(ctx, channel1)
				require.NoError(t, err)

				channel2 := RandomDiscordChannel(createdGuild.ID)
				created2, err := repo.CreateDiscordChannel(ctx, channel2)
				require.NoError(t, err)

				return createdGuild.ID, []string{created1.ID, created2.ID}
			},
			validate: func(t *testing.T, got []*domain.DiscordChannel, guildID string, expectedIDs []string) {
				t.Helper()
				assert.GreaterOrEqual(t, len(got), len(expectedIDs))

				found := make(map[string]bool)
				for _, id := range expectedIDs {
					found[id] = false
				}

				for _, c := range got {
					assert.Equal(t, guildID, c.GuildID)
					if _, exists := found[c.ID]; exists {
						found[c.ID] = true
					}
				}

				for id, wasFound := range found {
					assert.True(t, wasFound, "channel %s should be in the list", id)
				}
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			guildID, expectedIDs := testCase.setup()

			got, err := repo.GetDiscordChannelsByGuildID(ctx, guildID)
			require.NoError(t, err)

			if testCase.validate != nil {
				testCase.validate(t, got, guildID, expectedIDs)
			}
		})
	}
}

func TestRepository_DeleteDiscordChannel_Integration(t *testing.T) {
	t.Parallel()

	repo, _ := setupTestDB(t)
	ctx := t.Context()

	tests := []struct {
		name    string
		setup   func() string
		wantErr bool
		errMsg  string
	}{
		{
			name: "success",
			setup: func() string {
				guild := RandomDiscordGuild()
				createdGuild, err := repo.CreateDiscordGuild(ctx, guild)
				require.NoError(t, err)

				params := RandomDiscordChannel(createdGuild.ID)
				created, err := repo.CreateDiscordChannel(ctx, params)
				require.NoError(t, err)

				return created.ID
			},
			wantErr: false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			channelID := testCase.setup()

			err := repo.DeleteDiscordChannel(ctx, channelID)

			if testCase.wantErr {
				require.Error(t, err)

				if testCase.errMsg != "" {
					assert.Contains(t, err.Error(), testCase.errMsg)
				}
			} else {
				require.NoError(t, err)

				_, err = repo.GetDiscordChannelByID(ctx, channelID)
				require.Error(t, err)
				assert.Contains(t, err.Error(), "failed to get discord channel by id")
			}
		})
	}
}
