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
	ID         string
	Username   string
	Bot        bool
	AkariUserID *int
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func FromEntDiscordUser(entDiscordUser *ent.DiscordUser) *DiscordUser {
	if entDiscordUser == nil {
		return nil
	}

	return &DiscordUser{
		ID:        entDiscordUser.ID,
		Username:  entDiscordUser.Username,
		Bot:       entDiscordUser.Bot,
		CreatedAt: entDiscordUser.CreatedAt,
		UpdatedAt: entDiscordUser.UpdatedAt,
	}
}
