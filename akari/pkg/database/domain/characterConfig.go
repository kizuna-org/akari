package domain

import (
	"errors"

	"github.com/kizuna-org/akari/gen/ent"
)

type CharacterConfig struct {
	NameRegexp          *string
	DefaultSystemPrompt string
}

func FromEntCharacterConfig(entCharacterConfig *ent.CharacterConfig) (*CharacterConfig, error) {
	if entCharacterConfig == nil {
		return nil, errors.New("characterConfig is nil")
	}

	return &CharacterConfig{
		NameRegexp:          entCharacterConfig.NameRegexp,
		DefaultSystemPrompt: entCharacterConfig.DefaultSystemPrompt,
	}, nil
}
