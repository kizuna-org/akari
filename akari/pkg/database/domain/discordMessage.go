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
	ChannelID string
	AuthorID  string
	Content   string
	Timestamp time.Time
	CreatedAt time.Time
}

func ToDomainDiscordMessageFromDB(model *ent.DiscordMessage) *DiscordMessage {
	return &DiscordMessage{
		ID:        model.ID,
		ChannelID: model.Edges.Channel.ID,
		AuthorID:  model.AuthorID,
		Content:   model.Content,
		Timestamp: model.Timestamp,
		CreatedAt: model.CreatedAt,
	}
}
