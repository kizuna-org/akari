package handler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/bwmarrin/discordgo"
	databaseDomain "github.com/kizuna-org/akari/pkg/database/domain"
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
	domainMessage, mentions := buildDomainMessage(message)

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

	domainUser, err := fetchUser(session, message)
	if err != nil {
		h.logger.Error("Failed to fetch user", "error", err)

		return
	}

	data := &service.DiscordData{
		User:     domainUser,
		Message:  domainMessage,
		Mentions: mentions,
		Channel:  domainChannel,
		Guild:    domainGuild,
	}

	if err := h.interactor.Handle(ctx, data); err != nil {
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

func buildDomainMessage(message *discordgo.MessageCreate) (*databaseDomain.DiscordMessage, []string) {
	mentions := make([]string, len(message.Mentions))
	for i, mention := range message.Mentions {
		mentions[i] = mention.ID
	}

	return &databaseDomain.DiscordMessage{
		ID:        message.ID,
		ChannelID: message.ChannelID,
		AuthorID:  message.Author.ID,
		Content:   message.Content,
		Timestamp: message.Timestamp,
		CreatedAt: time.Now(),
	}, mentions
}

func fetchChannel(
	session *discordgo.Session,
	message *discordgo.MessageCreate,
) (*databaseDomain.DiscordChannel, error) {
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

	return &databaseDomain.DiscordChannel{
		ID:        discordgoChannel.ID,
		Type:      databaseDomain.DiscordgoChannelTypeToDomainChannelType(discordgoChannel.Type),
		Name:      discordgoChannel.Name,
		GuildID:   discordgoChannel.GuildID,
		CreatedAt: createdAt,
	}, nil
}

func fetchGuild(session *discordgo.Session, message *discordgo.MessageCreate) (*databaseDomain.DiscordGuild, error) {
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

	return &databaseDomain.DiscordGuild{
		ID:         discordgoGuild.ID,
		Name:       discordgoGuild.Name,
		ChannelIDs: []string{},
		CreatedAt:  createdAt,
	}, nil
}

func fetchUser(session *discordgo.Session, message *discordgo.MessageCreate) (*databaseDomain.DiscordUser, error) {
	if session == nil {
		return nil, errors.New("handler: session not found")
	}

	discordgoUser, err := session.User(message.Author.ID)
	if err != nil {
		return nil, fmt.Errorf("handler: %w", err)
	}

	return &databaseDomain.DiscordUser{
		ID:        discordgoUser.ID,
		Username:  discordgoUser.Username,
		Bot:       discordgoUser.Bot,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}
