package repository

//go:generate go tool mockgen -package=mock -source=discord.go -destination=mock/discord.go

import (
	"context"

	"github.com/kizuna-org/akari/pkg/discord/domain/entity"
)

type DiscordRepository interface {
	SendMessage(ctx context.Context, channelID string, content string) (*entity.Message, error)
	GetMessage(ctx context.Context, channelID string, messageID string) (*entity.Message, error)
	Start() error
	Stop() error
}
