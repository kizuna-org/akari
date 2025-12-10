package handler

import (
	"context"
	"errors"
	"fmt"
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

func (h *MessageHandler) HandleMessageCreate(session *discordgo.Session, message *discordgo.MessageCreate) {
	h.logger.Info("Received message",
		"author", message.Author.Username,
		"content", message.Content,
		"channel_id", message.ChannelID,
		"message_id", message.ID,
		"is_bot", message.Author.Bot,
	)

	ctx := context.Background()
	domainMessage := buildDomainMessage(message)

	domainChannel, err := fetchChannel(session, message)
	if err != nil {
		h.logger.Error("Failed to fetch channel", "error", err)

		return
	}

	domainGuild, err := fetchGuild(session, message)
	if err != nil {
		h.logger.Error("Failed to fetch guild", "error", err)

		return
	}

	if err := h.interactor.Handle(ctx, domainMessage, domainChannel, domainGuild); err != nil {
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

func fetchChannel(session *discordgo.Session, message *discordgo.MessageCreate) (*entity.Channel, error) {
	if session == nil {
		return nil, errors.New("handler: session not found")
	}

	discordgoChannel, err := session.Channel(message.ChannelID)
	if err != nil {
		return nil, fmt.Errorf("handler: failed to fetch channel: %w", err)
	}

	createdAt, err := discordgo.SnowflakeTimestamp(discordgoChannel.ID)
	if err != nil {
		return nil, fmt.Errorf("handler: failed to parse channel timestamp: %w", err)
	}

	return &entity.Channel{
		ID:        discordgoChannel.ID,
		Type:      int(discordgoChannel.Type),
		Name:      discordgoChannel.Name,
		GuildID:   discordgoChannel.GuildID,
		CreatedAt: createdAt,
	}, nil
}

func fetchGuild(session *discordgo.Session, message *discordgo.MessageCreate) (*entity.Guild, error) {
	if session == nil {
		return nil, errors.New("handler: session not found")
	}

	discordgoGuild, err := session.Guild(message.GuildID)
	if err != nil {
		return nil, fmt.Errorf("handler: failed to fetch guild: %w", err)
	}

	createdAt, err := discordgo.SnowflakeTimestamp(discordgoGuild.ID)
	if err != nil {
		return nil, fmt.Errorf("handler: failed to parse guild timestamp: %w", err)
	}

	return &entity.Guild{
		ID:        discordgoGuild.ID,
		Name:      discordgoGuild.Name,
		CreatedAt: createdAt,
	}, nil
}
