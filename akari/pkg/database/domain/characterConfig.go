package domain

import (
	"context"

	"github.com/kizuna-org/akari/gen/ent"
)

type CharacterConfigRepository interface {
	CreateCharacterConfig(
		ctx context.Context,
		characterID *int,
		nameRegexp *string,
		defaultSystemPrompt string,
	) (*CharacterConfig, error)
	GetCharacterConfigByID(ctx context.Context, id int) (*CharacterConfig, error)
	GetCharacterConfigByCharacterID(ctx context.Context, characterID int) (*CharacterConfig, error)
	UpdateCharacterConfig(
		ctx context.Context,
		id int,
		characterID *int,
		nameRegexp *string,
		defaultSystemPrompt *string,
	) (*CharacterConfig, error)
	DeleteCharacterConfig(ctx context.Context, id int) error
}

type CharacterConfig = ent.CharacterConfig
