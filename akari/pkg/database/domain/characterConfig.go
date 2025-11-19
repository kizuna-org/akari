package domain

import (
	"github.com/kizuna-org/akari/gen/ent"
)

type CharacterConfig struct {
	NameRegexp          *string
	DefaultSystemPrompt string
}

func FromEntCharacterConfig(entCharacterConfig *ent.CharacterConfig) *CharacterConfig {
	if entCharacterConfig == nil {
		return nil
	}

	return &CharacterConfig{
		NameRegexp:          entCharacterConfig.NameRegexp,
		DefaultSystemPrompt: entCharacterConfig.DefaultSystemPrompt,
	}
}
