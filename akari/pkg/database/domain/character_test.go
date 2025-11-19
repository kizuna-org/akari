package domain_test

import (
	"testing"
	"time"

	"github.com/kizuna-org/akari/gen/ent"
	"github.com/kizuna-org/akari/pkg/database/domain"
)

func TestFromEntCharacter(t *testing.T) {
	t.Parallel()

	now := time.Now()
	nameRegex := "^name$"
	entCharacter := &ent.Character{
		ID:        1,
		Name:      "character-name",
		CreatedAt: now,
		UpdatedAt: now,
		Edges: ent.CharacterEdges{
			Config: &ent.CharacterConfig{NameRegexp: &nameRegex, DefaultSystemPrompt: "default-systemPrompt"},
			SystemPrompts: []*ent.SystemPrompt{
				{ID: 1, Prompt: "systemPrompt"},
			},
		},
	}

	character := domain.FromEntCharacter(entCharacter)
	if character == nil {
		t.Fatalf("expected non-nil domain character")
	}

	if character.ID != entCharacter.ID || character.Name != entCharacter.Name {
		t.Fatalf("ID/Name mismatch: got id=%d name=%s", character.ID, character.Name)
	}

	checkCharacterConversion(t, character, entCharacter)
}

func checkCharacterConversion(t *testing.T, character *domain.Character, entCharacter *ent.Character) {
	t.Helper()

	if character.Config == nil {
		t.Fatalf("Config missing in domain character: %+v", character.Config)
	}

	if entCharacter.Edges.Config == nil {
		t.Fatalf("Config missing in ent fixture: %+v", entCharacter.Edges.Config)
	}

	if character.Config.DefaultSystemPrompt != entCharacter.Edges.Config.DefaultSystemPrompt {
		t.Fatalf("Config.DefaultSystemPrompt mismatch")
	}

	if character.Config.NameRegexp == nil {
		t.Fatalf("Config.NameRegexp missing in domain")
	}

	if entCharacter.Edges.Config.NameRegexp == nil {
		t.Fatalf("Config.NameRegexp missing in ent fixture")
	}

	if character.Config.NameRegexp != entCharacter.Edges.Config.NameRegexp {
		t.Fatalf("Config.NameRegexp mismatch")
	}

	if len(character.SystemPrompts) != len(entCharacter.Edges.SystemPrompts) {
		t.Fatalf("SystemPrompts length mismatch")
	}

	if character.SystemPrompts[0].ID != entCharacter.Edges.SystemPrompts[0].ID {
		t.Fatalf("SystemPrompts ID mismatch")
	}

	if character.SystemPrompts[0].Prompt != entCharacter.Edges.SystemPrompts[0].Prompt {
		t.Fatalf("SystemPrompts Prompt mismatch")
	}
}

func TestFromEntCharacter_Nil(t *testing.T) {
	t.Parallel()

	if domain.FromEntCharacter(nil) != nil {
		t.Fatalf("expected nil when input is nil")
	}
}
