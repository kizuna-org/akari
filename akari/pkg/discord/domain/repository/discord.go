package repository

import (
	"context"

	"github.com/kizuna-org/akari/pkg/discord/domain/entity"
)

// DiscordRepository defines the interface for Discord operations
type DiscordRepository interface {
	// SendMessage sends a message to a specific channel
	SendMessage(ctx context.Context, channelID string, content string) (*entity.Message, error)
	
	// GetMessage retrieves a message by its ID from a specific channel
	GetMessage(ctx context.Context, channelID string, messageID string) (*entity.Message, error)
	
	// Start starts the Discord bot session
	Start() error
	
	// Stop stops the Discord bot session
	Stop() error
}

