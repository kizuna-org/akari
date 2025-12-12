package interactor

//go:generate go tool mockgen -package=mock -source=discord.go -destination=mock/discord.go

import (
	"context"
	"fmt"

	databaseDomain "github.com/kizuna-org/akari/pkg/database/domain"
	"github.com/kizuna-org/akari/pkg/discord/domain/service"
)

type DiscordInteractor interface {
	SendMessage(ctx context.Context, channelID string, content string) (*databaseDomain.DiscordMessage, error)
	GetMessage(ctx context.Context, channelID string, messageID string) (*databaseDomain.DiscordMessage, error)
}

type discordInteractorImpl struct {
	service service.DiscordService
}

func NewDiscordInteractor(service service.DiscordService) DiscordInteractor {
	return &discordInteractorImpl{
		service: service,
	}
}

func (i *discordInteractorImpl) SendMessage(
	ctx context.Context,
	channelID string,
	content string,
) (*databaseDomain.DiscordMessage, error) {
	msg, err := i.service.SendMessage(ctx, channelID, content)
	if err != nil {
		return nil, fmt.Errorf("failed to send message in interactor: %w", err)
	}

	return msg, nil
}

func (i *discordInteractorImpl) GetMessage(
	ctx context.Context,
	channelID string,
	messageID string,
) (*databaseDomain.DiscordMessage, error) {
	msg, err := i.service.GetMessage(ctx, channelID, messageID)
	if err != nil {
		return nil, fmt.Errorf("failed to get message in interactor: %w", err)
	}

	return msg, nil
}
