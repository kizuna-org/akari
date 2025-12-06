package adapter

import (
	"context"
	"fmt"

	"github.com/kizuna-org/akari/internal/message/domain"
	databaseDomain "github.com/kizuna-org/akari/pkg/database/domain"
)

type messageRepository struct {
	discordMsgRepo databaseDomain.DiscordMessageRepository
}

func NewMessageRepository(discordMsgRepo databaseDomain.DiscordMessageRepository) domain.MessageRepository {
	return &messageRepository{
		discordMsgRepo: discordMsgRepo,
	}
}

func (r *messageRepository) SaveMessage(ctx context.Context, message *domain.Message) error {
	if _, err := r.discordMsgRepo.CreateDiscordMessage(ctx, databaseDomain.DiscordMessage{
		ID:        message.ID,
		ChannelID: message.ChannelID,
		AuthorID:  message.AuthorID,
		Content:   message.Content,
		Timestamp: message.Timestamp,
		CreatedAt: message.Timestamp,
	}); err != nil {
		return fmt.Errorf("failed to save discord message: %w", err)
	}

	return nil
}
