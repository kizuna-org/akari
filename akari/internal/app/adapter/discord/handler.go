package discord

import (
	"context"
	"regexp"

	"log/slog"

	"github.com/bwmarrin/discordgo"
	internalUsecase "github.com/kizuna-org/akari/internal/app/usecase/discord"
)

func isBotMentioned(session *discordgo.Session, message *discordgo.MessageCreate, botNameRegExp string) bool {
	for _, mention := range message.Mentions {
		if mention.ID == session.State.User.ID {
			return true
		}
	}

	return regexp.MustCompile(botNameRegExp).MatchString(message.Content)
}

func makeHandler(
	ctx context.Context,
	usecase internalUsecase.DiscordMessageUsecase,
	botNameRegExp, systemPrompt string,
) func(*discordgo.Session, *discordgo.MessageCreate) {
	return func(session *discordgo.Session, message *discordgo.MessageCreate) {
		if message.Author.Bot || !isBotMentioned(session, message, botNameRegExp) {
			return
		}

		slog.Info("Received message",
			"author", message.Author.Username,
			"content", message.Content,
			"channel_id", message.ChannelID,
			"message_id", message.ID,
		)

		if err := usecase.HandleMessage(ctx, message.ChannelID, message.Content, systemPrompt); err != nil {
			slog.Error("discord: usecase.HandleMessage failed", "error", err)
		}
	}
}
