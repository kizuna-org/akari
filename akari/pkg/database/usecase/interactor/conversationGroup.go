package interactor

import (
	"context"

	"github.com/kizuna-org/akari/pkg/database/domain"
)

type ConversationGroupInteractor interface {
	CreateConversationGroup(ctx context.Context, characterID int) (*domain.ConversationGroup, error)
	GetConversationGroupByID(ctx context.Context, id int) (*domain.ConversationGroup, error)
	GetConversationGroupByCharacterID(ctx context.Context, characterID int) (*domain.ConversationGroup, error)
	ListConversationGroups(ctx context.Context) ([]*domain.ConversationGroup, error)
	DeleteConversationGroup(ctx context.Context, id int) error
}

type conversationGroupInteractorImpl struct {
	repository domain.ConversationGroupRepository
}

func NewConversationGroupInteractor(repository domain.ConversationGroupRepository) ConversationGroupInteractor {
	return &conversationGroupInteractorImpl{
		repository: repository,
	}
}

func (c *conversationGroupInteractorImpl) CreateConversationGroup(
	ctx context.Context,
	characterID int,
) (*domain.ConversationGroup, error) {
	return c.repository.CreateConversationGroup(ctx, characterID)
}

func (c *conversationGroupInteractorImpl) GetConversationGroupByID(
	ctx context.Context,
	id int,
) (*domain.ConversationGroup, error) {
	return c.repository.GetConversationGroupByID(ctx, id)
}

func (c *conversationGroupInteractorImpl) GetConversationGroupByCharacterID(
	ctx context.Context,
	characterID int,
) (*domain.ConversationGroup, error) {
	return c.repository.GetConversationGroupByCharacterID(ctx, characterID)
}

func (c *conversationGroupInteractorImpl) ListConversationGroups(
	ctx context.Context,
) ([]*domain.ConversationGroup, error) {
	return c.repository.ListConversationGroups(ctx)
}

func (c *conversationGroupInteractorImpl) DeleteConversationGroup(ctx context.Context, id int) error {
	return c.repository.DeleteConversationGroup(ctx, id)
}
