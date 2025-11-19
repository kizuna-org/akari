package domain

//go:generate go tool mockgen -package=mock -source=discordGuild.go -destination=mock/discordGuild.go

import (
	"context"
	"errors"
	"time"

	"github.com/kizuna-org/akari/gen/ent"
)

type DiscordGuildRepository interface {
	CreateDiscordGuild(ctx context.Context, params DiscordGuild) (*DiscordGuild, error)
	GetDiscordGuildByID(ctx context.Context, id string) (*DiscordGuild, error)
	GetDiscordGuildByChannelID(ctx context.Context, channelID string) (*DiscordGuild, error)
	ListDiscordGuilds(ctx context.Context) ([]*DiscordGuild, error)
	DeleteDiscordGuild(ctx context.Context, id string) error
}

type DiscordGuild struct {
	ID         string
	Name       string
	ChannelIDs []string
	CreatedAt  time.Time
}

func FromEntDiscordGuild(entDiscordGuild *ent.DiscordGuild) (*DiscordGuild, error) {
	if entDiscordGuild == nil {
		return nil, errors.New("discordGuild is nil")
	}

	if entDiscordGuild.Edges.Channels == nil {
		return nil, errors.New("discordGuild.Channels edge is nil")
	}

	discordChannelIDs := make([]string, len(entDiscordGuild.Edges.Channels))
	for i, discordChannel := range entDiscordGuild.Edges.Channels {
		discordChannelIDs[i] = discordChannel.ID
	}

	return &DiscordGuild{
		ID:         entDiscordGuild.ID,
		Name:       entDiscordGuild.Name,
		ChannelIDs: discordChannelIDs,
		CreatedAt:  entDiscordGuild.CreatedAt,
	}, nil
}
