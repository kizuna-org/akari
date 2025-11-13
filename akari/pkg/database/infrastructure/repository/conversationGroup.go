package repository

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/kizuna-org/akari/pkg/database/domain"
)

func (r *repositoryImpl) CreateConversationGroup(ctx context.Context) (*domain.ConversationGroup, error) {
	conversationGroup, err := r.client.ConversationGroupClient().Create().Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create conversation group: %w", err)
	}

	r.logger.Info("conversation group created",
		slog.Int("id", conversationGroup.ID),
	)

	return conversationGroup, nil
}

func (r *repositoryImpl) GetConversationGroupByID(ctx context.Context, id int) (*domain.ConversationGroup, error) {
	cg, err := r.client.ConversationGroupClient().Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation group by id: %w", err)
	}

	return cg, nil
}

func (r *repositoryImpl) ListConversationGroups(ctx context.Context) ([]*domain.ConversationGroup, error) {
	cgs, err := r.client.ConversationGroupClient().Query().All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list conversation groups: %w", err)
	}

	return cgs, nil
}

func (r *repositoryImpl) DeleteConversationGroup(ctx context.Context, conversationGroupID int) error {
	if err := r.client.ConversationGroupClient().DeleteOneID(conversationGroupID).Exec(ctx); err != nil {
		return fmt.Errorf("failed to delete conversation group: %w", err)
	}

	r.logger.Info("conversation group deleted",
		slog.Int("id", conversationGroupID),
	)

	return nil
}
