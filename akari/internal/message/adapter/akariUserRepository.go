package adapter

import (
	"context"
	"fmt"

	"github.com/kizuna-org/akari/gen/ent"
	"github.com/kizuna-org/akari/internal/message/domain"
	databaseInteractor "github.com/kizuna-org/akari/pkg/database/usecase/interactor"
)

type akariUserRepository struct {
	interactor databaseInteractor.AkariUserInteractor
}

func NewAkariUserRepository(
	interactor databaseInteractor.AkariUserInteractor,
) domain.AkariUserRepository {
	return &akariUserRepository{
		interactor: interactor,
	}
}

func (r *akariUserRepository) GetOrCreateAkariUserByDiscordUserID(
	ctx context.Context,
	discordUserID string,
) (int, error) {
	akariUser, err := r.interactor.GetAkariUserByDiscordUserID(ctx, discordUserID)
	if err == nil {
		return akariUser.ID, nil
	}

	if !ent.IsNotFound(err) {
		return 0, fmt.Errorf("adapter: failed to get akari user by discord user id: %w", err)
	}

	akariUser, err = r.interactor.CreateAkariUser(ctx)
	if err != nil {
		return 0, fmt.Errorf("adapter: failed to create akari user: %w", err)
	}

	return akariUser.ID, nil
}
