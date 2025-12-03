package adapter

import (
	"context"
	"fmt"

	"github.com/kizuna-org/akari/internal/message/domain"
	databaseDomain "github.com/kizuna-org/akari/pkg/database/domain"
	databaseInteractor "github.com/kizuna-org/akari/pkg/database/usecase/interactor"
)

type discordUserRepository struct {
	interactor          databaseInteractor.DiscordUserInteractor
	akariUserInteractor databaseInteractor.AkariUserInteractor
}

func NewDiscordUserRepository(
	interactor databaseInteractor.DiscordUserInteractor,
	akariUserInteractor databaseInteractor.AkariUserInteractor,
) domain.DiscordUserRepository {
	return &discordUserRepository{
		interactor:          interactor,
		akariUserInteractor: akariUserInteractor,
	}
}

func (r *discordUserRepository) GetDiscordUserByID(
	ctx context.Context,
	discordUserID string,
) (int, error) {
	user, err := r.interactor.GetDiscordUserByID(ctx, discordUserID)
	if err != nil {
		return 0, fmt.Errorf(
			"failed to get discord user by id: %w",
			err,
		)
	}

	akariUser, err := r.akariUserInteractor.GetAkariUserByDiscordUserID(ctx, user.ID)
	if err != nil {
		return 0, fmt.Errorf(
			"failed to get akari user by discord user id: %w",
			err,
		)
	}

	return akariUser.ID, nil
}

func (r *discordUserRepository) GetOrCreateDiscordUser(
	ctx context.Context,
	discordUserID string,
	username string,
	isBot bool,
) (int, error) {
	discordUser, err := r.interactor.GetDiscordUserByID(ctx, discordUserID)
	if err == nil {
		akariUser, err := r.akariUserInteractor.GetAkariUserByDiscordUserID(
			ctx,
			discordUser.ID,
		)
		if err == nil {
			return akariUser.ID, nil
		}
	}

	discordUser, err = r.interactor.CreateDiscordUser(ctx, databaseDomain.DiscordUser{
		ID:       discordUserID,
		Username: username,
		Bot:      isBot,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to create discord user: %w", err)
	}

	akariUser, err := r.akariUserInteractor.CreateAkariUser(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to create akari user: %w", err)
	}

	return akariUser.ID, nil
}
