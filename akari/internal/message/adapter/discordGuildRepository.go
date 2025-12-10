package adapter

import (
	"context"
	"errors"
	"fmt"

	"github.com/kizuna-org/akari/gen/ent"
	"github.com/kizuna-org/akari/internal/message/domain"
	"github.com/kizuna-org/akari/internal/message/domain/entity"
	databaseInteractor "github.com/kizuna-org/akari/pkg/database/usecase/interactor"
)

type discordGuildRepository struct {
	discordGuildInteractor databaseInteractor.DiscordGuildInteractor
}

func NewDiscordGuildRepository(
	discordGuildInteractor databaseInteractor.DiscordGuildInteractor,
) domain.DiscordGuildRepository {
	return &discordGuildRepository{
		discordGuildInteractor: discordGuildInteractor,
	}
}

func (r *discordGuildRepository) CreateIfNotExists(ctx context.Context, guild *entity.Guild) (string, error) {
	if guild == nil {
		return "", errors.New("adapter: guild is required")
	}

	if _, err := r.discordGuildInteractor.GetDiscordGuildByID(ctx, guild.ID); err == nil {
		return guild.ID, nil
	} else if !ent.IsNotFound(err) {
		return "", fmt.Errorf("adapter: failed to get discord guild by id: %w", err)
	}

	discordGuild, err := r.discordGuildInteractor.CreateDiscordGuild(ctx, guild.ToDatabaseGuild())
	if err != nil {
		return "", fmt.Errorf("adapter: failed to create discord guild: %w", err)
	}

	return discordGuild.ID, nil
}
