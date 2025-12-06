package handler_test

import (
	"log/slog"
	"os"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/kizuna-org/akari/pkg/discord/domain/service/mock"
	"github.com/kizuna-org/akari/pkg/discord/handler"
	"github.com/kizuna-org/akari/pkg/discord/infrastructure"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func setupHandler(t *testing.T) (*handler.MessageHandler, *mock.MockHandleMessageInteractor) {
	t.Helper()

	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	mockInteractor := mock.NewMockHandleMessageInteractor(ctrl)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	client, err := infrastructure.NewDiscordClient("test-token")
	if err != nil {
		t.Fatalf("failed to create discord client: %v", err)
	}

	return handler.NewMessageHandler(mockInteractor, logger, client), mockInteractor
}

func createMessage(authorID, content string, isBot bool) *discordgo.MessageCreate {
	return &discordgo.MessageCreate{
		Message: &discordgo.Message{
			ID:        "msg-001",
			ChannelID: "channel-001",
			Content:   content,
			Author:    &discordgo.User{ID: authorID, Bot: isBot},
		},
	}
}

func TestNewMessageHandler(t *testing.T) {
	t.Parallel()

	h, _ := setupHandler(t)
	assert.NotNil(t, h)
}

func TestMessageHandler_HandleMessageCreate_BotMessageIgnored(t *testing.T) {
	t.Parallel()

	h, mock := setupHandler(t)
	msg := createMessage("bot-001", "test", true)

	mock.EXPECT().Handle(gomock.Any(), gomock.Any()).Return(nil)

	h.HandleMessageCreate(nil, msg)
}

func TestMessageHandler_HandleMessageCreate_Success(t *testing.T) {
	t.Parallel()

	h, mock := setupHandler(t)
	msg := createMessage("user-001", "Hello", false)

	mock.EXPECT().Handle(gomock.Any(), gomock.Any()).Return(nil)

	h.HandleMessageCreate(nil, msg)
}

func TestMessageHandler_HandleMessageCreate_Error(t *testing.T) {
	t.Parallel()

	h, mock := setupHandler(t)
	msg := createMessage("user-001", "Hello", false)

	mock.EXPECT().Handle(gomock.Any(), gomock.Any()).Return(assert.AnError)

	h.HandleMessageCreate(nil, msg)
}

func TestMessageHandler_RegisterHandlers(t *testing.T) {
	t.Parallel()

	h, _ := setupHandler(t)
	h.RegisterHandlers()
}

func TestMessageHandler_GetSession(t *testing.T) {
	t.Parallel()

	h, _ := setupHandler(t)
	session := h.GetSession()
	assert.NotNil(t, session)
}
