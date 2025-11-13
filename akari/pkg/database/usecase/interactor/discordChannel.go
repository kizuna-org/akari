//nolint:dupl
package interactor

import (
	"context"

	"github.com/kizuna-org/akari/pkg/database/domain"
)

type DiscordChannelInteractor interface {
	CreateDiscordChannel(
		ctx context.Context,
		params domain.DiscordChannel,
	) (*domain.DiscordChannel, error)
	GetDiscordChannelByID(
		ctx context.Context,
		channelID string,
	) (*domain.DiscordChannel, error)
	ListDiscordChannels(ctx context.Context) ([]*domain.DiscordChannel, error)
	DeleteDiscordChannel(ctx context.Context, id string) error
}

type discordChannelInteractorImpl struct {
	repository domain.DiscordChannelRepository
}

func NewDiscordChannelInteractor(repository domain.DiscordChannelRepository) DiscordChannelInteractor {
	return &discordChannelInteractorImpl{
		repository: repository,
	}
}

func (d *discordChannelInteractorImpl) CreateDiscordChannel(
	ctx context.Context,
	params domain.DiscordChannel,
) (*domain.DiscordChannel, error) {
	return d.repository.CreateDiscordChannel(ctx, params)
}

func (d *discordChannelInteractorImpl) GetDiscordChannelByID(
	ctx context.Context,
	channelID string,
) (*domain.DiscordChannel, error) {
	return d.repository.GetDiscordChannelByID(ctx, channelID)
}

func (d *discordChannelInteractorImpl) ListDiscordChannels(ctx context.Context) ([]*domain.DiscordChannel, error) {
	return d.repository.ListDiscordChannels(ctx)
}

func (d *discordChannelInteractorImpl) DeleteDiscordChannel(ctx context.Context, id string) error {
	return d.repository.DeleteDiscordChannel(ctx, id)
}
