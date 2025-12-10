package adapter

import (
	"context"
	"errors"
	"fmt"

	"github.com/kizuna-org/akari/gen/ent"
	"github.com/kizuna-org/akari/internal/message/domain"
	"github.com/kizuna-org/akari/internal/message/domain/entity"
	databaseInteractor "github.com/kizuna-org/akari/pkg/database/usecase/interactor"
)

type discordChannelRepository struct {
	discordChannelInteractor databaseInteractor.DiscordChannelInteractor
}

func NewDiscordChannelRepository(
	discordChannelInteractor databaseInteractor.DiscordChannelInteractor,
) domain.DiscordChannelRepository {
	return &discordChannelRepository{
		discordChannelInteractor: discordChannelInteractor,
	}
}

func (r *discordChannelRepository) CreateIfNotExists(ctx context.Context, channel *entity.Channel) (string, error) {
	if channel == nil {
		return "", errors.New("adapter: channel is required")
	}

	if _, err := r.discordChannelInteractor.GetDiscordChannelByID(ctx, channel.ID); err == nil {
		return channel.ID, nil
	} else if !ent.IsNotFound(err) {
		return "", fmt.Errorf("adapter: failed to get discord channel by id: %w", err)
	}

	discordChannel, err := r.discordChannelInteractor.CreateDiscordChannel(ctx, channel.ToDiscordChannel())
	if err != nil {
		return "", fmt.Errorf("adapter: failed to create discord channel: %w", err)
	}

	return discordChannel.ID, nil
}
