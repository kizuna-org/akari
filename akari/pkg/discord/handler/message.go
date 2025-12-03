package handler

import (
	"context"
	"log/slog"

	"github.com/bwmarrin/discordgo"
	"github.com/kizuna-org/akari/internal/message/domain"
	"github.com/kizuna-org/akari/internal/message/usecase"
	"github.com/kizuna-org/akari/pkg/discord/infrastructure"
)

type MessageHandler struct {
	handleMessageInteractor usecase.HandleMessageInteractor
	logger                  *slog.Logger
	client                  *infrastructure.DiscordClient
}

func NewMessageHandler(
	handleMessageInteractor usecase.HandleMessageInteractor,
	logger *slog.Logger,
	client *infrastructure.DiscordClient,
) *MessageHandler {
	return &MessageHandler{
		handleMessageInteractor: handleMessageInteractor,
		logger:                  logger,
		client:                  client,
	}
}

func (h *MessageHandler) HandleMessageCreate(s *discordgo.Session, message *discordgo.MessageCreate) {
	if message.Author.Bot {
		return
	}

	h.logger.Info("Received message",
		"author", message.Author.Username,
		"content", message.Content,
		"channel_id", message.ChannelID,
		"message_id", message.ID,
	)

	ctx := context.Background()

	msg := &domain.Message{
		ID:        message.ID,
		ChannelID: message.ChannelID,
		GuildID:   message.GuildID,
		AuthorID:  message.Author.ID,
		Content:   message.Content,
		Timestamp: message.Timestamp,
	}

	if err := h.handleMessageInteractor.Handle(ctx, msg); err != nil {
		h.logger.Error("Failed to handle message", "error", err)
	}
}

func (h *MessageHandler) RegisterHandlers() {
	h.client.Session.AddHandler(h.HandleMessageCreate)
	h.logger.Info("Message handlers registered")
}

func (h *MessageHandler) GetSession() *discordgo.Session {
	return h.client.Session
}
