package handler_test

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/kizuna-org/akari/pkg/discord/domain/entity"
	"github.com/kizuna-org/akari/pkg/discord/handler"
	"github.com/kizuna-org/akari/pkg/discord/infrastructure"
	interactormock "github.com/kizuna-org/akari/pkg/discord/usecase/interactor/mock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func setupHandler(t *testing.T) (*handler.MessageHandler, *interactormock.MockDiscordInteractor) {
	t.Helper()

	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	mockInteractor := interactormock.NewMockDiscordInteractor(ctrl)
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

func TestMessageHandler_HandleMessageCreate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		msg       *discordgo.MessageCreate
		mockSetup func(*interactormock.MockDiscordInteractor)
	}{
		{
			name:      "bot message ignored",
			msg:       createMessage("bot-001", "!ping", true),
			mockSetup: nil,
		},
		{
			name: "ping command",
			msg:  createMessage("user-001", "!ping", false),
			mockSetup: func(m *interactormock.MockDiscordInteractor) {
				m.EXPECT().SendMessage(gomock.Any(), "channel-001", "Pong!").
					Return(&entity.Message{
						ID:        "msg-002",
						Content:   "Pong!",
						Timestamp: time.Now(),
					}, nil)
			},
		},
		{
			name:      "non-ping message",
			msg:       createMessage("user-001", "Hello", false),
			mockSetup: nil,
		},
		{
			name: "send message error",
			msg:  createMessage("user-001", "!ping", false),
			mockSetup: func(m *interactormock.MockDiscordInteractor) {
				m.EXPECT().SendMessage(gomock.Any(), "channel-001", "Pong!").Return(nil, assert.AnError)
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			h, mock := setupHandler(t)
			if testCase.mockSetup != nil {
				testCase.mockSetup(mock)
			}

			h.HandleMessageCreate(nil, testCase.msg)
		})
	}
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
