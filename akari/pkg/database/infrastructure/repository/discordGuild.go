package repository

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/kizuna-org/akari/gen/ent/discordchannel"
	"github.com/kizuna-org/akari/gen/ent/discordguild"
	"github.com/kizuna-org/akari/pkg/database/domain"
)

func (r *repositoryImpl) CreateDiscordGuild(
	ctx context.Context,
	params domain.DiscordGuild,
) (*domain.DiscordGuild, error) {
	builder := r.client.DiscordGuildClient().Create().
		SetID(params.ID).
		SetName(params.Name)

	guild, err := builder.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create discord guild: %w", err)
	}

	r.logger.Info("Discord guild created",
		slog.String("guild_id", guild.ID),
		slog.String("guild_id", guild.Name),
	)

	return domain.FromEntDiscordGuild(guild)
}
func (r *repositoryImpl) GetDiscordGuildByID(
	ctx context.Context,
	guildID string,
) (*domain.DiscordGuild, error) {
	guild, err := r.client.DiscordGuildClient().Get(ctx, guildID)
	if err != nil {
		return nil, fmt.Errorf("failed to get discord guild by id: %w", err)
	}

	return domain.FromEntDiscordGuild(guild)
}

func (r *repositoryImpl) GetDiscordGuildByChannelID(
	ctx context.Context,
	channelID string,
) (*domain.DiscordGuild, error) {
	guild, err := r.client.DiscordGuildClient().
		Query().
		Where(discordguild.HasChannelsWith(discordchannel.IDEQ(channelID))).
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get discord channel: %w", err)
	}

	return domain.FromEntDiscordGuild(guild)
}

func (r *repositoryImpl) ListDiscordGuilds(ctx context.Context) ([]*domain.DiscordGuild, error) {
	guilds, err := r.client.DiscordGuildClient().Query().WithChannels().All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list discord guilds: %w", err)
	}

	domainDiscordGuilds := make([]*domain.DiscordGuild, len(guilds))

	for i, domainDiscordGuild := range guilds {
		var err error

		domainDiscordGuilds[i], err = domain.FromEntDiscordGuild(domainDiscordGuild)
		if err != nil {
			return nil, fmt.Errorf("failed to convert discord guild: %w", err)
		}
	}

	return domainDiscordGuilds, nil
}

func (r *repositoryImpl) DeleteDiscordGuild(ctx context.Context, guildID string) error {
	if err := r.client.DiscordGuildClient().DeleteOneID(guildID).Exec(ctx); err != nil {
		return fmt.Errorf("failed to delete discord guild: %w", err)
	}

	r.logger.Info("Discord guild deleted",
		slog.String("id", guildID),
	)

	return nil
}
