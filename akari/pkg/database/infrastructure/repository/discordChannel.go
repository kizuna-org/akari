package repository

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/kizuna-org/akari/pkg/database/domain"
)

func (r *repositoryImpl) CreateDiscordChannel(
	ctx context.Context,
	params domain.DiscordChannel,
) (*domain.DiscordChannel, error) {
	builder := r.client.DiscordChannelClient().Create().
		SetID(params.ID).
		SetName(params.Name).
		SetGuildID(params.GuildID)

	channel, err := builder.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create discord channel: %w", err)
	}

	r.logger.Info("Discord channel created",
		slog.String("channel_id", channel.ID),
		slog.String("channel_id", channel.Name),
		slog.String("author_id", channel.Edges.Guild.ID),
	)

	return domain.ToDomainDiscordChannelFromDB(channel), nil
}

func (r *repositoryImpl) GetDiscordChannelByID(
	ctx context.Context,
	channelID string,
) (*domain.DiscordChannel, error) {
	channel, err := r.client.DiscordChannelClient().Get(ctx, channelID)
	if err != nil {
		return nil, fmt.Errorf("failed to get discord channel by id: %w", err)
	}

	return domain.ToDomainDiscordChannelFromDB(channel), nil
}

func (r *repositoryImpl) ListDiscordChannels(ctx context.Context) ([]*domain.DiscordChannel, error) {
	channels, err := r.client.DiscordChannelClient().Query().All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list discord channels: %w", err)
	}

	domainDiscordChannels := make([]*domain.DiscordChannel, 0, len(channels))
	for _, domainDiscordChannel := range channels {
		domainDiscordChannels = append(domainDiscordChannels, domain.ToDomainDiscordChannelFromDB(domainDiscordChannel))
	}

	return domainDiscordChannels, nil
}

func (r *repositoryImpl) DeleteDiscordChannel(ctx context.Context, channelID string) error {
	if err := r.client.DiscordChannelClient().DeleteOneID(channelID).Exec(ctx); err != nil {
		return fmt.Errorf("failed to delete discord channel: %w", err)
	}

	r.logger.Info("Discord channel deleted",
		slog.String("id", channelID),
	)

	return nil
}
