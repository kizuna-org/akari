package service

//go:generate go tool mockgen -package=mock -source=interactor.go -destination=mock/interactor.go

import (
	"context"

	"github.com/kizuna-org/akari/pkg/discord/domain/entity"
)

type DiscordData struct {
	User    *entity.User
	Message *entity.Message
	Channel *entity.Channel
	Guild   *entity.Guild
}

type HandleMessageInteractor interface {
	Handle(ctx context.Context, discordParams *DiscordData) error
	SetBotUserID(botUserID string)
}
