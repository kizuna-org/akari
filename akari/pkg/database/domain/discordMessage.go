package domain

//go:generate go tool mockgen -package=mock -source=discordMessage.go -destination=mock/discordMessage.go

import (
	"context"
	"time"

	"github.com/kizuna-org/akari/gen/ent"
)

type DiscordMessageRepository interface {
	CreateDiscordMessage(ctx context.Context, params DiscordMessage) (*DiscordMessage, error)
	GetDiscordMessageByID(ctx context.Context, id string) (*DiscordMessage, error)
	DeleteDiscordMessage(ctx context.Context, id string) error
}

type DiscordMessage struct {
	ID        string
	Channel   *DiscordChannel
	Author    *DiscordUser
	Content   string
	Timestamp time.Time
	CreatedAt time.Time
}

func FromEntDiscordMessage(discordMessage *ent.DiscordMessage) *DiscordMessage {
	if discordMessage == nil {
		return nil
	}

	var discordChannel *DiscordChannel
	if discordMessage.Edges.Channel != nil {
		discordChannel = FromEntDiscordChannel(discordMessage.Edges.Channel)
	}

	var discordAuthor *DiscordUser
	if discordMessage.Edges.Author != nil {
		discordAuthor = FromEntDiscordUser(discordMessage.Edges.Author)
	}

	return &DiscordMessage{
		ID:        discordMessage.ID,
		Channel:   discordChannel,
		Author:    discordAuthor,
		Content:   discordMessage.Content,
		Timestamp: discordMessage.Timestamp,
		CreatedAt: discordMessage.CreatedAt,
	}
}
