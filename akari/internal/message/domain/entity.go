package domain

import (
	"time"

	discordEntity "github.com/kizuna-org/akari/pkg/discord/domain/entity"
)

type Message struct {
	ID        string
	ChannelID string
	GuildID   string
	AuthorID  string
	Content   string
	Timestamp time.Time
	IsBot     bool
	Mentions  []string
}

func ToMessage(msg *discordEntity.Message) *Message {
	return &Message{
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
