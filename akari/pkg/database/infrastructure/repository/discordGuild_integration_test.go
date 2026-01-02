//go:build integration

package repository_test

import (
	"context"
	"testing"

	"github.com/kizuna-org/akari/pkg/database/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepository_CreateDiscordGuild_Integration(t *testing.T) {
	t.Parallel()

	_, repo, _ := setupTestDB(t)
	ctx := context.Background()

	tests := []struct {
		name     string
		setup    func() domain.DiscordGuild
		validate func(t *testing.T, got *domain.DiscordGuild, expected domain.DiscordGuild)
	}{
		{
			name: "success",
			setup: func() domain.DiscordGuild {
				return RandomDiscordGuild()
			},
			validate: func(t *testing.T, got *domain.DiscordGuild, expected domain.DiscordGuild) {
				assert.Equal(t, expected.ID, got.ID)
				assert.Equal(t, expected.Name, got.Name)
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			params := testCase.setup()

			got, err := repo.CreateDiscordGuild(ctx, params)
			require.NoError(t, err)

			if testCase.validate != nil {
				testCase.validate(t, got, params)
			}
		})
	}
}

func TestRepository_GetDiscordGuildByID_Integration(t *testing.T) {
	t.Parallel()

	_, repo, _ := setupTestDB(t)
	ctx := context.Background()

	tests := []struct {
		name     string
		setup    func() string
		wantErr  bool
		errMsg   string
		validate func(t *testing.T, got *domain.DiscordGuild, expectedID string)
	}{
		{
			name: "not found",
			setup: func() string {
				return RandomDiscordID()
			},
			wantErr: true,
			errMsg:  "failed to get discord guild by id",
		},
		{
			name: "success",
			setup: func() string {
				params := RandomDiscordGuild()
				created, err := repo.CreateDiscordGuild(ctx, params)
				require.NoError(t, err)
				return created.ID
			},
			wantErr: false,
			validate: func(t *testing.T, got *domain.DiscordGuild, expectedID string) {
				assert.Equal(t, expectedID, got.ID)
				assert.NotEmpty(t, got.Name)
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			guildID := testCase.setup()

			got, err := repo.GetDiscordGuildByID(ctx, guildID)

			if testCase.wantErr {
				require.Error(t, err)
				if testCase.errMsg != "" {
					assert.Contains(t, err.Error(), testCase.errMsg)
				}
			} else {
				require.NoError(t, err)
				if testCase.validate != nil {
					testCase.validate(t, got, guildID)
				}
			}
		})
	}
}

func TestRepository_ListDiscordGuilds_Integration(t *testing.T) {
	t.Parallel()

	_, repo, _ := setupTestDB(t)
	ctx := context.Background()

	tests := []struct {
		name     string
		setup    func() []string
		validate func(t *testing.T, got []*domain.DiscordGuild, expectedIDs []string)
	}{
		{
			name: "with multiple guilds",
			setup: func() []string {
				guild1 := RandomDiscordGuild()
				created1, err := repo.CreateDiscordGuild(ctx, guild1)
				require.NoError(t, err)

				guild2 := RandomDiscordGuild()
				created2, err := repo.CreateDiscordGuild(ctx, guild2)
				require.NoError(t, err)

				return []string{created1.ID, created2.ID}
			},
			validate: func(t *testing.T, got []*domain.DiscordGuild, expectedIDs []string) {
				assert.GreaterOrEqual(t, len(got), len(expectedIDs))

				found := make(map[string]bool)
				for _, id := range expectedIDs {
					found[id] = false
				}

				for _, g := range got {
					if _, exists := found[g.ID]; exists {
						found[g.ID] = true
					}
				}

				for id, wasFound := range found {
					assert.True(t, wasFound, "guild %s should be in the list", id)
				}
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			expectedIDs := testCase.setup()

			got, err := repo.ListDiscordGuilds(ctx)
			require.NoError(t, err)

			if testCase.validate != nil {
				testCase.validate(t, got, expectedIDs)
			}
		})
	}
}

func TestRepository_DeleteDiscordGuild_Integration(t *testing.T) {
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
				params := RandomDiscordGuild()
				created, err := repo.CreateDiscordGuild(ctx, params)
				require.NoError(t, err)
				return created.ID
			},
			wantErr: false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			guildID := testCase.setup()

			err := repo.DeleteDiscordGuild(ctx, guildID)

			if testCase.wantErr {
				require.Error(t, err)
				if testCase.errMsg != "" {
					assert.Contains(t, err.Error(), testCase.errMsg)
				}
			} else {
				require.NoError(t, err)

				_, err = repo.GetDiscordGuildByID(ctx, guildID)
				require.Error(t, err)
				assert.Contains(t, err.Error(), "failed to get discord guild by id")
			}
		})
	}
}
