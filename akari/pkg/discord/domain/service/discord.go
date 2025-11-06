package service

import (
	"context"
	"fmt"

	"github.com/kizuna-org/akari/pkg/discord/domain/entity"
	"github.com/kizuna-org/akari/pkg/discord/domain/repository"
)

// DiscordService defines the domain service for Discord operations
type DiscordService interface {
	SendMessage(ctx context.Context, channelID string, content string) (*entity.Message, error)
	GetMessage(ctx context.Context, channelID string, messageID string) (*entity.Message, error)
}

type discordServiceImpl struct {
	repo repository.DiscordRepository
}

// NewDiscordService creates a new Discord service
func NewDiscordService(repo repository.DiscordRepository) DiscordService {
	return &discordServiceImpl{
		repo: repo,
	}
}

// SendMessage sends a message through the repository
func (s *discordServiceImpl) SendMessage(ctx context.Context, channelID string, content string) (*entity.Message, error) {
	if channelID == "" {
		return nil, fmt.Errorf("channel ID is required")
	}
	if content == "" {
		return nil, fmt.Errorf("message content is required")
	}

	return s.repo.SendMessage(ctx, channelID, content)
}

// GetMessage retrieves a message by its ID
func (s *discordServiceImpl) GetMessage(ctx context.Context, channelID string, messageID string) (*entity.Message, error) {
	if channelID == "" {
		return nil, fmt.Errorf("channel ID is required")
	}
	if messageID == "" {
		return nil, fmt.Errorf("message ID is required")
	}

	return s.repo.GetMessage(ctx, channelID, messageID)
}
