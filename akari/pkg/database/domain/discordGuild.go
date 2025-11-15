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
	ID         string
	Name       string
	ChannelIDs []string
	CreatedAt  time.Time
}

func ToDomainDiscordGuildFromDB(model *ent.DiscordGuild) *DiscordGuild {
	return &DiscordGuild{
		ID:   model.ID,
		Name: model.Name,
		ChannelIDs: func() []string {
			ids := make([]string, len(model.Edges.Channels))
			for i, channel := range model.Edges.Channels {
				ids[i] = channel.ID
			}

			return ids
		}(),
		CreatedAt: model.CreatedAt,
	}
}
