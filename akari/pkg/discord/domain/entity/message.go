package entity

import "time"

// Message represents a Discord message entity
type Message struct {
	ID        string
	ChannelID string
	GuildID   string
	AuthorID  string
	Content   string
	Timestamp time.Time
}
