package handler

import (
	"context"
	"log/slog"

	"github.com/bwmarrin/discordgo"
	"github.com/kizuna-org/akari/pkg/discord/domain/entity"
	"github.com/kizuna-org/akari/pkg/discord/domain/service"
	"github.com/kizuna-org/akari/pkg/discord/infrastructure"
)

type MessageHandler struct {
	interactor service.HandleMessageInteractor
	logger     *slog.Logger
	client     *infrastructure.DiscordClient
}

func NewMessageHandler(
	interactor service.HandleMessageInteractor,
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
	h.logger.Info("Received message",
		"author", message.Author.Username,
		"content", message.Content,
		"channel_id", message.ChannelID,
		"message_id", message.ID,
		"is_bot", message.Author.Bot,
	)

	ctx := context.Background()
	domainMessage := buildDomainMessage(message)

	if err := h.interactor.Handle(ctx, domainMessage); err != nil {
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

func buildDomainMessage(message *discordgo.MessageCreate) *entity.Message {
	mentions := make([]string, len(message.Mentions))
	for i, mention := range message.Mentions {
		mentions[i] = mention.ID
	}

	return &entity.Message{
		ID:        message.ID,
		ChannelID: message.ChannelID,
		GuildID:   message.GuildID,
		AuthorID:  message.Author.ID,
		Content:   message.Content,
		Timestamp: message.Timestamp,
		IsBot:     message.Author.Bot,
		Mentions:  mentions,
	}
}
