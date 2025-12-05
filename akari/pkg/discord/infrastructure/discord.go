package infrastructure

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/bwmarrin/discordgo"
)

type DiscordClient struct {
	Session     *discordgo.Session
	readyOnce   sync.Once
	readySignal chan struct{}
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
		Session:     session,
		readyOnce:   sync.Once{},
		readySignal: make(chan struct{}),
	}, nil
}

func (c *DiscordClient) WaitReady(ctx context.Context) error {
	select {
	case <-c.readySignal:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("failed to wait for discord ready: %w", ctx.Err())
	}
}

func (c *DiscordClient) RegisterReadyHandler() {
	c.Session.AddHandler(c.onReady)
}

func (c *DiscordClient) onReady(_ *discordgo.Session, _ *discordgo.Ready) {
	c.readyOnce.Do(func() {
		close(c.readySignal)
	})
}
