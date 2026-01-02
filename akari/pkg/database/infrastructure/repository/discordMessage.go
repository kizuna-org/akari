package repository

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/kizuna-org/akari/gen/ent/discordmessage"
	"github.com/kizuna-org/akari/pkg/database/domain"
)

func (r *repositoryImpl) CreateDiscordMessage(
	ctx context.Context,
	params domain.DiscordMessage,
) (*domain.DiscordMessage, error) {
	builder := r.client.DiscordMessageClient().Create().
		SetID(params.ID).
		SetAuthorID(params.AuthorID).
		SetChannelID(params.ChannelID).
		SetContent(params.Content).
		SetTimestamp(params.Timestamp)

	message, err := builder.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create discord message: %w", err)
	}

	r.logger.Info("Discord message created",
		slog.String("message_id", message.ID),
		slog.String("author_id", params.AuthorID),
		slog.String("channel_id", params.ChannelID),
		slog.String("timestamp", message.Timestamp.String()),
	)

	// Create domain object directly since we already have AuthorID and ChannelID
	return &domain.DiscordMessage{
		ID:        message.ID,
		AuthorID:  params.AuthorID,
		ChannelID: params.ChannelID,
		Content:   message.Content,
		Timestamp: message.Timestamp,
		CreatedAt: message.CreatedAt,
	}, nil
}

func (r *repositoryImpl) GetDiscordMessageByID(
	ctx context.Context,
	messageID string,
) (*domain.DiscordMessage, error) {
	message, err := r.client.DiscordMessageClient().
		Query().
		Where(discordmessage.IDEQ(messageID)).
		WithAuthor().
		WithChannel().
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get discord message by id: %w", err)
	}

	return domain.FromEntDiscordMessage(message)
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
