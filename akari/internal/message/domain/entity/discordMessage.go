package entity

import (
	"time"

	databaseDomain "github.com/kizuna-org/akari/pkg/database/domain"
	discordEntity "github.com/kizuna-org/akari/pkg/discord/domain/entity"
)

type DiscordMessage struct {
	ID        string
	ChannelID string
	GuildID   string
	AuthorID  string
	Content   string
	Timestamp time.Time
	IsBot     bool
	Mentions  []string
}

func (m *DiscordMessage) ToDatabaseDiscordMessage() databaseDomain.DiscordMessage {
	return databaseDomain.DiscordMessage{
		ID:        m.ID,
		ChannelID: m.ChannelID,
		AuthorID:  m.AuthorID,
		Content:   m.Content,
		Timestamp: m.Timestamp,
		CreatedAt: time.Now(),
	}
}

func ToDiscordMessage(msg *discordEntity.Message) *DiscordMessage {
	if msg == nil {
		return nil
	}

	return &DiscordMessage{
		ID:        msg.ID,
		ChannelID: msg.ChannelID,
		GuildID:   msg.GuildID,
		AuthorID:  msg.AuthorID,
		Content:   msg.Content,
		Timestamp: msg.Timestamp,
		IsBot:     msg.IsBot,
		Mentions:  msg.Mentions,
	}
}
