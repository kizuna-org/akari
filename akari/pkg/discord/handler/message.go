package handler

import (
	"context"
	"log/slog"

	"github.com/bwmarrin/discordgo"
	"github.com/kizuna-org/akari/pkg/discord/domain/entity"
	"github.com/kizuna-org/akari/pkg/discord/infrastructure"
	"github.com/kizuna-org/akari/pkg/discord/usecase/interactor"
)

type MessageHandler struct {
	interactor interactor.DiscordInteractor
	logger     *slog.Logger
	client     *infrastructure.DiscordClient
}

func NewMessageHandler(
	interactor interactor.DiscordInteractor,
	logger *slog.Logger,
	client *infrastructure.DiscordClient,
) *MessageHandler {
	return &MessageHandler{
		interactor: interactor,
		logger:     logger,
		client:     client,
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

	if err := h.interactor.SaveMessage(ctx, &entity.Message{
		ID:        message.ID,
		ChannelID: message.ChannelID,
		GuildID:   message.GuildID,
		AuthorID:  message.Author.ID,
		Content:   message.Content,
		Timestamp: message.Timestamp,
	}); err != nil {
		h.logger.Error("Failed to save received message", "error", err)
	}

	if message.Content == "!ping" {
		_, err := h.interactor.SendMessage(ctx, message.ChannelID, "Pong!")
		if err != nil {
			h.logger.Error("Failed to send response", "error", err)
		}
	}
}

func (h *MessageHandler) RegisterHandlers() {
	h.client.Session.AddHandler(h.HandleMessageCreate)
	h.logger.Info("Message handlers registered")
}

func (h *MessageHandler) GetSession() *discordgo.Session {
	return h.client.Session
}
