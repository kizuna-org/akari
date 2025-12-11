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
	params domain.Conversation,
) (*domain.Conversation, error) {
	builder := r.client.ConversationClient().Create().
		SetUserID(params.UserID).
		SetDiscordMessageID(params.DiscordMessageID).
		SetConversationGroupID(params.ConversationGroupID)

	conversation, err := builder.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create conversation: %w", err)
	}

	r.logger.Info("Conversation created",
		slog.Int("id", conversation.ID),
	)

	return domain.FromEntConversation(conversation)
}

func (r *repositoryImpl) GetConversationByID(ctx context.Context, id int) (*domain.Conversation, error) {
	conversation, err := r.client.ConversationClient().
		Query().
		Where(conversation.IDEQ(id)).
		WithUser().
		WithDiscordMessage().
		WithConversationGroup().
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation by id: %w", err)
	}

	return domain.FromEntConversation(conversation)
}

func (r *repositoryImpl) ListConversations(ctx context.Context) ([]*domain.Conversation, error) {
	conversations, err := r.client.ConversationClient().
		Query().
		Order(conversation.ByID()).
		WithUser().
		WithDiscordMessage().
		WithConversationGroup().
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list conversations: %w", err)
	}

	domainConversations := make([]*domain.Conversation, len(conversations))

	for i, conversation := range conversations {
		var err error

		domainConversations[i], err = domain.FromEntConversation(conversation)
		if err != nil {
			return nil, fmt.Errorf("failed to convert conversation: %w", err)
		}
	}

	return domainConversations, nil
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
