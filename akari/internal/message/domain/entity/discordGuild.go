package entity

import (
	"time"

	databaseDomain "github.com/kizuna-org/akari/pkg/database/domain"
	discordEntity "github.com/kizuna-org/akari/pkg/discord/domain/entity"
)

type DiscordGuild struct {
	ID        string
	Name      string
	CreatedAt time.Time
}

func (g *DiscordGuild) ToDatabaseDiscordGuild() databaseDomain.DiscordGuild {
	return databaseDomain.DiscordGuild{
		ID:         g.ID,
		Name:       g.Name,
		ChannelIDs: []string{},
		CreatedAt:  g.CreatedAt,
	}
}

func ToDiscordGuild(guild *discordEntity.Guild) *DiscordGuild {
	if guild == nil {
		return nil
	}

	return &DiscordGuild{
		ID:        guild.ID,
		Name:      guild.Name,
		CreatedAt: guild.CreatedAt,
	}
}
