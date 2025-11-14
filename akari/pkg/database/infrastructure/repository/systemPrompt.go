package repository

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/kizuna-org/akari/pkg/database/domain"
)

func (r *repositoryImpl) CreateSystemPrompt(
	ctx context.Context,
	title, prompt string,
	purpose domain.SystemPromptPurpose,
) (*domain.SystemPrompt, error) {
	systemPrompt, err := r.client.SystemPromptClient().
		Create().
		SetTitle(title).
		SetPrompt(prompt).
		SetPurpose(purpose).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create system prompt: %w", err)
	}

	r.logger.Info("system prompt created",
		slog.Int("id", systemPrompt.ID),
		slog.String("title", title),
		slog.String("purpose", string(purpose)),
		slog.String("prompt", prompt),
	)

	return systemPrompt, nil
}

func (r *repositoryImpl) GetSystemPromptByID(ctx context.Context, id int) (*domain.SystemPrompt, error) {
	systemPrompt, err := r.client.SystemPromptClient().Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get system prompt by id: %w", err)
	}

	return systemPrompt, nil
}

func (r *repositoryImpl) UpdateSystemPrompt(
	ctx context.Context,
	promptID int,
	title, prompt *string,
	purpose *domain.SystemPromptPurpose,
) (*domain.SystemPrompt, error) {
	updater := r.client.SystemPromptClient().UpdateOneID(promptID)
	if title != nil {
		updater = updater.SetTitle(*title)
	}

	if purpose != nil {
		updater = updater.SetPurpose(*purpose)
	}

	if prompt != nil {
		updater = updater.SetPrompt(*prompt)
	}

	systemPrompt, err := updater.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to update system prompt: %w", err)
	}

	logAttrs := []any{slog.Int("id", promptID)}
	if title != nil {
		logAttrs = append(logAttrs, slog.String("title", *title))
	}

	if purpose != nil {
		logAttrs = append(logAttrs, slog.String("purpose", string(*purpose)))
	}

	if prompt != nil {
		logAttrs = append(logAttrs, slog.String("prompt", *prompt))
	}

	r.logger.Info("system prompt updated", logAttrs...)

	return systemPrompt, nil
}

func (r *repositoryImpl) DeleteSystemPrompt(ctx context.Context, promptID int) error {
	if err := r.client.SystemPromptClient().DeleteOneID(promptID).Exec(ctx); err != nil {
		return fmt.Errorf("failed to delete system prompt: %w", err)
	}

	r.logger.Info("system prompt deleted",
		slog.Int("id", promptID),
	)

	return nil
}
