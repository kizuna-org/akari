package domain_test

import (
	"testing"

	"github.com/kizuna-org/akari/gen/ent"
	"github.com/kizuna-org/akari/pkg/database/domain"
)

func TestFromEntCharacterConfig_Converts(t *testing.T) {
	t.Parallel()

	nameRegex := "^name$"
	entCfg := &ent.CharacterConfig{
		NameRegexp:          &nameRegex,
		DefaultSystemPrompt: "default-systemPrompt",
	}

	characterConfig, err := domain.FromEntCharacterConfig(entCfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if characterConfig == nil {
		t.Fatalf("expected non-nil config")
	}

	if characterConfig.DefaultSystemPrompt != entCfg.DefaultSystemPrompt {
		t.Fatalf(
			"DefaultSystemPrompt mismatch: got=%q want=%q",
			characterConfig.DefaultSystemPrompt, entCfg.DefaultSystemPrompt,
		)
	}

	if characterConfig.NameRegexp == nil {
		t.Fatalf("NameRegexp missing in domain config")
	}

	if entCfg.NameRegexp == nil {
		t.Fatalf("NameRegexp missing in ent fixture")
	}

	if characterConfig.NameRegexp != entCfg.NameRegexp {
		t.Fatalf("NameRegexp pointer mismatch")
	}
}

func TestFromEntCharacterConfig_Nil(t *testing.T) {
	t.Parallel()

	characterConfig, err := domain.FromEntCharacterConfig(nil)
	if err == nil {
		t.Fatalf("expected error when input is nil")
	}

	if characterConfig != nil {
		t.Fatalf("expected nil characterConfig when input is nil")
	}
}
