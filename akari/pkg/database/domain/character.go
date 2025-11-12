package domain

import (
	"context"

	"github.com/kizuna-org/akari/gen/ent"
)

type CharacterRepository interface {
	CreateCharacter(ctx context.Context, name string) (*Character, error)
	GetCharacterByID(ctx context.Context, characterID int) (*Character, error)
	GetCharacterWithSystemPromptByID(ctx context.Context, characterID int) (*Character, error)
	ListCharacters(ctx context.Context) ([]*Character, error)
	UpdateCharacter(ctx context.Context, characterID int, name *string) (*Character, error)
	DeleteCharacter(ctx context.Context, characterID int) error
}

type Character = ent.Character
