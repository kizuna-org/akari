package domain

//go:generate go tool mockgen -package=mock -source=discordChannel.go -destination=mock/discordChannel.go

import (
	"context"
	"time"

	"github.com/kizuna-org/akari/gen/ent"
)

type DiscordChannelRepository interface {
	CreateDiscordChannel(ctx context.Context, params DiscordChannel) (*DiscordChannel, error)
	GetDiscordChannelByID(ctx context.Context, id string) (*DiscordChannel, error)
	ListDiscordChannels(ctx context.Context) ([]*DiscordChannel, error)
	DeleteDiscordChannel(ctx context.Context, id string) error
}

type DiscordChannel struct {
	ID        string
	Name      string
	Messages  []string
	GuildID   string
	CreatedAt time.Time
}

func ToDomainDiscordChannelFromDB(model *ent.DiscordChannel) *DiscordChannel {
	return &DiscordChannel{
		ID:   model.ID,
		Name: model.Name,
		Messages: func() []string {
			ids := make([]string, len(model.Edges.Messages))
			for i, message := range model.Edges.Messages {
				ids[i] = message.ID
			}

			return ids
		}(),
		GuildID:   model.Edges.Guild.ID,
		CreatedAt: model.CreatedAt,
	}
}
