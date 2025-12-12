package repository

//go:generate go tool mockgen -package=mock -source=discord.go -destination=mock/discord.go

import (
	"context"

	databaseDomain "github.com/kizuna-org/akari/pkg/database/domain"
)

type DiscordRepository interface {
	SendMessage(ctx context.Context, channelID string, content string) (*databaseDomain.DiscordMessage, error)
	GetMessage(ctx context.Context, channelID string, messageID string) (*databaseDomain.DiscordMessage, error)
	Start() error
	Stop() error
}
