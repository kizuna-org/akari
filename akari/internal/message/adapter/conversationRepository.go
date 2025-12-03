package adapter

import (
	"context"
	"fmt"

	"github.com/kizuna-org/akari/internal/message/domain"
	databaseInteractor "github.com/kizuna-org/akari/pkg/database/usecase/interactor"
)

type conversationRepository struct {
	interactor databaseInteractor.ConversationInteractor
}

func NewConversationRepository(interactor databaseInteractor.ConversationInteractor) domain.ConversationRepository {
	return &conversationRepository{
		interactor: interactor,
	}
}

func (r *conversationRepository) CreateConversation(
	ctx context.Context,
	messageID string,
	conversationGroupID *int,
) error {
	if _, err := r.interactor.CreateConversation(ctx, messageID, conversationGroupID); err != nil {
		return fmt.Errorf("failed to create conversation: %w", err)
	}

	return nil
}
