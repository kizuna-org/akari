package interactor

import (
	"context"
	"fmt"

	"github.com/kizuna-org/akari/pkg/discord/domain/entity"
	"github.com/kizuna-org/akari/pkg/discord/domain/service"
)

// DiscordInteractor defines the use case for Discord operations.
type DiscordInteractor interface {
	SendMessage(ctx context.Context, channelID string, content string) (*entity.Message, error)
	GetMessage(ctx context.Context, channelID string, messageID string) (*entity.Message, error)
}

type discordInteractorImpl struct {
	service service.DiscordService
}

// NewDiscordInteractor creates a new Discord interactor.
func NewDiscordInteractor(service service.DiscordService) DiscordInteractor {
	return &discordInteractorImpl{
		service: service,
	}
}

// SendMessage sends a message to a Discord channel.
func (i *discordInteractorImpl) SendMessage(
    ctx context.Context,
    channelID string,
    content string,
) (*entity.Message, error) {
	msg, err := i.service.SendMessage(ctx, channelID, content)
	if err != nil {
		return nil, fmt.Errorf("failed to send message in interactor: %w", err)
	}

	return msg, nil
}

// GetMessage retrieves a message by its ID from a Discord channel.
func (i *discordInteractorImpl) GetMessage(
    ctx context.Context,
    channelID string,
    messageID string,
) (*entity.Message, error) {
	msg, err := i.service.GetMessage(ctx, channelID, messageID)
	if err != nil {
		return nil, fmt.Errorf("failed to get message in interactor: %w", err)
	}

	return msg, nil
}
