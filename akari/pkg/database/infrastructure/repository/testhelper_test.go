package repository_test

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql"
	"github.com/brianvoe/gofakeit/v7"
	"github.com/kizuna-org/akari/gen/ent"
	"github.com/kizuna-org/akari/gen/ent/enttest"
	"github.com/kizuna-org/akari/pkg/config"
	"github.com/kizuna-org/akari/pkg/database/domain"
	"github.com/kizuna-org/akari/pkg/database/infrastructure/repository"
	_ "github.com/lib/pq"
)

// setupTestDB creates a test database client and repository.
func setupTestDB(t *testing.T) (repository.Repository, *ent.Client) {
	t.Helper()

	cfgRepo := config.NewConfigRepository()
	dbCfg := cfgRepo.GetConfig().Database
	dsn := dbCfg.BuildDSN()

	drv, err := sql.Open(dialect.Postgres, dsn)
	if err != nil {
		t.Fatalf("failed to open database connection: %v", err)
	}

	entClient := enttest.Open(t, dialect.Postgres, dsn)

	client := &testClientWrapper{
		Client: entClient,
		driver: drv,
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	repo := repository.NewRepository(client, logger)

	return repo, entClient
}

// testClientWrapper wraps ent.Client to implement infrastructure.Client interface.
type testClientWrapper struct {
	*ent.Client
	driver *sql.Driver
}

func (c *testClientWrapper) Ping(ctx context.Context) error {
	return c.driver.DB().PingContext(ctx)
}

func (c *testClientWrapper) Close() error {
	return c.Client.Close()
}

func (c *testClientWrapper) WithTx(ctx context.Context, txFunc domain.TxFunc) error {
	transaction, err := c.Tx(ctx)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}

	defer func() {
		if v := recover(); v != nil {
			if err := transaction.Rollback(); err != nil {
				panic(fmt.Sprintf("failed to rollback transaction after panic: %v (original panic: %v)", err, v))
			}

			panic(v)
		}
	}()

	if err := txFunc(ctx, transaction); err != nil {
		if rerr := transaction.Rollback(); rerr != nil {
			return fmt.Errorf("failed to rollback transaction: %w (original error: %w)", rerr, err)
		}

		return err
	}

	if err := transaction.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (c *testClientWrapper) CharacterClient() *ent.CharacterClient {
	return c.Character
}

func (c *testClientWrapper) AkariUserClient() *ent.AkariUserClient {
	return c.AkariUser
}

func (c *testClientWrapper) ConversationClient() *ent.ConversationClient {
	return c.Conversation
}

func (c *testClientWrapper) ConversationGroupClient() *ent.ConversationGroupClient {
	return c.ConversationGroup
}

func (c *testClientWrapper) DiscordMessageClient() *ent.DiscordMessageClient {
	return c.DiscordMessage
}

func (c *testClientWrapper) DiscordChannelClient() *ent.DiscordChannelClient {
	return c.DiscordChannel
}

func (c *testClientWrapper) DiscordUserClient() *ent.DiscordUserClient {
	return c.DiscordUser
}

func (c *testClientWrapper) DiscordGuildClient() *ent.DiscordGuildClient {
	return c.DiscordGuild
}

func (c *testClientWrapper) SystemPromptClient() *ent.SystemPromptClient {
	return c.SystemPrompt
}

// RandomDiscordID generates a random Discord ID (18-digit numeric string).
func RandomDiscordID() string {
	_ = gofakeit.Seed(time.Now().UnixNano())

	return gofakeit.Numerify("##################")
}

// RandomDiscordUsername generates a random Discord username.
func RandomDiscordUsername() string {
	_ = gofakeit.Seed(time.Now().UnixNano())

	return gofakeit.Username()
}

// RandomChannelName generates a random channel name.
func RandomChannelName() string {
	_ = gofakeit.Seed(time.Now().UnixNano())

	return gofakeit.Word()
}

// RandomGuildName generates a random guild name.
func RandomGuildName() string {
	_ = gofakeit.Seed(time.Now().UnixNano())

	return gofakeit.Company()
}

// RandomMessageContent generates random message content.
func RandomMessageContent() string {
	_ = gofakeit.Seed(time.Now().UnixNano())

	return gofakeit.Sentence(10)
}

// RandomTimestamp generates a random timestamp.
func RandomTimestamp() time.Time {
	_ = gofakeit.Seed(time.Now().UnixNano())

	return gofakeit.Date()
}

// RandomDiscordUser creates a random DiscordUser domain object.
func RandomDiscordUser() domain.DiscordUser {
	_ = gofakeit.Seed(time.Now().UnixNano())

	return domain.DiscordUser{
		ID:       RandomDiscordID(),
		Username: RandomDiscordUsername(),
		Bot:      gofakeit.Bool(),
	}
}

// RandomDiscordGuild creates a random DiscordGuild domain object.
func RandomDiscordGuild() domain.DiscordGuild {
	_ = gofakeit.Seed(time.Now().UnixNano())

	return domain.DiscordGuild{
		ID:   RandomDiscordID(),
		Name: RandomGuildName(),
	}
}

// RandomDiscordChannel creates a random DiscordChannel domain object.
func RandomDiscordChannel(guildID string) domain.DiscordChannel {
	_ = gofakeit.Seed(time.Now().UnixNano())

	channelTypes := []domain.DiscordChannelType{
		domain.DiscordChannelTypeGuildText,
		domain.DiscordChannelTypeDM,
		domain.DiscordChannelTypeGuildVoice,
		domain.DiscordChannelTypeGuildCategory,
	}

	return domain.DiscordChannel{
		ID:      RandomDiscordID(),
		Type:    channelTypes[gofakeit.IntRange(0, len(channelTypes)-1)],
		Name:    RandomChannelName(),
		GuildID: guildID,
	}
}

// RandomDiscordMessage creates a random DiscordMessage domain object.
func RandomDiscordMessage(authorID, channelID string) domain.DiscordMessage {
	_ = gofakeit.Seed(time.Now().UnixNano())

	return domain.DiscordMessage{
		ID:        RandomDiscordID(),
		AuthorID:  authorID,
		ChannelID: channelID,
		Content:   RandomMessageContent(),
		Timestamp: RandomTimestamp(),
	}
}

// RandomConversation creates a random Conversation domain object.
func RandomConversation(userID int, discordMessageID string, conversationGroupID int) domain.Conversation {
	return domain.Conversation{
		UserID:              userID,
		DiscordMessageID:    discordMessageID,
		ConversationGroupID: conversationGroupID,
	}
}

// TestMain sets up and tears down the test database.
func TestMain(m *testing.M) {
	// Setup: Create a test client for cleanup
	cfgRepo := config.NewConfigRepository()
	dbCfg := cfgRepo.GetConfig().Database
	dsn := dbCfg.BuildDSN()

	drv, err := sql.Open(dialect.Postgres, dsn)
	if err != nil {
		os.Exit(1)
	}

	entClient := ent.NewClient(ent.Driver(drv))

	// Run tests
	code := m.Run()

	// Cleanup: Delete all test data
	cleanupTestDB(entClient)

	// Close connections
	_ = entClient.Close()
	_ = drv.Close()

	os.Exit(code)
}

// cleanupTestDB cleans up all test data from the database.
func cleanupTestDB(client *ent.Client) {
	ctx := context.Background()

	// Delete in reverse order of dependencies
	if _, err := client.Conversation.Delete().Exec(ctx); err != nil {
		// Ignore errors during cleanup
		_ = err
	}

	if _, err := client.ConversationGroup.Delete().Exec(ctx); err != nil {
		_ = err
	}

	if _, err := client.DiscordMessage.Delete().Exec(ctx); err != nil {
		_ = err
	}

	if _, err := client.DiscordChannel.Delete().Exec(ctx); err != nil {
		_ = err
	}

	if _, err := client.DiscordGuild.Delete().Exec(ctx); err != nil {
		_ = err
	}

	if _, err := client.AkariUser.Delete().Exec(ctx); err != nil {
		_ = err
	}

	if _, err := client.DiscordUser.Delete().Exec(ctx); err != nil {
		_ = err
	}
}
