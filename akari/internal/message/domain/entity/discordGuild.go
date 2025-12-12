package entity

import (
	"time"

	databaseDomain "github.com/kizuna-org/akari/pkg/database/domain"
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

func ToDiscordGuild(guild *databaseDomain.DiscordGuild) *DiscordGuild {
	if guild == nil {
		return nil
	}

	return &DiscordGuild{
		ID:        guild.ID,
		Name:      guild.Name,
		CreatedAt: guild.CreatedAt,
	}
}
