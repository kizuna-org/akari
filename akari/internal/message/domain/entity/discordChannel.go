package entity

import (
	"time"

	databaseDomain "github.com/kizuna-org/akari/pkg/database/domain"
)

type DiscordChannel struct {
	ID        string
	Type      string
	Name      string
	GuildID   string
	CreatedAt time.Time
}

func (c *DiscordChannel) ToDatabaseDiscordChannel() databaseDomain.DiscordChannel {
	return databaseDomain.DiscordChannel{
		ID:        c.ID,
		Type:      databaseDomain.DiscordChannelType(c.Type),
		Name:      c.Name,
		GuildID:   c.GuildID,
		CreatedAt: c.CreatedAt,
	}
}

func ToDiscordChannel(channel *databaseDomain.DiscordChannel) *DiscordChannel {
	if channel == nil {
		return nil
	}

	return &DiscordChannel{
		ID:        channel.ID,
		Type:      string(channel.Type),
		Name:      channel.Name,
		GuildID:   channel.GuildID,
		CreatedAt: channel.CreatedAt,
	}
}
