package entity

import (
	"strconv"
	"time"

	databaseDomain "github.com/kizuna-org/akari/pkg/database/domain"
	discordEntity "github.com/kizuna-org/akari/pkg/discord/domain/entity"
)

type Channel struct {
	ID        string
	Type      int
	Name      string
	GuildID   string
	CreatedAt time.Time
}

func (c *Channel) ToDiscordChannel() databaseDomain.DiscordChannel {
	return databaseDomain.DiscordChannel{
		ID:        c.ID,
		Type:      databaseDomain.DiscordChannelType(strconv.Itoa(c.Type)),
		Name:      c.Name,
		GuildID:   c.GuildID,
		CreatedAt: c.CreatedAt,
	}
}

func ToChannel(channel *discordEntity.Channel) *Channel {
	if channel == nil {
		return nil
	}

	return &Channel{
		ID:        channel.ID,
		Type:      channel.Type,
		Name:      channel.Name,
		GuildID:   channel.GuildID,
		CreatedAt: channel.CreatedAt,
	}
}
