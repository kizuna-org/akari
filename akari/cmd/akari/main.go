package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/kizuna-org/akari/internal/di"
	"github.com/kizuna-org/akari/pkg/config"
	"github.com/kizuna-org/akari/pkg/llm/adapter/repository"
	"github.com/kizuna-org/akari/pkg/llm/usecase/interactor"
	"go.uber.org/fx"
	"google.golang.org/genai"
)

var version = "0.1.0"

func setupLogger(envMode config.EnvMode) {
	var handler slog.Handler
	
	switch envMode {
	case config.EnvModeDevelopment:
		opts := &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}
		handler = slog.NewTextHandler(os.Stdout, opts)
	case config.EnvModeProduction:
		opts := &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}
		handler = slog.NewJSONHandler(os.Stdout, opts)
	default:
		opts := &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}
		handler = slog.NewTextHandler(os.Stdout, opts)
	}
	
	slog.SetDefault(slog.New(handler))
}

func main() {
	showVersion := flag.Bool("version", false, "Show akari version")
	flag.Parse()

	configRepo := config.NewConfigRepository()
	cfg := configRepo.GetConfig()

	setupLogger(cfg.EnvMode)

	if *showVersion {
		slog.Info("akari version", "version", version)
		return
	}

	fmt.Print("You: ")
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		slog.Error("Failed to read input")
		return
	}
	userMessage := scanner.Text()

	app := fx.New(
		di.Module,
		fx.NopLogger,
		fx.Invoke(func(llmInteractor interactor.LLMInteractor) {
			ctx := context.Background()
			slog.Info("Akari started")

			systemPrompt := "You are a helpful AI assistant."
			history := []*genai.Content{}
			functions := []repository.AkariFunction{}

			messages, _, err := llmInteractor.SendChatMessage(ctx, systemPrompt, history, userMessage, functions)
			if err != nil {
				slog.Error("Failed to send message to LLM", "error", err)
				return
			}

			if len(messages) > 0 {
				fmt.Printf("AI: %s\n", *messages[0])
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
