package entity

import (
	"time"

	databaseDomain "github.com/kizuna-org/akari/pkg/database/domain"
	discordEntity "github.com/kizuna-org/akari/pkg/discord/domain/entity"
)

type Guild struct {
	ID        string
	Name      string
	CreatedAt time.Time
}

func (g *Guild) ToDatabaseGuild() databaseDomain.DiscordGuild {
	return databaseDomain.DiscordGuild{
		ID:         g.ID,
		Name:       g.Name,
		ChannelIDs: []string{},
		CreatedAt:  g.CreatedAt,
	}
}

func ToGuild(guild *discordEntity.Guild) *Guild {
	if guild == nil {
		return nil
	}

	return &Guild{
		ID:        guild.ID,
		Name:      guild.Name,
		CreatedAt: guild.CreatedAt,
	}
}
