package adapter

import (
	"context"
	"fmt"
	"time"

	"github.com/kizuna-org/akari/gen/ent"
	"github.com/kizuna-org/akari/internal/message/domain"
	databaseDomain "github.com/kizuna-org/akari/pkg/database/domain"
)

type discordUserRepository struct {
	repository          databaseDomain.DiscordUserRepository
	akariUserRepository databaseDomain.AkariUserRepository
}

func NewDiscordUserRepository(
	repository databaseDomain.DiscordUserRepository,
	akariUserRepository databaseDomain.AkariUserRepository,
) domain.DiscordUserRepository {
	return &discordUserRepository{
		repository:          repository,
		akariUserRepository: akariUserRepository,
	}
}

func (r *discordUserRepository) GetDiscordUserByID(
	ctx context.Context,
	discordUserID string,
) (int, error) {
	user, err := r.repository.GetDiscordUserByID(ctx, discordUserID)
	if err != nil {
		return 0, fmt.Errorf(
			"failed to get discord user by id: %w",
			err,
		)
	}

	akariUser, err := r.akariUserRepository.GetAkariUserByDiscordUserID(ctx, user.ID)
	if err != nil {
		if ent.IsNotFound(err) {
			akariUser, err = r.akariUserRepository.CreateAkariUser(ctx)
			if err != nil {
				return 0, fmt.Errorf("failed to create akari user: %w", err)
			}

			return akariUser.ID, nil
		}

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
	discordUser, err := r.repository.GetDiscordUserByID(ctx, discordUserID)
	if err == nil {
		akariUser, err := r.akariUserRepository.GetAkariUserByDiscordUserID(
			ctx,
			discordUser.ID,
		)
		if err == nil {
			return akariUser.ID, nil
		}

		if !ent.IsNotFound(err) {
			return 0, fmt.Errorf(
				"failed to get akari user by discord user id: %w",
				err,
			)
		}

		akariUser, err = r.akariUserRepository.CreateAkariUser(ctx)
		if err != nil {
			return 0, fmt.Errorf("failed to create akari user: %w", err)
		}

		return akariUser.ID, nil
	}

	if !ent.IsNotFound(err) {
		return 0, fmt.Errorf("failed to get discord user by id: %w", err)
	}

	now := time.Now()
	if _, err := r.repository.CreateDiscordUser(ctx, databaseDomain.DiscordUser{
		ID:        discordUserID,
		Username:  username,
		Bot:       isBot,
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		return 0, fmt.Errorf("failed to create discord user: %w", err)
	}

	akariUser, err := r.akariUserRepository.CreateAkariUser(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to create akari user: %w", err)
	}

	return akariUser.ID, nil
}
