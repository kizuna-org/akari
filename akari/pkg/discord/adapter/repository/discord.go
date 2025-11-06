package repository

import (
	"context"
	"fmt"

	"github.com/kizuna-org/akari/pkg/discord/domain/entity"
	"github.com/kizuna-org/akari/pkg/discord/domain/repository"
	"github.com/kizuna-org/akari/pkg/discord/infrastructure"
)

type discordRepositoryImpl struct {
	client *infrastructure.DiscordClient
}

func NewDiscordRepository(client *infrastructure.DiscordClient) repository.DiscordRepository {
	return &discordRepositoryImpl{
		client: client,
	}
}

func (r *discordRepositoryImpl) SendMessage(
    ctx context.Context,
    channelID string,
    content string,
) (*entity.Message, error) {
	msg, err := r.client.Session.ChannelMessageSend(channelID, content)
	if err != nil {
		return nil, fmt.Errorf("failed to send message: %w", err)
	}

	return &entity.Message{
		ID:        msg.ID,
		ChannelID: msg.ChannelID,
		GuildID:   msg.GuildID,
		AuthorID:  msg.Author.ID,
		Content:   msg.Content,
		Timestamp: msg.Timestamp,
	}, nil
}

func (r *discordRepositoryImpl) GetMessage(
    ctx context.Context,
    channelID string,
    messageID string,
) (*entity.Message, error) {
	msg, err := r.client.Session.ChannelMessage(channelID, messageID)
	if err != nil {
		return nil, fmt.Errorf("failed to get message: %w", err)
	}

	return &entity.Message{
		ID:        msg.ID,
		ChannelID: msg.ChannelID,
		GuildID:   msg.GuildID,
		AuthorID:  msg.Author.ID,
		Content:   msg.Content,
		Timestamp: msg.Timestamp,
	}, nil
}

func (r *discordRepositoryImpl) Start() error {
	if err := r.client.Session.Open(); err != nil {
		return fmt.Errorf("failed to open discord session: %w", err)
	}

	return nil
}

func (r *discordRepositoryImpl) Stop() error {
	if err := r.client.Session.Close(); err != nil {
		return fmt.Errorf("failed to close discord session: %w", err)
	}

	return nil
}
