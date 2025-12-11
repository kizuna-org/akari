package entity

import (
	"strconv"
	"time"

	databaseDomain "github.com/kizuna-org/akari/pkg/database/domain"
	discordEntity "github.com/kizuna-org/akari/pkg/discord/domain/entity"
)

type DiscordChannel struct {
	ID        string
	Type      int
	Name      string
	GuildID   string
	CreatedAt time.Time
}

func (c *DiscordChannel) ToDatabaseDiscordChannel() databaseDomain.DiscordChannel {
	return databaseDomain.DiscordChannel{
		ID:        c.ID,
		Type:      databaseDomain.DiscordChannelType(strconv.Itoa(c.Type)),
		Name:      c.Name,
		GuildID:   c.GuildID,
		CreatedAt: c.CreatedAt,
	}
}

func ToDiscordChannel(channel *discordEntity.Channel) *DiscordChannel {
	if channel == nil {
		return nil
	}

	return &DiscordChannel{
		ID:        channel.ID,
		Type:      channel.Type,
		Name:      channel.Name,
		GuildID:   channel.GuildID,
		CreatedAt: channel.CreatedAt,
	}
}
