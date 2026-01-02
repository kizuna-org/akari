package repository_test

import (
	"context"
	"testing"

	"github.com/kizuna-org/akari/pkg/database/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepository_CreateDiscordUser_Integration(t *testing.T) {
	t.Parallel()

	_, repo, _ := setupTestDB(t)
	ctx := context.Background()

	tests := []struct {
		name     string
		setup    func() domain.DiscordUser
		validate func(t *testing.T, got *domain.DiscordUser, expected domain.DiscordUser)
	}{
		{
			name: "success",
			setup: func() domain.DiscordUser {
				akariUser, err := repo.CreateAkariUser(ctx)
				require.NoError(t, err)

				discordUser := RandomDiscordUser()
				discordUser.AkariUserID = &akariUser.ID
				return discordUser
			},
			validate: func(t *testing.T, got *domain.DiscordUser, expected domain.DiscordUser) {
				assert.Equal(t, expected.ID, got.ID)
				assert.Equal(t, expected.Username, got.Username)
				assert.Equal(t, expected.Bot, got.Bot)
				assert.NotZero(t, got.CreatedAt)
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			params := testCase.setup()

			got, err := repo.CreateDiscordUser(ctx, params)
			require.NoError(t, err)

			if testCase.validate != nil {
				testCase.validate(t, got, params)
			}
		})
	}
}

func TestRepository_GetDiscordUserByID_Integration(t *testing.T) {
	t.Parallel()

	_, repo, _ := setupTestDB(t)
	ctx := context.Background()

	tests := []struct {
		name     string
		setup    func() string
		wantErr  bool
		errMsg   string
		validate func(t *testing.T, got *domain.DiscordUser, expectedID string)
	}{
		{
			name: "not found",
			setup: func() string {
				return RandomDiscordID()
			},
			wantErr: true,
			errMsg:  "failed to get discord user by id",
		},
		{
			name: "success",
			setup: func() string {
				akariUser, err := repo.CreateAkariUser(ctx)
				require.NoError(t, err)

				params := RandomDiscordUser()
				params.AkariUserID = &akariUser.ID
				created, err := repo.CreateDiscordUser(ctx, params)
				require.NoError(t, err)
				return created.ID
			},
			wantErr: false,
			validate: func(t *testing.T, got *domain.DiscordUser, expectedID string) {
				assert.Equal(t, expectedID, got.ID)
				assert.NotEmpty(t, got.Username)
				assert.NotZero(t, got.CreatedAt)
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			userID := testCase.setup()

			got, err := repo.GetDiscordUserByID(ctx, userID)

			if testCase.wantErr {
				require.Error(t, err)
				if testCase.errMsg != "" {
					assert.Contains(t, err.Error(), testCase.errMsg)
				}
			} else {
				require.NoError(t, err)
				if testCase.validate != nil {
					testCase.validate(t, got, userID)
				}
			}
		})
	}
}

func TestRepository_ListDiscordUsers_Integration(t *testing.T) {
	t.Parallel()

	_, repo, _ := setupTestDB(t)
	ctx := context.Background()

	tests := []struct {
		name     string
		setup    func() []string
		validate func(t *testing.T, got []*domain.DiscordUser, expectedIDs []string)
	}{
		{
			name: "with multiple users",
			setup: func() []string {
				akariUser1, err := repo.CreateAkariUser(ctx)
				require.NoError(t, err)

				akariUser2, err := repo.CreateAkariUser(ctx)
				require.NoError(t, err)

				user1 := RandomDiscordUser()
				user1.AkariUserID = &akariUser1.ID
				created1, err := repo.CreateDiscordUser(ctx, user1)
				require.NoError(t, err)

				user2 := RandomDiscordUser()
				user2.AkariUserID = &akariUser2.ID
				created2, err := repo.CreateDiscordUser(ctx, user2)
				require.NoError(t, err)

				return []string{created1.ID, created2.ID}
			},
			validate: func(t *testing.T, got []*domain.DiscordUser, expectedIDs []string) {
				assert.GreaterOrEqual(t, len(got), len(expectedIDs))

				found := make(map[string]bool)
				for _, id := range expectedIDs {
					found[id] = false
				}

				for _, u := range got {
					if _, exists := found[u.ID]; exists {
						found[u.ID] = true
					}
				}

				for id, wasFound := range found {
					assert.True(t, wasFound, "user %s should be in the list", id)
				}
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			expectedIDs := testCase.setup()

			got, err := repo.ListDiscordUsers(ctx)
			require.NoError(t, err)

			if testCase.validate != nil {
				testCase.validate(t, got, expectedIDs)
			}
		})
	}
}

func TestRepository_DeleteDiscordUser_Integration(t *testing.T) {
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
				akariUser, err := repo.CreateAkariUser(ctx)
				require.NoError(t, err)

				params := RandomDiscordUser()
				params.AkariUserID = &akariUser.ID
				created, err := repo.CreateDiscordUser(ctx, params)
				require.NoError(t, err)
				return created.ID
			},
			wantErr: false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			userID := testCase.setup()

			err := repo.DeleteDiscordUser(ctx, userID)

			if testCase.wantErr {
				require.Error(t, err)
				if testCase.errMsg != "" {
					assert.Contains(t, err.Error(), testCase.errMsg)
				}
			} else {
				require.NoError(t, err)

				_, err = repo.GetDiscordUserByID(ctx, userID)
				require.Error(t, err)
				assert.Contains(t, err.Error(), "failed to get discord user by id")
			}
		})
	}
}
