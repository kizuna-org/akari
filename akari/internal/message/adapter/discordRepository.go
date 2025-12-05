package adapter

import (
	"context"
	"fmt"

	"github.com/kizuna-org/akari/internal/message/domain"
	discordInteractor "github.com/kizuna-org/akari/pkg/discord/usecase/interactor"
)

type discordRepository struct {
	discordInteractor discordInteractor.DiscordInteractor
}

func NewDiscordRepository(discordInteractor discordInteractor.DiscordInteractor) domain.DiscordRepository {
	return &discordRepository{
		discordInteractor: discordInteractor,
	}
}

func (r *discordRepository) SendMessage(ctx context.Context, channelID string, content string) error {
	if _, err := r.discordInteractor.SendMessage(ctx, channelID, content); err != nil {
		return fmt.Errorf("failed to send discord message: %w", err)
	}

	return nil
}
