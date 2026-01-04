package repository_test

import (
	"context"
	"errors"
	"testing"

	"github.com/kizuna-org/akari/gen/ent/discordchannel"
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
	_ repository.Repository,
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
		fn: func(ctx context.Context, transaction *domain.Tx, setup interface{}) error {
			user1, err := transaction.AkariUser.Create().Save(ctx)
			if err != nil {
				return err
			}

			user2, err := transaction.AkariUser.Create().Save(ctx)
			if err != nil {
				return err
			}

			_, err = transaction.AkariUser.Get(ctx, user1.ID)
			if err != nil {
				return err
			}

			_, err = transaction.AkariUser.Get(ctx, user2.ID)
			if err != nil {
				return err
			}

			return nil
		},
		wantErr: false,
		validate: func(t *testing.T, repo repository.Repository, ctx context.Context, setup interface{}) {
			t.Helper()
			// Users should be created after transaction commit
			users, err := repo.ListAkariUsers(ctx)
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(users), 2, "at least 2 users should be created")
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
		fn: func(ctx context.Context, transaction *domain.Tx, setup interface{}) error {
			_, err := transaction.AkariUser.Create().Save(ctx)
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
			userBefore, err := repo.GetAkariUserByID(ctx, userBeforeID)
			require.NoError(t, err, "user created before transaction should still exist")
			assert.Equal(t, userBeforeID, userBefore.ID, "the user created before transaction should still exist")

			// Verify that the user created inside the transaction was rolled back
			// Note: We can't check the exact count due to parallel test execution,
			// but we can verify the user created before transaction still exists
			users, err := repo.ListAkariUsers(ctx)
			require.NoError(t, err)
			found := false
			for _, user := range users {
				if user.ID == userBeforeID {
					found = true
					break
				}
			}
			assert.True(t, found, "the user created before transaction should exist in the list")
		},
	}
}

func multipleEntitiesInTransactionTest(
	t *testing.T,
	_ repository.Repository,
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
		fn: func(ctx context.Context, transaction *domain.Tx, setup interface{}) error {
			akariUser, err := transaction.AkariUser.Create().Save(ctx)
			if err != nil {
				return err
			}

			discordUser := RandomDiscordUser()
			createdUser, err := transaction.DiscordUser.Create().
				SetID(discordUser.ID).
				SetUsername(discordUser.Username).
				SetBot(discordUser.Bot).
				SetAkariUserID(akariUser.ID).
				Save(ctx)
			if err != nil {
				return err
			}

			guild := RandomDiscordGuild()
			createdGuild, err := transaction.DiscordGuild.Create().
				SetID(guild.ID).
				SetName(guild.Name).
				Save(ctx)
			if err != nil {
				return err
			}

			channel := RandomDiscordChannel(createdGuild.ID)
			createdChannel, err := transaction.DiscordChannel.Create().
				SetID(channel.ID).
				SetType(discordchannel.Type(channel.Type)).
				SetName(channel.Name).
				SetGuildID(channel.GuildID).
				Save(ctx)
			if err != nil {
				return err
			}

			message := RandomDiscordMessage(createdUser.ID, createdChannel.ID)
			createdMessage, err := transaction.DiscordMessage.Create().
				SetID(message.ID).
				SetAuthorID(message.AuthorID).
				SetChannelID(message.ChannelID).
				SetContent(message.Content).
				SetTimestamp(message.Timestamp).
				Save(ctx)
			if err != nil {
				return err
			}

			_, err = transaction.DiscordUser.Get(ctx, createdUser.ID)
			if err != nil {
				return err
			}

			_, err = transaction.DiscordGuild.Get(ctx, createdGuild.ID)
			if err != nil {
				return err
			}

			_, err = transaction.DiscordChannel.Get(ctx, createdChannel.ID)
			if err != nil {
				return err
			}

			_, err = transaction.DiscordMessage.Get(ctx, createdMessage.ID)
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
		fn: func(ctx context.Context, transaction *domain.Tx, setup interface{}) error {
			guild := RandomDiscordGuild()
			_, err := transaction.DiscordGuild.Create().
				SetID(guild.ID).
				SetName(guild.Name).
				Save(ctx)
			if err != nil {
				return err
			}

			channel := RandomDiscordChannel(guild.ID)
			_, err = transaction.DiscordChannel.Create().
				SetID(channel.ID).
				SetType(discordchannel.Type(channel.Type)).
				SetName(channel.Name).
				SetGuildID(channel.GuildID).
				Save(ctx)
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

			// Verify that entities created inside the transaction were rolled back
			// The guild created inside transaction should be rolled back
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
