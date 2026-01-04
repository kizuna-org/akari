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
		SetType(discordchannel.Type(params.Type)).
		SetName(params.Name).
		SetGuildID(params.GuildID)

	channel, err := builder.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create discord channel: %w", err)
	}

	r.logger.Info("Discord channel created",
		slog.String("channel_id", channel.ID),
		slog.String("channel_name", channel.Name),
		slog.String("guild_id", params.GuildID),
	)

	// Create domain object directly since we already have GuildID
	return &domain.DiscordChannel{
		ID:        channel.ID,
		Type:      domain.DiscordChannelType(channel.Type),
		Name:      channel.Name,
		GuildID:   params.GuildID,
		CreatedAt: channel.CreatedAt,
	}, nil
}

func (r *repositoryImpl) GetDiscordChannelByID(
	ctx context.Context,
	channelID string,
) (*domain.DiscordChannel, error) {
	channel, err := r.client.DiscordChannelClient().
		Query().
		Where(discordchannel.IDEQ(channelID)).
		WithGuild().
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get discord channel by id: %w", err)
	}

	return domain.FromEntDiscordChannel(channel)
}

func (r *repositoryImpl) GetDiscordChannelByMessageID(
	ctx context.Context,
	messageID string,
) (*domain.DiscordChannel, error) {
	channel, err := r.client.DiscordChannelClient().
		Query().
		Where(discordchannel.HasMessagesWith(discordmessage.IDEQ(messageID))).
		WithGuild().
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get discord channel: %w", err)
	}

	return domain.FromEntDiscordChannel(channel)
}

func (r *repositoryImpl) GetDiscordChannelsByGuildID(
	ctx context.Context,
	guildID string,
) ([]*domain.DiscordChannel, error) {
	channels, err := r.client.DiscordChannelClient().
		Query().
		Where(discordchannel.HasGuildWith(discordguild.IDEQ(guildID))).
		WithGuild().
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get discord channels: %w", err)
	}

	domainDiscordChannels := make([]*domain.DiscordChannel, len(channels))

	for i, domainDiscordChannel := range channels {
		var err error

		domainDiscordChannels[i], err = domain.FromEntDiscordChannel(domainDiscordChannel)
		if err != nil {
			return nil, fmt.Errorf("failed to convert discord channel: %w", err)
		}
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
