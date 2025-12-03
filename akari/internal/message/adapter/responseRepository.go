package adapter

import (
	"context"
	"fmt"

	"github.com/kizuna-org/akari/internal/message/domain"
	databaseDomain "github.com/kizuna-org/akari/pkg/database/domain"
)

type responseRepository struct {
	discordMsgRepo databaseDomain.DiscordMessageRepository
}

func NewResponseRepository(discordMsgRepo databaseDomain.DiscordMessageRepository) domain.ResponseRepository {
	return &responseRepository{
		discordMsgRepo: discordMsgRepo,
	}
}

func (r *responseRepository) SaveResponse(ctx context.Context, response *domain.Response) error {
	if response == nil {
		return fmt.Errorf("response is nil")
	}

	if _, err := r.discordMsgRepo.CreateDiscordMessage(ctx, databaseDomain.DiscordMessage{
		ID:        response.ID,
		ChannelID: response.ChannelID,
		AuthorID:  "",
		Content:   response.Content,
		Timestamp: response.CreatedAt,
		CreatedAt: response.CreatedAt,
	}); err != nil {
		return fmt.Errorf("failed to save response: %w", err)
	}

	return nil
}
