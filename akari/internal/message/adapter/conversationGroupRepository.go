package adapter

import (
	"context"
	"errors"
	"fmt"

	"github.com/kizuna-org/akari/internal/message/domain"
	dbdomain "github.com/kizuna-org/akari/pkg/database/domain"
	databaseInteractor "github.com/kizuna-org/akari/pkg/database/usecase/interactor"
)

type conversationGroupRepository struct {
	interactor databaseInteractor.ConversationGroupInteractor
}

func NewConversationGroupRepository(
	interactor databaseInteractor.ConversationGroupInteractor,
) domain.ConversationGroupRepository {
	return &conversationGroupRepository{
		interactor: interactor,
	}
}

func (r *conversationGroupRepository) GetConversationGroupByCharacterID(
	ctx context.Context,
	characterID int,
) (*domain.ConversationGroup, error) {
	conversationGroup, err := r.interactor.GetConversationGroupByCharacterID(
		ctx,
		characterID,
	)
	if err != nil {
		if errors.Is(err, dbdomain.ErrNotFound) {
			return nil, dbdomain.ErrNotFound
		}

		return nil, fmt.Errorf(
			"adapter: failed to get conversation group by character id: %w",
			err,
		)
	}

	return &domain.ConversationGroup{
		ID:          conversationGroup.ID,
		CharacterID: conversationGroup.CharacterID,
	}, nil
}

func (r *conversationGroupRepository) CreateConversationGroup(
	ctx context.Context,
	characterID int,
) (*domain.ConversationGroup, error) {
	conversationGroup, err := r.interactor.CreateConversationGroup(ctx, characterID)
	if err != nil {
		return nil, fmt.Errorf("failed to create conversation group: %w", err)
	}

	return &domain.ConversationGroup{
		ID:          conversationGroup.ID,
		CharacterID: conversationGroup.CharacterID,
	}, nil
}
