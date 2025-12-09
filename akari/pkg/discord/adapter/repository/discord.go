package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/kizuna-org/akari/pkg/discord/domain/entity"
	"github.com/kizuna-org/akari/pkg/discord/domain/repository"
	"github.com/kizuna-org/akari/pkg/discord/infrastructure"
)

const defaultReadyTimeout = 10 * time.Second

type discordRepositoryImpl struct {
	client       *infrastructure.DiscordClient
	readyTimeout time.Duration
}

func NewDiscordRepository(
	client *infrastructure.DiscordClient,
	timeout time.Duration,
) repository.DiscordRepository {
	if timeout == 0 {
		timeout = defaultReadyTimeout
	}

	return &discordRepositoryImpl{
		client:       client,
		readyTimeout: timeout,
	}
}

func (r *discordRepositoryImpl) SendMessage(
	ctx context.Context,
	channelID string,
	content string,
) (*entity.Message, error) {
	msg, err := r.client.Session.ChannelMessageSend(channelID, content)
	if err != nil {
		return nil, fmt.Errorf("repository: failed to send message: %w", err)
	}

	return &entity.Message{
		ID:        msg.ID,
		ChannelID: msg.ChannelID,
		GuildID:   msg.GuildID,
		AuthorID:  msg.Author.ID,
		Content:   msg.Content,
		Timestamp: msg.Timestamp,
		IsBot:     msg.Author.Bot,
		Mentions:  make([]string, 0),
	}, nil
}

func (r *discordRepositoryImpl) GetMessage(
	ctx context.Context,
	channelID string,
	messageID string,
) (*entity.Message, error) {
	msg, err := r.client.Session.ChannelMessage(channelID, messageID)
	if err != nil {
		return nil, fmt.Errorf("repository: failed to get message: %w", err)
	}

	return &entity.Message{
		ID:        msg.ID,
		ChannelID: msg.ChannelID,
		GuildID:   msg.GuildID,
		AuthorID:  msg.Author.ID,
		Content:   msg.Content,
		Timestamp: msg.Timestamp,
		IsBot:     msg.Author.Bot,
		Mentions:  make([]string, 0),
	}, nil
}

func (r *discordRepositoryImpl) Start() error {
	r.client.RegisterReadyHandler()

	if err := r.client.Session.Open(); err != nil {
		return fmt.Errorf("repository: failed to open discord session: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), r.readyTimeout)
	defer cancel()

	if err := r.client.WaitReady(ctx); err != nil {
		if err := r.client.Session.Close(); err != nil {
			return fmt.Errorf("repository: failed to close discord session after ready timeout: %w", err)
		}

		return fmt.Errorf("repository: failed to wait for discord ready: %w", err)
	}

	return nil
}

func (r *discordRepositoryImpl) Stop() error {
	if err := r.client.Session.Close(); err != nil {
		return fmt.Errorf("repository: failed to close discord session: %w", err)
	}

	return nil
}
