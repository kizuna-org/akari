package domain

//go:generate go tool mockgen -package=mock -source=discordGuild.go -destination=mock/discordGuild.go

import (
	"context"
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
	ID        string
	Name      string
	Channels  []*DiscordChannel
	CreatedAt time.Time
}

func FromEntDiscordGuild(entDiscordGuild *ent.DiscordGuild) *DiscordGuild {
	if entDiscordGuild == nil {
		return nil
	}

	var discordChannels []*DiscordChannel
	if entDiscordGuild.Edges.Channels != nil {
		discordChannels = make([]*DiscordChannel, len(entDiscordGuild.Edges.Channels))
		for i, discordChannel := range entDiscordGuild.Edges.Channels {
			discordChannels[i] = FromEntDiscordChannel(discordChannel)
		}
	}

	return &DiscordGuild{
		ID:        entDiscordGuild.ID,
		Name:      entDiscordGuild.Name,
		Channels:  discordChannels,
		CreatedAt: entDiscordGuild.CreatedAt,
	}
}
