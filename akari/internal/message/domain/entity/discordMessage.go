package entity

import (
	"time"

	databaseDomain "github.com/kizuna-org/akari/pkg/database/domain"
)

type DiscordMessage struct {
	ID        string
	ChannelID string
	AuthorID  string
	Content   string
	Timestamp time.Time
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

func ToDiscordMessage(msg *databaseDomain.DiscordMessage) *DiscordMessage {
	if msg == nil {
		return nil
	}

	return &DiscordMessage{
		ID:        msg.ID,
		ChannelID: msg.ChannelID,
		AuthorID:  msg.AuthorID,
		Content:   msg.Content,
		Timestamp: msg.Timestamp,
	}
}
