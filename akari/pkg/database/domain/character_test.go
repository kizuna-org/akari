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

	character, err := domain.FromEntCharacter(entCharacter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

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

	if entCharacter.Edges.Config == nil {
		t.Fatalf("Config missing in ent fixture: %+v", entCharacter.Edges.Config)
	}

	if character.ConfigID != entCharacter.Edges.Config.ID {
		t.Fatalf("Config mismatch")
	}

	for i, systemPrompt := range entCharacter.Edges.SystemPrompts {
		if character.SystemPromptIDs[i] != systemPrompt.ID {
			t.Fatalf("SystemPrompts ID mismatch at index %d", i)
		}
	}
}

func TestFromEntCharacter_Nil(t *testing.T) {
	t.Parallel()

	character, err := domain.FromEntCharacter(nil)
	if err == nil {
		t.Fatalf("expected error when input is nil")
	}

	if character != nil {
		t.Fatalf("expected nil character when input is nil")
	}
}
