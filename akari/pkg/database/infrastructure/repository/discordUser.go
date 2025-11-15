package repository

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/kizuna-org/akari/gen/ent/discorduser"
	"github.com/kizuna-org/akari/pkg/database/domain"
)

func (r *repositoryImpl) CreateDiscordUser(
	ctx context.Context,
	params domain.DiscordUser,
) (*domain.DiscordUser, error) {
	user, err := r.client.DiscordUserClient().Create().
		SetID(params.ID).
		SetUsername(params.Username).
		SetBot(params.Bot).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create discord user: %w", err)
	}

	r.logger.Info("Discord user created",
		slog.String("user_id", user.ID),
		slog.String("username", user.Username),
	)

	return domain.ToDomainDiscordUserFromDB(user), nil
}

func (r *repositoryImpl) GetDiscordUserByID(
	ctx context.Context,
	id string,
) (*domain.DiscordUser, error) {
	user, err := r.client.DiscordUserClient().
		Query().
		Where(discorduser.IDEQ(id)).
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get discord user by id: %w", err)
	}

	return domain.ToDomainDiscordUserFromDB(user), nil
}

func (r *repositoryImpl) ListDiscordUsers(ctx context.Context) ([]*domain.DiscordUser, error) {
	users, err := r.client.DiscordUserClient().Query().All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list discord users: %w", err)
	}

	domainUsers := make([]*domain.DiscordUser, 0, len(users))
	for _, user := range users {
		domainUsers = append(domainUsers, domain.ToDomainDiscordUserFromDB(user))
	}

	return domainUsers, nil
}

func (r *repositoryImpl) DeleteDiscordUser(ctx context.Context, userID string) error {
	if err := r.client.DiscordUserClient().DeleteOneID(userID).Exec(ctx); err != nil {
		return fmt.Errorf("failed to delete discord user: %w", err)
	}

	r.logger.Info("Discord user deleted",
		slog.String("id", userID),
	)

	return nil
}
