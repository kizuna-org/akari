package infrastructure

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/kizuna-org/akari/gen/ent/character"
	"github.com/kizuna-org/akari/pkg/database/domain"
)

type Repository interface {
	domain.DatabaseRepository
	domain.SystemPromptRepository
	domain.CharacterRepository
	HealthCheck(ctx context.Context) error
}

type repositoryImpl struct {
	client Client
	logger *slog.Logger
}

func NewRepository(client Client, logger *slog.Logger) Repository {
	return &repositoryImpl{
		client: client,
		logger: logger.With("component", "postgres_repository"),
	}
}

func (r *repositoryImpl) HealthCheck(ctx context.Context) error {
	if err := r.client.Ping(ctx); err != nil {
		return fmt.Errorf("database health check failed: %w", err)
	}

	return nil
}

func (r *repositoryImpl) WithTransaction(ctx context.Context, fn domain.TxFunc) error {
	return r.client.WithTx(ctx, fn)
}

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

func (r *repositoryImpl) CreateCharacter(
	ctx context.Context,
	name string,
) (*domain.Character, error) {
	character, err := r.client.CharacterClient().
		Create().
		SetName(name).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create character: %w", err)
	}

	r.logger.Info("character created",
		slog.Int("id", character.ID),
		slog.String("name", name),
	)

	return character, nil
}

func (r *repositoryImpl) GetCharacterByID(ctx context.Context, characterID int) (*domain.Character, error) {
	char, err := r.client.CharacterClient().Get(ctx, characterID)
	if err != nil {
		return nil, fmt.Errorf("failed to get character by id: %w", err)
	}

	return char, nil
}

func (r *repositoryImpl) GetCharacterWithEdgesByID(
	ctx context.Context,
	characterID int,
) (*domain.Character, error) {
	char, err := r.client.CharacterClient().
		Query().
		Where(character.IDEQ(characterID)).
		WithConfig().
		WithSystemPrompts().
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get character with system prompt: %w", err)
	}

	return char, nil
}

func (r *repositoryImpl) ListCharacters(ctx context.Context) ([]*domain.Character, error) {
	query := r.client.CharacterClient().Query()

	characters, err := query.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list characters: %w", err)
	}

	return characters, nil
}

func (r *repositoryImpl) UpdateCharacter(
	ctx context.Context,
	characterID int,
	name *string,
) (*domain.Character, error) {
	updater := r.client.CharacterClient().UpdateOneID(characterID)

	if name != nil {
		updater = updater.SetName(*name)
	}

	char, err := updater.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to update character: %w", err)
	}

	logAttrs := []any{slog.Int("id", characterID)}
	if name != nil {
		logAttrs = append(logAttrs, slog.String("name", *name))
	}

	r.logger.Info("character updated", logAttrs...)

	return char, nil
}

func (r *repositoryImpl) DeleteCharacter(ctx context.Context, characterID int) error {
	if err := r.client.CharacterClient().DeleteOneID(characterID).Exec(ctx); err != nil {
		return fmt.Errorf("failed to delete character: %w", err)
	}

	r.logger.Info("character deleted",
		slog.Int("id", characterID),
	)

	return nil
}
