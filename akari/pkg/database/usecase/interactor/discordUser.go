package interactor

import (
	"context"
	"errors"

	"github.com/kizuna-org/akari/pkg/database/domain"
)

type DiscordUserInteractor interface {
	CreateDiscordUser(ctx context.Context, params domain.DiscordUser) (*domain.DiscordUser, error)
	GetDiscordUserByID(ctx context.Context, userID string) (*domain.DiscordUser, error)
	ListDiscordUsers(ctx context.Context) ([]*domain.DiscordUser, error)
	DeleteDiscordUser(ctx context.Context, userID string) error
}

type discordUserInteractorImpl struct {
	repository domain.DiscordUserRepository
}

func NewDiscordUserInteractor(repository domain.DiscordUserRepository) DiscordUserInteractor {
	return &discordUserInteractorImpl{repository: repository}
}

func (d *discordUserInteractorImpl) CreateDiscordUser(
	ctx context.Context,
	params domain.DiscordUser,
) (*domain.DiscordUser, error) {
	if params.ID == "" {
		return nil, errors.New("discord user id is required")
	}

	return d.repository.CreateDiscordUser(ctx, params)
}

func (d *discordUserInteractorImpl) GetDiscordUserByID(
	ctx context.Context,
	userID string,
) (*domain.DiscordUser, error) {
	if userID == "" {
		return nil, errors.New("id is required")
	}

	return d.repository.GetDiscordUserByID(ctx, userID)
}

func (d *discordUserInteractorImpl) ListDiscordUsers(ctx context.Context) ([]*domain.DiscordUser, error) {
	return d.repository.ListDiscordUsers(ctx)
}

func (d *discordUserInteractorImpl) DeleteDiscordUser(ctx context.Context, userID string) error {
	if userID == "" {
		return errors.New("userID is required")
	}

	return d.repository.DeleteDiscordUser(ctx, userID)
}
