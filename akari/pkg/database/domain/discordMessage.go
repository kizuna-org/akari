package domain

//go:generate go tool mockgen -package=mock -source=discordMessage.go -destination=mock/discordMessage.go

import (
	"context"
	"errors"
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
	ChannelID string
	AuthorID  string
	Content   string
	Timestamp time.Time
	CreatedAt time.Time
}

func FromEntDiscordMessage(discordMessage *ent.DiscordMessage) (*DiscordMessage, error) {
	if discordMessage == nil {
		return nil, errors.New("discordMessage is nil")
	}

	if discordMessage.Edges.Channel == nil {
		return nil, errors.New("discordMessage.Channel edge is nil")
	}

	channelID := discordMessage.Edges.Channel.ID

	if discordMessage.Edges.Author == nil {
		return nil, errors.New("discordMessage.Author edge is nil")
	}

	authorID := discordMessage.Edges.Author.ID

	return &DiscordMessage{
		ID:        discordMessage.ID,
		ChannelID: channelID,
		AuthorID:  authorID,
		Content:   discordMessage.Content,
		Timestamp: discordMessage.Timestamp,
		CreatedAt: discordMessage.CreatedAt,
	}, nil
}
