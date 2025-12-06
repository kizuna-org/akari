package adapter

import (
	"context"
	"fmt"

	"github.com/kizuna-org/akari/internal/message/domain"
	databaseDomain "github.com/kizuna-org/akari/pkg/database/domain"
)

type conversationRepository struct {
	repository databaseDomain.ConversationRepository
}

func NewConversationRepository(repository databaseDomain.ConversationRepository) domain.ConversationRepository {
	return &conversationRepository{
		repository: repository,
	}
}

func (r *conversationRepository) CreateConversation(
	ctx context.Context,
	messageID string,
	userID int,
	conversationGroupID *int,
) error {
	if _, err := r.repository.CreateConversation(
		ctx,
		messageID,
		userID,
		conversationGroupID,
	); err != nil {
		return fmt.Errorf("adapter: failed to create conversation: %w", err)
	}

	return nil
}
