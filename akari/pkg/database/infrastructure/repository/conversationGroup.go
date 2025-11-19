package repository

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/kizuna-org/akari/gen/ent/character"
	"github.com/kizuna-org/akari/gen/ent/conversationgroup"
	"github.com/kizuna-org/akari/pkg/database/domain"
)

func (r *repositoryImpl) CreateConversationGroup(
	ctx context.Context,
	characterID int,
) (*domain.ConversationGroup, error) {
	conversationGroup, err := r.client.ConversationGroupClient().
		Create().
		SetCharacterID(characterID).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create conversation group: %w", err)
	}

	r.logger.Info("Conversation group created",
		slog.Int("id", conversationGroup.ID),
	)

	return domain.FromEntConversationGroup(conversationGroup), nil
}

func (r *repositoryImpl) GetConversationGroupByID(
	ctx context.Context,
	id int,
) (*domain.ConversationGroup, error) {
	conversationGroup, err := r.client.ConversationGroupClient().
		Query().
		Where(conversationgroup.IDEQ(id)).
		WithCharacter().
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation group by id: %w", err)
	}

	return domain.FromEntConversationGroup(conversationGroup), nil
}

func (r *repositoryImpl) GetConversationGroupByCharacterID(
	ctx context.Context,
	characterID int,
) (*domain.ConversationGroup, error) {
	conversationGroup, err := r.client.ConversationGroupClient().
		Query().
		Where(conversationgroup.HasCharacterWith(character.IDEQ(characterID))).
		WithCharacter().
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation group by character id: %w", err)
	}

	return domain.FromEntConversationGroup(conversationGroup), nil
}

func (r *repositoryImpl) ListConversationGroups(ctx context.Context) ([]*domain.ConversationGroup, error) {
	conversationGroups, err := r.client.ConversationGroupClient().
		Query().
		WithCharacter().
		WithConversations().
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list conversation groups: %w", err)
	}

	domainConversationGroups := make([]*domain.ConversationGroup, len(conversationGroups))
	for i, conversationGroup := range conversationGroups {
		domainConversationGroups[i] = domain.FromEntConversationGroup(conversationGroup)
	}

	return domainConversationGroups, nil
}

func (r *repositoryImpl) DeleteConversationGroup(ctx context.Context, conversationGroupID int) error {
	if err := r.client.ConversationGroupClient().DeleteOneID(conversationGroupID).Exec(ctx); err != nil {
		return fmt.Errorf("failed to delete conversation group: %w", err)
	}

	r.logger.Info("Conversation group deleted",
		slog.Int("id", conversationGroupID),
	)

	return nil
}
