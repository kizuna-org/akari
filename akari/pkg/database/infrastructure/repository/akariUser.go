package repository

import (
	"context"
	"fmt"

	"github.com/kizuna-org/akari/pkg/database/domain"
)

func (r *repositoryImpl) CreateAkariUser(ctx context.Context, name string) (*domain.AkariUser, error) {
	user, err := r.client.AkariUserClient().
		Create().
		SetName(name).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create akari user: %w", err)
	}

	return user, nil
}

func (r *repositoryImpl) GetAkariUserByID(ctx context.Context, id int) (*domain.AkariUser, error) {
	user, err := r.client.AkariUserClient().Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get akari user: %w", err)
	}

	return user, nil
}

func (r *repositoryImpl) ListAkariUsers(ctx context.Context) ([]*domain.AkariUser, error) {
	users, err := r.client.AkariUserClient().
		Query().
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list akari users: %w", err)
	}

	return users, nil
}

func (r *repositoryImpl) UpdateAkariUser(ctx context.Context, id int, name string) (*domain.AkariUser, error) {
	user, err := r.client.AkariUserClient().
		UpdateOneID(id).
		SetName(name).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to update akari user: %w", err)
	}

	return user, nil
}

func (r *repositoryImpl) DeleteAkariUser(ctx context.Context, id int) error {
	if err := r.client.AkariUserClient().
		DeleteOneID(id).
		Exec(ctx); err != nil {
		return fmt.Errorf("failed to delete akari user: %w", err)
	}

	return nil
}
