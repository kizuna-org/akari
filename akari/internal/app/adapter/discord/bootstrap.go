package discord

import (
	"context"
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

	app := fx.New(
		di.NewModule(),
		fx.NopLogger,
		fx.Invoke(func(
			repo repository.DiscordRepository,
			usecase internalUsecase.DiscordMessageUsecase,
			characterInteractor databaseInteractor.CharacterInteractor,
			client *infrastructure.DiscordClient,
		) {
			character, err := characterInteractor.GetCharacterWithEdgesByID(context.Background(), defaultCharacterID)
			if err != nil {
				slog.Error("Failed to get character", "error", err)

				return
			}

			client.Session.AddHandler(makeHandler(
				usecase,
				*character.Edges.Config.NameRegexp,
				character.Edges.SystemPrompts[defaultSystemPromptID].Prompt,
			))

			if err := repo.Start(); err != nil {
				slog.Error("Failed to start Discord bot", "error", err)

				return
			}

			slog.Info("Discord bot is now running. Press CTRL-C to exit.")

			sc := make(chan os.Signal, 1)
			signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM)
			<-sc

			if err := repo.Stop(); err != nil {
				slog.Error("Failed to stop Discord bot", "error", err)
			}
		}),
	)

	if err := app.Start(ctx); err != nil {
		slog.Error("Failed to start application", "error", err)

		return err
	}

	if err := app.Stop(ctx); err != nil {
		slog.Error("Failed to stop application", "error", err)

		return err
	}

	return nil
}
