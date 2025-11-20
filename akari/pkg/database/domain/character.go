package domain

//go:generate go tool mockgen -package=mock -source=character.go -destination=mock/character.go

import (
	"context"
	"errors"
	"time"

	"github.com/kizuna-org/akari/gen/ent"
)

type CharacterRepository interface {
	GetCharacterByID(ctx context.Context, characterID int) (*Character, error)
	ListCharacters(ctx context.Context) ([]*Character, error)
}

type Character struct {
	ID              int
	Name            string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	Config          *Config
	SystemPromptIDs []int
}

type Config struct {
	NameRegExp          *string
	DefaultSystemPrompt string
}

func FromEntCharacter(entCharacter *ent.Character) (*Character, error) {
	if entCharacter == nil {
		return nil, errors.New("character is nil")
	}

	if entCharacter.Edges.Config == nil {
		return nil, errors.New("character.Config edge is nil")
	}

	characterConfig := fromEntCharacterConfig(entCharacter.Edges.Config)

	if entCharacter.Edges.SystemPrompts == nil {
		return nil, errors.New("character.SystemPrompts edge is nil")
	}

	systemPromptIDs := make([]int, len(entCharacter.Edges.SystemPrompts))
	for i, systemPrompt := range entCharacter.Edges.SystemPrompts {
		systemPromptIDs[i] = systemPrompt.ID
	}

	return &Character{
		ID:              entCharacter.ID,
		Name:            entCharacter.Name,
		CreatedAt:       entCharacter.CreatedAt,
		UpdatedAt:       entCharacter.UpdatedAt,
		Config:          characterConfig,
		SystemPromptIDs: systemPromptIDs,
	}, nil
}

func fromEntCharacterConfig(entCharacterConfig *ent.CharacterConfig) *Config {
	return &Config{
		NameRegExp:          entCharacterConfig.NameRegexp,
		DefaultSystemPrompt: entCharacterConfig.DefaultSystemPrompt,
	}
}
