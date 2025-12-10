package service

//go:generate go tool mockgen -package=mock -source=interactor.go -destination=mock/interactor.go

import (
	"context"

	"github.com/kizuna-org/akari/pkg/discord/domain/entity"
)

type HandleMessageInteractor interface {
	Handle(ctx context.Context, message *entity.Message, channel *entity.Channel, guild *entity.Guild) error
	SetBotUserID(botUserID string)
}
