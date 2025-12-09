package adapter

import (
	"context"
	"errors"
	"log/slog"

	"github.com/kizuna-org/akari/pkg/discord/domain/repository"
	"github.com/kizuna-org/akari/pkg/discord/domain/service"
	"github.com/kizuna-org/akari/pkg/discord/handler"
	"github.com/kizuna-org/akari/pkg/discord/infrastructure"
	"go.uber.org/fx"
)

type BotRunner struct {
	msgHandler  *handler.MessageHandler
	discordRepo repository.DiscordRepository
	interactor  service.HandleMessageInteractor
	client      *infrastructure.DiscordClient
	logger      *slog.Logger
}

func NewBotRunner(
	msgHandler *handler.MessageHandler,
	discordRepo repository.DiscordRepository,
	interactor service.HandleMessageInteractor,
	client *infrastructure.DiscordClient,
	logger *slog.Logger,
) *BotRunner {
	return &BotRunner{
		msgHandler:  msgHandler,
		discordRepo: discordRepo,
		interactor:  interactor,
		client:      client,
		logger:      logger,
	}
}

func (b *BotRunner) RegisterLifecycle(lc fx.Lifecycle) {
	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			b.msgHandler.RegisterHandlers()

			if err := b.discordRepo.Start(); err != nil {
				return err
			}

			if b.client.Session == nil || b.client.Session.State == nil || b.client.Session.State.User == nil {
				return errors.New("discord session not ready")
			}

			b.interactor.SetBotUserID(b.client.Session.State.User.ID)
			b.logger.Info("Discord bot started", "bot_id", b.client.Session.State.User.ID)

			return nil
		},
		OnStop: func(_ context.Context) error {
			b.logger.Info("Shutting down Discord bot")

			return b.discordRepo.Stop()
		},
	})
}
