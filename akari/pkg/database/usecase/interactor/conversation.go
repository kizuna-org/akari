package interactor

import (
	"context"

	"github.com/kizuna-org/akari/pkg/database/domain"
)

type ConversationInteractor interface {
	CreateConversation(ctx context.Context, messageID string, conversationGroupID *int) (*domain.Conversation, error)
	GetConversationByID(ctx context.Context, id int) (*domain.Conversation, error)
	ListConversations(ctx context.Context) ([]*domain.Conversation, error)
	DeleteConversation(ctx context.Context, id int) error
}

type conversationInteractorImpl struct {
	repository domain.ConversationRepository
}

func NewConversationInteractor(repository domain.ConversationRepository) ConversationInteractor {
	return &conversationInteractorImpl{
		repository: repository,
	}
}

func (c *conversationInteractorImpl) CreateConversation(
	ctx context.Context,
	messageID string,
	conversationGroupID *int,
) (*domain.Conversation, error) {
	return c.repository.CreateConversation(ctx, messageID, conversationGroupID)
}

func (c *conversationInteractorImpl) GetConversationByID(ctx context.Context, id int) (*domain.Conversation, error) {
	return c.repository.GetConversationByID(ctx, id)
}

func (c *conversationInteractorImpl) ListConversations(ctx context.Context) ([]*domain.Conversation, error) {
	return c.repository.ListConversations(ctx)
}

func (c *conversationInteractorImpl) DeleteConversation(ctx context.Context, id int) error {
	return c.repository.DeleteConversation(ctx, id)
}
