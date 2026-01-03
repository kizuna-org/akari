package repository_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/kizuna-org/akari/pkg/database/domain"
	"github.com/kizuna-org/akari/pkg/database/infrastructure/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepository_Transactions_Integration(t *testing.T) {
	t.Parallel()

	repo, _ := setupTestDB(t)
	ctx := t.Context()

	tests := getTransactionTests(t, repo, ctx)

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			runTransactionTest(t, repo, ctx, testCase)
		})
	}
}

func getTransactionTests(
	t *testing.T,
	repo repository.Repository,
	ctx context.Context,
) []struct {
	name     string
	setup    func() interface{}
	fn       func(ctx context.Context, tx *domain.Tx, setup interface{}) error
	wantErr  bool
	errMsg   string
	validate func(t *testing.T, repo repository.Repository, ctx context.Context, setup interface{})
} {
	t.Helper()

	return []struct {
		name     string
		setup    func() interface{}
		fn       func(ctx context.Context, tx *domain.Tx, setup interface{}) error
		wantErr  bool
		errMsg   string
		validate func(t *testing.T, repo repository.Repository, ctx context.Context, setup interface{})
	}{
		successfulTransactionCommitTest(t, repo),
		transactionRollbackOnErrorTest(t, repo, ctx),
		multipleEntitiesInTransactionTest(t, repo),
		transactionRollbackWithMultipleEntitiesTest(t, repo, ctx),
	}
}

func successfulTransactionCommitTest(
	t *testing.T,
	repo repository.Repository,
) struct {
	name     string
	setup    func() interface{}
	fn       func(ctx context.Context, tx *domain.Tx, setup interface{}) error
	wantErr  bool
	errMsg   string
	validate func(t *testing.T, repo repository.Repository, ctx context.Context, setup interface{})
} {
	t.Helper()

	return struct {
		name     string
		setup    func() interface{}
		fn       func(ctx context.Context, tx *domain.Tx, setup interface{}) error
		wantErr  bool
		errMsg   string
		validate func(t *testing.T, repo repository.Repository, ctx context.Context, setup interface{})
	}{
		name: "successful transaction commit",
		setup: func() interface{} {
			return nil
		},
		fn: func(ctx context.Context, tx *domain.Tx, setup interface{}) error {
			user1, err := repo.CreateAkariUser(ctx)
			if err != nil {
				return err
			}

			user2, err := repo.CreateAkariUser(ctx)
			if err != nil {
				return err
			}

			_, err = repo.GetAkariUserByID(ctx, user1.ID)
			if err != nil {
				return err
			}

			_, err = repo.GetAkariUserByID(ctx, user2.ID)
			if err != nil {
				return err
			}

			return nil
		},
		wantErr: false,
		validate: func(t *testing.T, repo repository.Repository, ctx context.Context, setup interface{}) {
			t.Helper()
			// Users should be created after transaction
		},
	}
}

func transactionRollbackOnErrorTest(
	t *testing.T,
	repo repository.Repository,
	ctx context.Context,
) struct {
	name     string
	setup    func() interface{}
	fn       func(ctx context.Context, tx *domain.Tx, setup interface{}) error
	wantErr  bool
	errMsg   string
	validate func(t *testing.T, repo repository.Repository, ctx context.Context, setup interface{})
} {
	t.Helper()

	return struct {
		name     string
		setup    func() interface{}
		fn       func(ctx context.Context, tx *domain.Tx, setup interface{}) error
		wantErr  bool
		errMsg   string
		validate func(t *testing.T, repo repository.Repository, ctx context.Context, setup interface{})
	}{
		name: "transaction rollback on error",
		setup: func() interface{} {
			userBefore, err := repo.CreateAkariUser(ctx)
			require.NoError(t, err)

			return userBefore.ID
		},
		fn: func(ctx context.Context, tx *domain.Tx, setup interface{}) error {
			_, err := repo.CreateAkariUser(ctx)
			if err != nil {
				return err
			}

			return errors.New("transaction error")
		},
		wantErr: true,
		errMsg:  "transaction error",
		validate: func(t *testing.T, repo repository.Repository, ctx context.Context, setup interface{}) {
			t.Helper()
			userBeforeID, ok := setup.(int)
			require.True(t, ok, "setup should be int")
			_, err := repo.GetAkariUserByID(ctx, userBeforeID)
			require.NoError(t, err, "user created before transaction should still exist")
		},
	}
}

func multipleEntitiesInTransactionTest(
	t *testing.T,
	repo repository.Repository,
) struct {
	name     string
	setup    func() interface{}
	fn       func(ctx context.Context, tx *domain.Tx, setup interface{}) error
	wantErr  bool
	errMsg   string
	validate func(t *testing.T, repo repository.Repository, ctx context.Context, setup interface{})
} {
	t.Helper()

	return struct {
		name     string
		setup    func() interface{}
		fn       func(ctx context.Context, tx *domain.Tx, setup interface{}) error
		wantErr  bool
		errMsg   string
		validate func(t *testing.T, repo repository.Repository, ctx context.Context, setup interface{})
	}{
		name: "multiple entities in transaction",
		setup: func() interface{} {
			return nil
		},
		fn: func(ctx context.Context, tx *domain.Tx, setup interface{}) error {
			_ = gofakeit.Seed(time.Now().UnixNano())

			akariUser, err := repo.CreateAkariUser(ctx)
			if err != nil {
				return err
			}

			discordUser := RandomDiscordUser()
			discordUser.AkariUserID = &akariUser.ID
			createdUser, err := repo.CreateDiscordUser(ctx, discordUser)
			if err != nil {
				return err
			}

			guild := RandomDiscordGuild()
			createdGuild, err := repo.CreateDiscordGuild(ctx, guild)
			if err != nil {
				return err
			}

			channel := RandomDiscordChannel(createdGuild.ID)
			createdChannel, err := repo.CreateDiscordChannel(ctx, channel)
			if err != nil {
				return err
			}

			message := RandomDiscordMessage(createdUser.ID, createdChannel.ID)
			createdMessage, err := repo.CreateDiscordMessage(ctx, message)
			if err != nil {
				return err
			}

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
		},
		wantErr: false,
	}
}

func transactionRollbackWithMultipleEntitiesTest(
	t *testing.T,
	repo repository.Repository,
	ctx context.Context,
) struct {
	name     string
	setup    func() interface{}
	fn       func(ctx context.Context, tx *domain.Tx, setup interface{}) error
	wantErr  bool
	errMsg   string
	validate func(t *testing.T, repo repository.Repository, ctx context.Context, setup interface{})
} {
	t.Helper()

	return struct {
		name     string
		setup    func() interface{}
		fn       func(ctx context.Context, tx *domain.Tx, setup interface{}) error
		wantErr  bool
		errMsg   string
		validate func(t *testing.T, repo repository.Repository, ctx context.Context, setup interface{})
	}{
		name: "transaction rollback with multiple entities",
		setup: func() interface{} {
			akariUser, err := repo.CreateAkariUser(ctx)
			require.NoError(t, err)

			discordUserBefore := RandomDiscordUser()
			discordUserBefore.AkariUserID = &akariUser.ID
			createdUserBefore, err := repo.CreateDiscordUser(ctx, discordUserBefore)
			require.NoError(t, err)

			return createdUserBefore.ID
		},
		fn: func(ctx context.Context, tx *domain.Tx, setup interface{}) error {
			guild := RandomDiscordGuild()
			_, err := repo.CreateDiscordGuild(ctx, guild)
			if err != nil {
				return err
			}

			channel := RandomDiscordChannel(guild.ID)
			_, err = repo.CreateDiscordChannel(ctx, channel)
			if err != nil {
				return err
			}

			return errors.New("transaction rollback test")
		},
		wantErr: true,
		errMsg:  "transaction rollback test",
		validate: func(t *testing.T, repo repository.Repository, ctx context.Context, setup interface{}) {
			t.Helper()
			userBeforeID, ok := setup.(string)
			require.True(t, ok, "setup should be string")
			_, err := repo.GetDiscordUserByID(ctx, userBeforeID)
			require.NoError(t, err, "user created before transaction should still exist")
		},
	}
}

func runTransactionTest(
	t *testing.T,
	repo repository.Repository,
	ctx context.Context,
	testCase struct {
		name     string
		setup    func() interface{}
		fn       func(ctx context.Context, tx *domain.Tx, setup interface{}) error
		wantErr  bool
		errMsg   string
		validate func(t *testing.T, repo repository.Repository, ctx context.Context, setup interface{})
	},
) {
	t.Helper()

	setup := testCase.setup()

	err := repo.WithTransaction(ctx, func(ctx context.Context, tx *domain.Tx) error {
		return testCase.fn(ctx, tx, setup)
	})

	if testCase.wantErr {
		require.Error(t, err)

		if testCase.errMsg != "" {
			assert.Contains(t, err.Error(), testCase.errMsg)
		}
	} else {
		require.NoError(t, err)
	}

	if testCase.validate != nil {
		testCase.validate(t, repo, ctx, setup)
	}
}
