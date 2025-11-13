package service

//go:generate go tool mockgen -package=mock -source=discord.go -destination=mock/discord.go

import (
	"context"
	"errors"

	"github.com/kizuna-org/akari/pkg/discord/domain/entity"
	"github.com/kizuna-org/akari/pkg/discord/domain/repository"
)

type DiscordService interface {
	SendMessage(ctx context.Context, channelID string, content string) (*entity.Message, error)
	GetMessage(ctx context.Context, channelID string, messageID string) (*entity.Message, error)
}

type discordServiceImpl struct {
	repo repository.DiscordRepository
}

func NewDiscordService(repo repository.DiscordRepository) DiscordService {
	return &discordServiceImpl{
		repo: repo,
	}
}

func (s *discordServiceImpl) SendMessage(
	ctx context.Context,
	channelID string,
	content string,
) (*entity.Message, error) {
	if channelID == "" {
		return nil, errors.New("channel ID is required")
	}

	if content == "" {
		return nil, errors.New("message content is required")
	}

	return s.repo.SendMessage(ctx, channelID, content)
}

func (s *discordServiceImpl) GetMessage(
	ctx context.Context,
	channelID string,
	messageID string,
) (*entity.Message, error) {
	if channelID == "" {
		return nil, errors.New("channel ID is required")
	}

	if messageID == "" {
		return nil, errors.New("message ID is required")
	}

	return s.repo.GetMessage(ctx, channelID, messageID)
}
