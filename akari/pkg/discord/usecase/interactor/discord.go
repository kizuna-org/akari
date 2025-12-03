package interactor

//go:generate go tool mockgen -package=mock -source=discord.go -destination=mock/discord.go

import (
	"context"
	"fmt"

	databaseDomain "github.com/kizuna-org/akari/pkg/database/domain"
	"github.com/kizuna-org/akari/pkg/discord/domain/entity"
	"github.com/kizuna-org/akari/pkg/discord/domain/service"
)

type DiscordInteractor interface {
	SendMessage(ctx context.Context, channelID string, content string) (*entity.Message, error)
	GetMessage(ctx context.Context, channelID string, messageID string) (*entity.Message, error)
	SaveMessage(ctx context.Context, message *entity.Message) error
}

type discordInteractorImpl struct {
	service           service.DiscordService
	messageRepository databaseDomain.DiscordMessageRepository
}

func NewDiscordInteractor(
	service service.DiscordService,
	messageRepository databaseDomain.DiscordMessageRepository,
) DiscordInteractor {
	return &discordInteractorImpl{
		service:           service,
		messageRepository: messageRepository,
	}
}

func (i *discordInteractorImpl) SendMessage(
	ctx context.Context,
	channelID string,
	content string,
) (*entity.Message, error) {
	msg, err := i.service.SendMessage(ctx, channelID, content)
	if err != nil {
		return nil, fmt.Errorf("failed to send message in interactor: %w", err)
	}

	if err := i.SaveMessage(ctx, msg); err != nil {
		return nil, err
	}

	return msg, nil
}

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

func (i *discordInteractorImpl) SaveMessage(ctx context.Context, message *entity.Message) error {
	if _, err := i.messageRepository.CreateDiscordMessage(ctx, databaseDomain.DiscordMessage{
		ID:        message.ID,
		ChannelID: message.ChannelID,
		AuthorID:  message.AuthorID,
		Content:   message.Content,
		Timestamp: message.Timestamp,
		CreatedAt: message.Timestamp,
	}); err != nil {
		return fmt.Errorf("failed to save message to database: %w", err)
	}

	return nil
}
