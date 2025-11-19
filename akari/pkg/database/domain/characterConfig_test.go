package domain_test

import (
	"testing"

	"github.com/kizuna-org/akari/gen/ent"
	"github.com/kizuna-org/akari/pkg/database/domain"
)

func TestFromEntCharacterConfig_Converts(t *testing.T) {
	t.Parallel()

	nameRegex := "^hello$"
	entCfg := &ent.CharacterConfig{
		NameRegexp:          &nameRegex,
		DefaultSystemPrompt: "default-prompt",
	}

	cfg := domain.FromEntCharacterConfig(entCfg)
	if cfg == nil {
		t.Fatalf("expected non-nil config")
	}

	if cfg.DefaultSystemPrompt != entCfg.DefaultSystemPrompt {
		t.Fatalf("DefaultSystemPrompt mismatch: got=%q want=%q", cfg.DefaultSystemPrompt, entCfg.DefaultSystemPrompt)
	}

	if cfg.NameRegexp == nil {
		t.Fatalf("NameRegexp missing in domain config")
	}

	if entCfg.NameRegexp == nil {
		t.Fatalf("NameRegexp missing in ent fixture")
	}

	if cfg.NameRegexp != entCfg.NameRegexp {
		t.Fatalf("NameRegexp pointer mismatch")
	}
}

func TestFromEntCharacterConfig_Nil(t *testing.T) {
	t.Parallel()

	if domain.FromEntCharacterConfig(nil) != nil {
		t.Fatalf("expected nil when input is nil")
	}
}
