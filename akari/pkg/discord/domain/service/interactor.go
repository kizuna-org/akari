package service

//go:generate go tool mockgen -package=mock -source=interactor.go -destination=mock/interactor.go

import (
	"context"

	databaseDomain "github.com/kizuna-org/akari/pkg/database/domain"
)

type DiscordData struct {
	User     *databaseDomain.DiscordUser
	Message  *databaseDomain.DiscordMessage
	Mentions []string
	Channel  *databaseDomain.DiscordChannel
	Guild    *databaseDomain.DiscordGuild
}

type HandleMessageInteractor interface {
	Handle(ctx context.Context, discordParams *DiscordData) error
	SetBotUserID(botUserID string)
}
