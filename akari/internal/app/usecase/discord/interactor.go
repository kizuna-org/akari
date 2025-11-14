package discord

import (
	"context"
	"errors"
	"log/slog"
	"time"

	discordInteractor "github.com/kizuna-org/akari/pkg/discord/usecase/interactor"
	llmInteractor "github.com/kizuna-org/akari/pkg/llm/usecase/interactor"
)

const messageTimeout = 30 * time.Second

type discordMessageUsecaseImpl struct {
	discord discordInteractor.DiscordInteractor
	llm     llmInteractor.LLMInteractor
	logger  *slog.Logger
}

func NewDiscordMessageUsecase(
	discord discordInteractor.DiscordInteractor,
	llm llmInteractor.LLMInteractor,
	logger *slog.Logger,
) DiscordMessageUsecase {
	return &discordMessageUsecaseImpl{
		discord: discord,
		llm:     llm,
		logger:  logger,
	}
}

func (u *discordMessageUsecaseImpl) HandleMessage(ctx context.Context, channelID, content, systemPrompt string) error {
	ctx, cancel := context.WithTimeout(ctx, messageTimeout)
	defer cancel()

	responses, _, err := u.llm.SendChatMessage(ctx, systemPrompt, nil, content, nil)
	if err != nil {
		u.logger.Error("Failed to send request to LLM", "error", err)

		return err
	}

	if len(responses) == 0 {
		u.logger.Error("No response from LLM")

		return errors.New("no response from llm")
	}

	if _, err := u.discord.SendMessage(ctx, channelID, *responses[0]); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			u.logger.Error("Sending message timed out", "error", err, "timeout", messageTimeout)
		} else {
			u.logger.Error("Failed to send message", "error", err)
		}

		return err
	}

	return nil
}
