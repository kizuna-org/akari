package main

import (
	"bufio"
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/kizuna-org/akari/internal/di"
	"github.com/kizuna-org/akari/pkg/config"
	discordRepo "github.com/kizuna-org/akari/pkg/discord/domain/repository"
	"github.com/kizuna-org/akari/pkg/discord/handler"
	"github.com/kizuna-org/akari/pkg/llm/domain"
	"github.com/kizuna-org/akari/pkg/llm/usecase/interactor"
	"go.uber.org/fx"
)

var version = "0.1.0"

func setupLogger(envMode config.EnvMode) {
	var handler slog.Handler

	switch envMode {
	case config.EnvModeDevelopment:
		opts := &slog.HandlerOptions{
			Level:       slog.LevelDebug,
			AddSource:   false,
			ReplaceAttr: nil,
		}
		handler = slog.NewTextHandler(os.Stdout, opts)
	case config.EnvModeProduction:
		opts := &slog.HandlerOptions{
			Level:       slog.LevelInfo,
			AddSource:   false,
			ReplaceAttr: nil,
		}
		handler = slog.NewJSONHandler(os.Stdout, opts)
	case config.EnvModeTest:
		opts := &slog.HandlerOptions{
			Level:       slog.LevelDebug,
			AddSource:   false,
			ReplaceAttr: nil,
		}
		handler = slog.NewTextHandler(os.Stdout, opts)
	default:
		opts := &slog.HandlerOptions{
			Level:       slog.LevelInfo,
			AddSource:   false,
			ReplaceAttr: nil,
		}
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	slog.SetDefault(slog.New(handler))
}

func main() {
	showVersion := flag.Bool("version", false, "Show akari version")
	useDiscord := flag.Bool("discord", false, "Start Discord bot mode")
	flag.Parse()

	configRepo := config.NewConfigRepository()
	cfg := configRepo.GetConfig()

	setupLogger(cfg.EnvMode)

	if *showVersion {
		slog.Info("akari version", "version", version)

		return
	}

	if *useDiscord {
		runDiscordBot()

		return
	}

	runCLI()
}

func runDiscordBot() {
	slog.Info("Starting Discord bot mode")

	app := fx.New(
		di.NewModule(),
		fx.NopLogger,
		fx.Invoke(func(
			repo discordRepo.DiscordRepository,
			messageHandler *handler.MessageHandler,
		) {
			messageHandler.RegisterHandlers()

			if err := repo.Start(); err != nil {
				slog.Error("Failed to start Discord bot", "error", err)

				return
			}

			slog.Info("Discord bot is now running. Press CTRL-C to exit.")

			sc := make(chan os.Signal, 1)
			signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
			<-sc

			if err := repo.Stop(); err != nil {
				slog.Error("Failed to stop Discord bot", "error", err)
			}
		}),
	)

	ctx := context.Background()
	if err := app.Start(ctx); err != nil {
		slog.Error("Failed to start application", "error", err)

		return
	}

	if err := app.Stop(ctx); err != nil {
		slog.Error("Failed to stop application", "error", err)
	}
}

func runCLI() {
	slog.Info("You: ")

	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		slog.Error("Failed to read input")

		return
	}

	userMessage := scanner.Text()

	app := fx.New(
		di.NewModule(),
		fx.NopLogger,
		fx.Invoke(func(llmInteractor interactor.LLMInteractor) {
			ctx := context.Background()

			slog.Info("Akari started")

			systemPrompt := "You are a helpful AI assistant."
			history := []*domain.Content{}
			functions := []domain.Function{}

			messages, _, err := llmInteractor.SendChatMessage(ctx, systemPrompt, history, userMessage, functions)
			if err != nil {
				slog.Error("Failed to send message to LLM", "error", err)

				return
			}

			if len(messages) > 0 {
				slog.Info("AI", "response", *messages[0])
			}
		}),
	)

	ctx := context.Background()
	if err := app.Start(ctx); err != nil {
		slog.Error("Failed to start application", "error", err)

		return
	}

	if err := app.Stop(ctx); err != nil {
		slog.Error("Failed to stop application", "error", err)
	}
}
