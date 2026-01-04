package adapter

import (
	"context"
	"errors"
	"fmt"

	"github.com/kizuna-org/akari/gen/ent"
	"github.com/kizuna-org/akari/internal/message/domain"
	"github.com/kizuna-org/akari/internal/message/domain/entity"
	databaseInteractor "github.com/kizuna-org/akari/pkg/database/usecase/interactor"
)

type discordUserRepository struct {
	discordUserInteractor databaseInteractor.DiscordUserInteractor
	akariUserInteractor   databaseInteractor.AkariUserInteractor
}

func NewDiscordUserRepository(
	discordUserInteractor databaseInteractor.DiscordUserInteractor,
	akariUserInteractor databaseInteractor.AkariUserInteractor,
) domain.DiscordUserRepository {
	return &discordUserRepository{
		discordUserInteractor: discordUserInteractor,
		akariUserInteractor:   akariUserInteractor,
	}
}

func (r *discordUserRepository) CreateIfNotExists(ctx context.Context, user *entity.DiscordUser) (string, error) {
	if user == nil {
		return "", errors.New("adapter: user is required")
	}

	if _, err := r.discordUserInteractor.GetDiscordUserByID(ctx, user.ID); err == nil {
		return user.ID, nil
	} else if !ent.IsNotFound(err) {
		return "", fmt.Errorf("adapter: failed to get discord user by id: %w", err)
	}

	dbUser := user.ToDatabaseDiscordUser()

	if dbUser.AkariUserID == nil {
		akariUser, err := r.akariUserInteractor.CreateAkariUser(ctx)
		if err != nil {
			return "", fmt.Errorf("adapter: failed to create akari user: %w", err)
		}
		dbUser.AkariUserID = &akariUser.ID
	}

	discordUser, err := r.discordUserInteractor.CreateDiscordUser(ctx, dbUser)
	if err != nil {
		return "", fmt.Errorf("adapter: failed to create discord user: %w", err)
	}

	return discordUser.ID, nil
}
