package domain

//go:generate go tool mockgen -package=mock -source=character.go -destination=mock/character.go

import (
	"context"
	"time"

	"github.com/kizuna-org/akari/gen/ent"
)

type CharacterRepository interface {
	GetCharacterByID(ctx context.Context, characterID int) (*Character, error)
	ListCharacters(ctx context.Context) ([]*Character, error)
}

type Character struct {
	ID            int
	Name          string
	CreatedAt     time.Time
	UpdatedAt     time.Time
	Config        *CharacterConfig
	SystemPrompts []*SystemPrompt
}

func FromEntCharacter(entCharacter *ent.Character) *Character {
	if entCharacter == nil {
		return nil
	}

	var characterConfig *CharacterConfig
	if entCharacter.Edges.Config != nil {
		characterConfig = FromEntCharacterConfig(entCharacter.Edges.Config)
	}

	var systemPrompts []*SystemPrompt
	if entCharacter.Edges.SystemPrompts != nil {
		systemPrompts = make([]*SystemPrompt, len(entCharacter.Edges.SystemPrompts))
		for i, systemPrompt := range entCharacter.Edges.SystemPrompts {
			systemPrompts[i] = FromEntSystemPrompt(systemPrompt)
		}
	}

	return &Character{
		ID:            entCharacter.ID,
		Name:          entCharacter.Name,
		CreatedAt:     entCharacter.CreatedAt,
		UpdatedAt:     entCharacter.UpdatedAt,
		Config:        characterConfig,
		SystemPrompts: systemPrompts,
	}
}
