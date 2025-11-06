package infrastructure

import (
	"errors"
	"fmt"

	"github.com/bwmarrin/discordgo"
)

type DiscordClient struct {
	Session *discordgo.Session
}

func NewDiscordClient(token string) (*DiscordClient, error) {
	if token == "" {
		return nil, errors.New("discord token is required")
	}

	session, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, fmt.Errorf("failed to create discord session: %w", err)
	}

	session.Identify.Intents = discordgo.IntentsGuildMessages |
		discordgo.IntentsMessageContent |
		discordgo.IntentsGuilds

	return &DiscordClient{
		Session: session,
	}, nil
}
