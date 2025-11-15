package interactor

import (
	"context"

	"github.com/kizuna-org/akari/pkg/database/domain"
)

type DiscordMessageInteractor interface {
	CreateDiscordMessage(
		ctx context.Context,
		params domain.DiscordMessage,
	) (*domain.DiscordMessage, error)
	GetDiscordMessageByID(
		ctx context.Context,
		messageID string,
	) (*domain.DiscordMessage, error)
	DeleteDiscordMessage(ctx context.Context, id string) error
}

type discordMessageInteractorImpl struct {
	repository domain.DiscordMessageRepository
}

func NewDiscordMessageInteractor(repository domain.DiscordMessageRepository) DiscordMessageInteractor {
	return &discordMessageInteractorImpl{
		repository: repository,
	}
}

func (d *discordMessageInteractorImpl) CreateDiscordMessage(
	ctx context.Context,
	params domain.DiscordMessage,
) (*domain.DiscordMessage, error) {
	return d.repository.CreateDiscordMessage(ctx, params)
}

func (d *discordMessageInteractorImpl) GetDiscordMessageByID(
	ctx context.Context,
	messageID string,
) (*domain.DiscordMessage, error) {
	return d.repository.GetDiscordMessageByID(ctx, messageID)
}

func (d *discordMessageInteractorImpl) DeleteDiscordMessage(ctx context.Context, id string) error {
	return d.repository.DeleteDiscordMessage(ctx, id)
}
