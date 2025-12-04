package adapter

import (
	"context"
	"fmt"

	"github.com/kizuna-org/akari/gen/ent"
	"github.com/kizuna-org/akari/internal/message/domain"
	databaseDomain "github.com/kizuna-org/akari/pkg/database/domain"
)

type akariUserRepository struct {
	repository databaseDomain.AkariUserRepository
}

func NewAkariUserRepository(
	repository databaseDomain.AkariUserRepository,
) domain.AkariUserRepository {
	return &akariUserRepository{
		repository: repository,
	}
}

func (r *akariUserRepository) GetOrCreateAkariUserByDiscordUserID(
	ctx context.Context,
	discordUserID string,
) (int, error) {
	akariUser, err := r.repository.GetAkariUserByDiscordUserID(ctx, discordUserID)
	if err == nil {
		return akariUser.ID, nil
	}

	if !ent.IsNotFound(err) {
		return 0, fmt.Errorf("adapter: failed to get akari user by discord user id: %w", err)
	}

	akariUser, err = r.repository.CreateAkariUser(ctx)
	if err != nil {
		return 0, fmt.Errorf("adapter: failed to create akari user: %w", err)
	}

	return akariUser.ID, nil
}
