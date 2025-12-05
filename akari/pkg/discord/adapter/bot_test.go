package adapter_test

import (
	"log/slog"
	"os"
	"testing"

	"github.com/kizuna-org/akari/pkg/discord/adapter"
	repomock "github.com/kizuna-org/akari/pkg/discord/domain/repository/mock"
	"github.com/kizuna-org/akari/pkg/discord/domain/service/mock"
	"github.com/kizuna-org/akari/pkg/discord/handler"
	"github.com/kizuna-org/akari/pkg/discord/infrastructure"
	"github.com/stretchr/testify/assert"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
	"go.uber.org/mock/gomock"
)

func TestNewBotRunner(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	mockInteractor := mock.NewMockHandleMessageInteractor(ctrl)
	mockRepo := repomock.NewMockDiscordRepository(ctrl)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	client, err := infrastructure.NewDiscordClient("test-token")
	if err != nil {
		t.Fatalf("failed to create discord client: %v", err)
	}

	msgHandler := handler.NewMessageHandler(mockInteractor, logger, client)
	runner := adapter.NewBotRunner(msgHandler, mockRepo, logger)

	assert.NotNil(t, runner)
}

func TestBotRunner_RegisterLifecycle(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	mockInteractor := mock.NewMockHandleMessageInteractor(ctrl)
	mockRepo := repomock.NewMockDiscordRepository(ctrl)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	client, err := infrastructure.NewDiscordClient("test-token")
	if err != nil {
		t.Fatalf("failed to create discord client: %v", err)
	}

	msgHandler := handler.NewMessageHandler(mockInteractor, logger, client)
	runner := adapter.NewBotRunner(msgHandler, mockRepo, logger)

	mockRepo.EXPECT().Start().Return(nil)
	mockRepo.EXPECT().Stop().Return(nil)

	app := fxtest.New(t,
		fx.Invoke(func(lc fx.Lifecycle) {
			runner.RegisterLifecycle(lc)
		}),
	)

	app.RequireStart()
	app.RequireStop()
}

func TestBotRunner_RegisterLifecycle_StartError(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	mockInteractor := mock.NewMockHandleMessageInteractor(ctrl)
	mockRepo := repomock.NewMockDiscordRepository(ctrl)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	client, err := infrastructure.NewDiscordClient("test-token")
	if err != nil {
		t.Fatalf("failed to create discord client: %v", err)
	}

	msgHandler := handler.NewMessageHandler(mockInteractor, logger, client)
	runner := adapter.NewBotRunner(msgHandler, mockRepo, logger)

	mockRepo.EXPECT().Start().Return(assert.AnError)

	app := fxtest.New(t,
		fx.Invoke(func(lc fx.Lifecycle) {
			runner.RegisterLifecycle(lc)
		}),
	)

	err = app.Start(t.Context())
	assert.Error(t, err)
}
