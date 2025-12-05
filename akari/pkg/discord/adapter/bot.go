package adapter

import (
	"context"
	"log/slog"

	"github.com/kizuna-org/akari/pkg/discord/domain/repository"
	"github.com/kizuna-org/akari/pkg/discord/handler"
	"go.uber.org/fx"
)

type BotRunner struct {
	msgHandler  *handler.MessageHandler
	discordRepo repository.DiscordRepository
	logger      *slog.Logger
}

func NewBotRunner(
	msgHandler *handler.MessageHandler,
	discordRepo repository.DiscordRepository,
	logger *slog.Logger,
) *BotRunner {
	return &BotRunner{
		msgHandler:  msgHandler,
		discordRepo: discordRepo,
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

			b.logger.Info("Discord bot started")

			return nil
		},
		OnStop: func(_ context.Context) error {
			b.logger.Info("Shutting down Discord bot")

			return b.discordRepo.Stop()
		},
	})
}
