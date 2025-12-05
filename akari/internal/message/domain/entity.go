package domain

import "time"

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
