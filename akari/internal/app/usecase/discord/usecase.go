package discord

import "context"

type DiscordMessageUsecase interface {
	HandleMessage(ctx context.Context, channelID, content, systemPrompt string) error
}
