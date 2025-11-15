package domain

//go:generate go tool mockgen -package=mock -source=akariUser.go -destination=mock/akariUser.go

import (
	"context"

	"github.com/kizuna-org/akari/gen/ent"
)

type AkariUserRepository interface {
	CreateAkariUser(ctx context.Context, name string) (*AkariUser, error)
	GetAkariUserByID(ctx context.Context, id int) (*AkariUser, error)
	ListAkariUsers(ctx context.Context) ([]*AkariUser, error)
	UpdateAkariUser(ctx context.Context, id int, name string) (*AkariUser, error)
	DeleteAkariUser(ctx context.Context, id int) error
}

type AkariUser = ent.AkariUser
