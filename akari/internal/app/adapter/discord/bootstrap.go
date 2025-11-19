package discord

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	internalUsecase "github.com/kizuna-org/akari/internal/app/usecase/discord"
	"github.com/kizuna-org/akari/internal/di"
	databaseInteractor "github.com/kizuna-org/akari/pkg/database/usecase/interactor"
	"github.com/kizuna-org/akari/pkg/discord/domain/repository"
	"github.com/kizuna-org/akari/pkg/discord/infrastructure"
	"go.uber.org/fx"
)

func RunDiscordBot(ctx context.Context) error {
	slog.Info("Starting Discord bot mode")

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)

	app := fx.New(
		di.NewModule(),
		fx.NopLogger,
		fx.Invoke(initDiscord),
	)

	if err := app.Start(ctx); err != nil {
		slog.Error("discord: failed to start application", "error", err)

		return fmt.Errorf("discord: failed to start application: %w", err)
	}

	<-signalCh
	slog.Info("Received shutdown signal")

	if err := app.Stop(ctx); err != nil {
		slog.Error("discord: failed to stop application", "error", err)

		return fmt.Errorf("discord: failed to stop application: %w", err)
	}

	return nil
}

func initDiscord(
	lifecycle fx.Lifecycle,
	repo repository.DiscordRepository,
	usecase internalUsecase.DiscordMessageUsecase,
	characterInteractor databaseInteractor.CharacterInteractor,
	client *infrastructure.DiscordClient,
) error {
	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			character, err := characterInteractor.GetCharacterByID(ctx, defaultCharacterID)
			if err != nil {
				return fmt.Errorf("discord: failed to get character: %w", err)
			}

			var nameRegexp string
			if character.Config != nil && character.Config.NameRegexp != nil {
				nameRegexp = *character.Config.NameRegexp
			}

			var prompt string
			if len(character.SystemPrompts) > defaultSystemPromptID {
				prompt = character.SystemPrompts[defaultSystemPromptID].Prompt
			}

			client.Session.AddHandler(makeHandler(ctx, usecase, nameRegexp, prompt))

			if err := repo.Start(); err != nil {
				return fmt.Errorf("discord: failed to start repository: %w", err)
			}
			slog.Info("Discord bot is now running. Press CTRL-C to exit.")

			return nil
		},
		OnStop: func(ctx context.Context) error {
			if err := repo.Stop(); err != nil {
				return fmt.Errorf("discord: failed to stop repository: %w", err)
			}

			return nil
		},
	})

	return nil
}
