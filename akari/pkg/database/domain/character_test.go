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

	valid := &ent.Character{
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

	missingConfig := &ent.Character{
		ID:        2,
		Name:      "no-config",
		CreatedAt: now,
		UpdatedAt: now,
		Edges: ent.CharacterEdges{
			Config:        nil,
			SystemPrompts: []*ent.SystemPrompt{{ID: 1}},
		},
	}

	missingPrompts := &ent.Character{
		ID:        3,
		Name:      "no-prompts",
		CreatedAt: now,
		UpdatedAt: now,
		Edges: ent.CharacterEdges{
			Config: &ent.CharacterConfig{ID: 1},
		},
	}

	tests := []struct {
		name    string
		input   *ent.Character
		wantErr bool
	}{
		{name: "valid character", input: valid, wantErr: false},
		{name: "nil input", input: nil, wantErr: true},
		{name: "missing config edge", input: missingConfig, wantErr: true},
		{name: "missing system prompts edge", input: missingPrompts, wantErr: true},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, err := domain.FromEntCharacter(testCase.input)
			if (err != nil) != testCase.wantErr {
				t.Fatalf("unexpected error state: %v", err)
			}

			if testCase.wantErr {
				if got != nil {
					t.Fatalf("expected nil on error, got: %+v", got)
				}

				return
			}

			if got == nil {
				t.Fatalf("expected non-nil result")
			}

			if got.ID != testCase.input.ID || got.Name != testCase.input.Name {
				t.Fatalf("ID/Name mismatch: got id=%d name=%s", got.ID, got.Name)
			}

			checkCharacterConversion(t, got, testCase.input)
		})
	}
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
