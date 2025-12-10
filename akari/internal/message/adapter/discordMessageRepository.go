package adapter

import (
	"context"
	"errors"

	"github.com/kizuna-org/akari/internal/message/domain"
	"github.com/kizuna-org/akari/internal/message/domain/entity"
	databaseInteractor "github.com/kizuna-org/akari/pkg/database/usecase/interactor"
)

type discordMessageRepository struct {
	discordMessageInteractor databaseInteractor.DiscordMessageInteractor
}

func NewDiscordMessageRepository(
	discordMessageInteractor databaseInteractor.DiscordMessageInteractor,
) domain.DiscordMessageRepository {
	return &discordMessageRepository{
		discordMessageInteractor: discordMessageInteractor,
	}
}

func (r *discordMessageRepository) SaveMessage(ctx context.Context, message *entity.Message) error {
	if message == nil {
		return errors.New("adapter: message is nil")
	}

	if _, err := r.discordMessageInteractor.CreateDiscordMessage(ctx, message.ToDiscordMessage()); err != nil {
		return err
	}

	return nil
}
