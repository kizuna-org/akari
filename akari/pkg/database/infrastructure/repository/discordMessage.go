package repository

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/kizuna-org/akari/pkg/database/domain"
)

func (r *repositoryImpl) CreateDiscordMessage(
	ctx context.Context,
	params domain.DiscordMessage,
) (*domain.DiscordMessage, error) {
	builder := r.client.DiscordMessageClient().Create().
		SetID(params.ID).
		SetChannelID(params.ChannelID).
		SetAuthorID(params.AuthorID).
		SetContent(params.Content).
		SetTimestamp(params.Timestamp)

	message, err := builder.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create discord message: %w", err)
	}

	r.logger.Info("Discord message created",
		slog.String("message_id", message.ID),
		slog.String("channel_id", message.Edges.Channel.ID),
		slog.String("author_id", message.AuthorID),
		slog.String("timestamp", message.Timestamp.String()),
	)

	return domain.ToDomainDiscordMessageFromDB(message), nil
}

func (r *repositoryImpl) GetDiscordMessageByID(
	ctx context.Context,
	messageID string,
) (*domain.DiscordMessage, error) {
	message, err := r.client.DiscordMessageClient().Get(ctx, messageID)
	if err != nil {
		return nil, fmt.Errorf("failed to get discord message by id: %w", err)
	}

	return domain.ToDomainDiscordMessageFromDB(message), nil
}

func (r *repositoryImpl) ListDiscordMessages(ctx context.Context) ([]*domain.DiscordMessage, error) {
	messages, err := r.client.DiscordMessageClient().Query().All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list discord messages: %w", err)
	}

	domainDiscordMessages := make([]*domain.DiscordMessage, 0, len(messages))
	for _, domainDiscordMessage := range messages {
		domainDiscordMessages = append(domainDiscordMessages, domain.ToDomainDiscordMessageFromDB(domainDiscordMessage))
	}

	return domainDiscordMessages, nil
}

func (r *repositoryImpl) DeleteDiscordMessage(ctx context.Context, messageID string) error {
	if err := r.client.DiscordMessageClient().DeleteOneID(messageID).Exec(ctx); err != nil {
		return fmt.Errorf("failed to delete discord message: %w", err)
	}

	r.logger.Info("Discord message deleted",
		slog.String("id", messageID),
	)

	return nil
}
