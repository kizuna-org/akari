//nolint:dupl
package interactor

import (
	"context"

	"github.com/kizuna-org/akari/pkg/database/domain"
)

type DiscordGuildInteractor interface {
	CreateDiscordGuild(
		ctx context.Context,
		params domain.DiscordGuild,
	) (*domain.DiscordGuild, error)
	GetDiscordGuildByID(
		ctx context.Context,
		guildID string,
	) (*domain.DiscordGuild, error)
	ListDiscordGuilds(ctx context.Context) ([]*domain.DiscordGuild, error)
	DeleteDiscordGuild(ctx context.Context, id string) error
}

type discordGuildInteractorImpl struct {
	repository domain.DiscordGuildRepository
}

func NewDiscordGuildInteractor(repository domain.DiscordGuildRepository) DiscordGuildInteractor {
	return &discordGuildInteractorImpl{
		repository: repository,
	}
}

func (d *discordGuildInteractorImpl) CreateDiscordGuild(
	ctx context.Context,
	params domain.DiscordGuild,
) (*domain.DiscordGuild, error) {
	return d.repository.CreateDiscordGuild(ctx, params)
}

func (d *discordGuildInteractorImpl) GetDiscordGuildByID(
	ctx context.Context,
	guildID string,
) (*domain.DiscordGuild, error) {
	return d.repository.GetDiscordGuildByID(ctx, guildID)
}

func (d *discordGuildInteractorImpl) ListDiscordGuilds(ctx context.Context) ([]*domain.DiscordGuild, error) {
	return d.repository.ListDiscordGuilds(ctx)
}

func (d *discordGuildInteractorImpl) DeleteDiscordGuild(ctx context.Context, id string) error {
	return d.repository.DeleteDiscordGuild(ctx, id)
}
