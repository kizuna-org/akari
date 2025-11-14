package domain

//go:generate go tool mockgen -package=mock -source=character.go -destination=mock/character.go

import (
	"context"

	"github.com/kizuna-org/akari/gen/ent"
)

type CharacterRepository interface {
	GetCharacterByID(ctx context.Context, characterID int) (*Character, error)
	ListCharacters(ctx context.Context) ([]*Character, error)
}

type Character = ent.Character
