package interactor

import (
	"context"

	"github.com/kizuna-org/akari/pkg/database/domain"
)

type AkariUserInteractor interface {
	CreateAkariUser(ctx context.Context) (*domain.AkariUser, error)
	GetAkariUserByID(ctx context.Context, id int) (*domain.AkariUser, error)
	GetAkariUserByDiscordUserID(ctx context.Context, discordUserID string) (*domain.AkariUser, error)
	ListAkariUsers(ctx context.Context) ([]*domain.AkariUser, error)
	DeleteAkariUser(ctx context.Context, id int) error
}

type akariUserInteractorImpl struct {
	repository domain.AkariUserRepository
}

func NewAkariUserInteractor(repository domain.AkariUserRepository) AkariUserInteractor {
	return &akariUserInteractorImpl{repository: repository}
}

func (i *akariUserInteractorImpl) CreateAkariUser(ctx context.Context) (*domain.AkariUser, error) {
	return i.repository.CreateAkariUser(ctx)
}

func (i *akariUserInteractorImpl) GetAkariUserByID(ctx context.Context, id int) (*domain.AkariUser, error) {
	return i.repository.GetAkariUserByID(ctx, id)
}

func (i *akariUserInteractorImpl) GetAkariUserByDiscordUserID(
	ctx context.Context,
	discordUserID string,
) (*domain.AkariUser, error) {
	return i.repository.GetAkariUserByDiscordUserID(ctx, discordUserID)
}

func (i *akariUserInteractorImpl) ListAkariUsers(ctx context.Context) ([]*domain.AkariUser, error) {
	return i.repository.ListAkariUsers(ctx)
}

func (i *akariUserInteractorImpl) DeleteAkariUser(ctx context.Context, id int) error {
	return i.repository.DeleteAkariUser(ctx, id)
}
