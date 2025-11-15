package domain

//go:generate go tool mockgen -package=mock -source=discordUser.go -destination=mock/discordUser.go

import (
	"context"
	"time"

	"github.com/kizuna-org/akari/gen/ent"
)

type DiscordUserRepository interface {
	CreateDiscordUser(ctx context.Context, params DiscordUser) (*DiscordUser, error)
	GetDiscordUserByID(ctx context.Context, id string) (*DiscordUser, error)
	ListDiscordUsers(ctx context.Context) ([]*DiscordUser, error)
	DeleteDiscordUser(ctx context.Context, id string) error
}

type DiscordUser struct {
	ID        string
	Username  string
	Bot       bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

func ToDomainDiscordUserFromDB(model *ent.DiscordUser) *DiscordUser {
	return &DiscordUser{
		ID:        model.ID,
		Username:  model.Username,
		Bot:       model.Bot,
		CreatedAt: model.CreatedAt,
		UpdatedAt: model.UpdatedAt,
	}
}
