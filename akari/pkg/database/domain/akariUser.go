package domain

//go:generate go tool mockgen -package=mock -source=akariUser.go -destination=mock/akariUser.go

import (
	"context"
	"errors"
	"time"

	"github.com/kizuna-org/akari/gen/ent"
)

type AkariUserRepository interface {
	CreateAkariUser(ctx context.Context) (*AkariUser, error)
	GetAkariUserByID(ctx context.Context, id int) (*AkariUser, error)
	GetAkariUserByDiscordUserID(ctx context.Context, discordUserID string) (*AkariUser, error)
	ListAkariUsers(ctx context.Context) ([]*AkariUser, error)
	DeleteAkariUser(ctx context.Context, id int) error
}

type AkariUser struct {
	ID        int
	CreatedAt time.Time
	UpdatedAt time.Time
}

func FromEntAkariUser(entAkariUser *ent.AkariUser) (*AkariUser, error) {
	if entAkariUser == nil {
		return nil, errors.New("akariUser is nil")
	}

	return &AkariUser{
		ID:        entAkariUser.ID,
		CreatedAt: entAkariUser.CreatedAt,
		UpdatedAt: entAkariUser.UpdatedAt,
	}, nil
}
