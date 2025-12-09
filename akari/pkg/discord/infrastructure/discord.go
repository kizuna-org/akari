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
	readyMutex  sync.Mutex
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
		readyMutex:  sync.Mutex{},
		readyOnce:   sync.Once{},
		readySignal: nil,
	}, nil
}

func (c *DiscordClient) WaitReady(ctx context.Context) error {
	c.readyMutex.Lock()
	readySignal := c.readySignal
	c.readyMutex.Unlock()

	if readySignal == nil {
		return errors.New("ready handler not registered: call RegisterReadyHandler before WaitReady")
	}

	select {
	case <-readySignal:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("failed to wait for discord ready: %w", ctx.Err())
	}
}

func (c *DiscordClient) RegisterReadyHandler() {
	c.readyMutex.Lock()
	if c.readySignal == nil {
		c.readySignal = make(chan struct{})
	}
	c.readyMutex.Unlock()

	c.Session.AddHandler(c.onReady)
}

func (c *DiscordClient) onReady(_ *discordgo.Session, _ *discordgo.Ready) {
	c.readyOnce.Do(func() {
		close(c.readySignal)
	})
}
