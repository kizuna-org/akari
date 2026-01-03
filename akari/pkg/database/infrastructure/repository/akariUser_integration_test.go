package repository_test

import (
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/kizuna-org/akari/pkg/database/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepository_CreateAkariUser_Integration(t *testing.T) {
	t.Parallel()

	repo, _ := setupTestDB(t)
	ctx := t.Context()

	tests := []struct {
		name     string
		validate func(t *testing.T, got *domain.AkariUser)
	}{
		{
			name: "success",
			validate: func(t *testing.T, got *domain.AkariUser) {
				t.Helper()
				assert.Positive(t, got.ID)
				assert.NotZero(t, got.CreatedAt)
				assert.NotZero(t, got.UpdatedAt)
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, err := repo.CreateAkariUser(ctx)
			require.NoError(t, err)

			if testCase.validate != nil {
				testCase.validate(t, got)
			}
		})
	}
}

func TestRepository_GetAkariUserByID_Integration(t *testing.T) {
	t.Parallel()

	repo, _ := setupTestDB(t)
	ctx := t.Context()

	tests := []struct {
		name     string
		setup    func() int
		wantErr  bool
		errMsg   string
		validate func(t *testing.T, got *domain.AkariUser, expectedID int)
	}{
		{
			name: "not found",
			setup: func() int {
				return 99999
			},
			wantErr: true,
			errMsg:  "failed to get akari user",
		},
		{
			name: "success",
			setup: func() int {
				created, err := repo.CreateAkariUser(ctx)
				require.NoError(t, err)

				return created.ID
			},
			wantErr: false,
			validate: func(t *testing.T, got *domain.AkariUser, expectedID int) {
				t.Helper()
				assert.Equal(t, expectedID, got.ID)
				assert.NotZero(t, got.CreatedAt)
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			userID := testCase.setup()

			got, err := repo.GetAkariUserByID(ctx, userID)

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

func TestRepository_GetAkariUserByDiscordUserID_Integration(t *testing.T) {
	t.Parallel()

	repo, entClient := setupTestDB(t)
	ctx := t.Context()

	tests := []struct {
		name     string
		setup    func() string
		wantErr  bool
		errMsg   string
		validate func(t *testing.T, got *domain.AkariUser, expectedDiscordID string)
	}{
		{
			name:    "not found",
			setup:   RandomDiscordID,
			wantErr: true,
			errMsg:  "failed to get akari user by discord id",
		},
		{
			name: "success",
			setup: func() string {
				_ = gofakeit.Seed(time.Now().UnixNano())

				akariUser, err := entClient.AkariUser.Create().
					Save(ctx)
				require.NoError(t, err)

				discordUser, err := entClient.DiscordUser.Create().
					SetID(RandomDiscordID()).
					SetUsername(RandomDiscordUsername()).
					SetBot(gofakeit.Bool()).
					SetAkariUser(akariUser).
					Save(ctx)
				require.NoError(t, err)

				return discordUser.ID
			},
			wantErr: false,
			validate: func(t *testing.T, got *domain.AkariUser, expectedDiscordID string) {
				t.Helper()
				assert.Positive(t, got.ID)
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			discordID := testCase.setup()

			got, err := repo.GetAkariUserByDiscordUserID(ctx, discordID)

			if testCase.wantErr {
				require.Error(t, err)

				if testCase.errMsg != "" {
					assert.Contains(t, err.Error(), testCase.errMsg)
				}
			} else {
				require.NoError(t, err)

				if testCase.validate != nil {
					testCase.validate(t, got, discordID)
				}
			}
		})
	}
}

func TestRepository_ListAkariUsers_Integration(t *testing.T) {
	t.Parallel()

	repo, _ := setupTestDB(t)
	ctx := t.Context()

	tests := []struct {
		name     string
		setup    func() []int
		validate func(t *testing.T, got []*domain.AkariUser, expectedIDs []int)
	}{
		{
			name: "with multiple users",
			setup: func() []int {
				user1, err := repo.CreateAkariUser(ctx)
				require.NoError(t, err)

				user2, err := repo.CreateAkariUser(ctx)
				require.NoError(t, err)

				return []int{user1.ID, user2.ID}
			},
			validate: func(t *testing.T, got []*domain.AkariUser, expectedIDs []int) {
				t.Helper()
				found := make(map[int]bool)
				for _, id := range expectedIDs {
					found[id] = false
				}

				for _, u := range got {
					if _, exists := found[u.ID]; exists {
						found[u.ID] = true
					}
				}

				for id, wasFound := range found {
					assert.True(t, wasFound, "user %d should be in the list", id)
				}
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			expectedIDs := testCase.setup()

			got, err := repo.ListAkariUsers(ctx)
			require.NoError(t, err)

			if testCase.validate != nil {
				testCase.validate(t, got, expectedIDs)
			}
		})
	}
}

func TestRepository_DeleteAkariUser_Integration(t *testing.T) {
	t.Parallel()

	repo, _ := setupTestDB(t)
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
				user, err := repo.CreateAkariUser(ctx)
				require.NoError(t, err)

				return user.ID
			},
			wantErr: false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			userID := testCase.setup()

			err := repo.DeleteAkariUser(ctx, userID)

			if testCase.wantErr {
				require.Error(t, err)

				if testCase.errMsg != "" {
					assert.Contains(t, err.Error(), testCase.errMsg)
				}
			} else {
				require.NoError(t, err)

				_, err = repo.GetAkariUserByID(ctx, userID)
				require.Error(t, err)
				assert.Contains(t, err.Error(), "failed to get akari user")
			}
		})
	}
}
