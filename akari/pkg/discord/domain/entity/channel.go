package entity

import "time"

type Channel struct {
	ID        string
	Type      int
	Name      string
	GuildID   string
	CreatedAt time.Time
}
