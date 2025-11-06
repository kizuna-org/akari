package handler

import (
	"context"
	"log/slog"

	"github.com/bwmarrin/discordgo"
	"github.com/kizuna-org/akari/pkg/discord/infrastructure"
	"github.com/kizuna-org/akari/pkg/discord/usecase/interactor"
)

// MessageHandler handles Discord message events
type MessageHandler struct {
	interactor interactor.DiscordInteractor
	logger     *slog.Logger
	client     *infrastructure.DiscordClient
}

// NewMessageHandler creates a new message handler
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

// HandleMessageCreate handles message creation events
func (h *MessageHandler) HandleMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore messages from bots
	if m.Author.Bot {
		return
	}

	h.logger.Info("Received message",
		"author", m.Author.Username,
		"content", m.Content,
		"channel_id", m.ChannelID,
		"message_id", m.ID,
	)

	// You can add custom logic here to process messages
	// For example, respond to specific commands, etc.
	ctx := context.Background()

	// Example: Echo the message back
	if m.Content == "!ping" {
		_, err := h.interactor.SendMessage(ctx, m.ChannelID, "Pong!")
		if err != nil {
			h.logger.Error("Failed to send response", "error", err)
		}
	}
}

// RegisterHandlers registers all message handlers to the Discord session
func (h *MessageHandler) RegisterHandlers() {
	h.client.Session.AddHandler(h.HandleMessageCreate)
	h.logger.Info("Message handlers registered")
}

// GetSession returns the Discord session
func (h *MessageHandler) GetSession() *discordgo.Session {
	return h.client.Session
}

