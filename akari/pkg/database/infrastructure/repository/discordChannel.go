package repository

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/kizuna-org/akari/gen/ent/discordchannel"
	"github.com/kizuna-org/akari/gen/ent/discordguild"
	"github.com/kizuna-org/akari/gen/ent/discordmessage"
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

func (r *repositoryImpl) GetDiscordChannelByMessageID(
	ctx context.Context,
	messageID string,
) (*domain.DiscordChannel, error) {
	channel, err := r.client.DiscordChannelClient().
		Query().
		Where(discordchannel.HasMessagesWith(discordmessage.IDEQ(messageID))).
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get discord channel by message id: %w", err)
	}

	return domain.ToDomainDiscordChannelFromDB(channel), nil
}

func (r *repositoryImpl) GetDiscordChannelsByGuildID(
	ctx context.Context,
	guildID string,
) ([]*domain.DiscordChannel, error) {
	channels, err := r.client.DiscordChannelClient().
		Query().
		Where(discordchannel.HasGuildWith(discordguild.IDEQ(guildID))).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get discord channels by guild id: %w", err)
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
