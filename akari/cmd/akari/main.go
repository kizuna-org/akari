package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/kizuna-org/akari/internal/di"
	"github.com/kizuna-org/akari/pkg/config"
	databaseInteractor "github.com/kizuna-org/akari/pkg/database/usecase/interactor"
	discordRepo "github.com/kizuna-org/akari/pkg/discord/domain/repository"
	discordInfra "github.com/kizuna-org/akari/pkg/discord/infrastructure"
	"github.com/kizuna-org/akari/pkg/llm/domain"
	"github.com/kizuna-org/akari/pkg/llm/usecase/interactor"
	"go.uber.org/fx"
)

const (
	version               = "0.1.0"
	defaultSystemPrompt   = "You are a helpful AI assistant."
	defaultSystemPromptID = 1
	messageTimeout        = 30 * time.Second
)

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

func getSystemPrompt(ctx context.Context, systemPromptInteractor databaseInteractor.SystemPromptInteractor) string {
	systemPrompt, err := systemPromptInteractor.GetSystemPromptByID(ctx, defaultSystemPromptID)
	if err != nil {
		slog.Error("Failed to get system prompt", "error", err)

		return defaultSystemPrompt
	}

	return systemPrompt.Prompt
}

func getLLMResponse(
	ctx context.Context,
	llmInteractor interactor.LLMInteractor,
	systemPrompt, userMessage string,
) (*string, error) {
	responses, _, err := llmInteractor.SendChatMessage(ctx, systemPrompt, nil, userMessage, nil)
	if err != nil {
		return nil, err
	}

	if len(responses) == 0 {
		return nil, errors.New("no response from LLM")
	}

	return responses[0], nil
}

func handleDiscordMessage(
	repo discordRepo.DiscordRepository,
	llmInteractor interactor.LLMInteractor,
	systemPromptInteractor databaseInteractor.SystemPromptInteractor,
) func(*discordgo.Session, *discordgo.MessageCreate) {
	return func(session *discordgo.Session, message *discordgo.MessageCreate) {
		if message.Author.Bot {
			return
		}

		slog.Info("Received message",
			"author", message.Author.Username,
			"content", message.Content,
			"channel_id", message.ChannelID,
			"message_id", message.ID,
		)

		ctx, cancel := context.WithTimeout(context.Background(), messageTimeout)
		defer cancel()

		response, err := getLLMResponse(
			ctx,
			llmInteractor,
			getSystemPrompt(ctx, systemPromptInteractor),
			message.Content,
		)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				slog.Error("Request timed out", "error", err, "timeout", messageTimeout)
			} else {
				slog.Error("Failed to get response", "error", err)
			}

			return
		}

		if _, err = repo.SendMessage(ctx, message.ChannelID, *response); err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				slog.Error("Sending message timed out", "error", err, "timeout", messageTimeout)
			} else {
				slog.Error("Failed to send message", "error", err)
			}
		}
	}
}

func runDiscordBot() {
	slog.Info("Starting Discord bot mode")

	app := fx.New(
		di.NewModule(),
		fx.NopLogger,
		fx.Invoke(func(
			repo discordRepo.DiscordRepository,
			llmInteractor interactor.LLMInteractor,
			systemPromptInteractor databaseInteractor.SystemPromptInteractor,
			client *discordInfra.DiscordClient,
		) {
			client.Session.AddHandler(handleDiscordMessage(repo, llmInteractor, systemPromptInteractor))

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

			history := []*domain.Content{}
			functions := []domain.Function{}

			messages, _, err := llmInteractor.SendChatMessage(ctx, defaultSystemPrompt, history, userMessage, functions)
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
