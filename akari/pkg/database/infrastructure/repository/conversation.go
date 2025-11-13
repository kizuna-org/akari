package repository

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/kizuna-org/akari/gen/ent/conversation"
	"github.com/kizuna-org/akari/pkg/database/domain"
)

func (r *repositoryImpl) CreateConversation(
	ctx context.Context,
	triggerMessageID, responseMessageID string,
	conversationGroupID *int,
) (*domain.Conversation, error) {
	builder := r.client.ConversationClient().Create().
		SetTriggerMessageID(triggerMessageID).
		SetResponseMessageID(responseMessageID)

	if conversationGroupID != nil {
		builder = builder.SetNillableConversationGroupID(conversationGroupID)
	}

	conv, err := builder.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create conversation: %w", err)
	}

	r.logger.Info("Conversation created",
		slog.Int("id", conv.ID),
	)

	return conv, nil
}

func (r *repositoryImpl) GetConversationByID(ctx context.Context, id int) (*domain.Conversation, error) {
	conv, err := r.client.ConversationClient().
		Query().
		Where(conversation.IDEQ(id)).
		WithTriggerMessage().
		WithResponseMessage().
		WithConversationGroup().
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation by id: %w", err)
	}

	return conv, nil
}

func (r *repositoryImpl) ListConversations(ctx context.Context) ([]*domain.Conversation, error) {
	convs, err := r.client.ConversationClient().
		Query().
		Order(conversation.ByID()).
		WithTriggerMessage().
		WithResponseMessage().
		WithConversationGroup().
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list conversations: %w", err)
	}

	return convs, nil
}

func (r *repositoryImpl) DeleteConversation(ctx context.Context, conversationID int) error {
	if err := r.client.ConversationClient().DeleteOneID(conversationID).Exec(ctx); err != nil {
		return fmt.Errorf("failed to delete conversation: %w", err)
	}

	r.logger.Info("Conversation deleted",
		slog.Int("id", conversationID),
	)

	return nil
}
