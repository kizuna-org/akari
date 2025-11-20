package domain_test

import (
	"testing"
	"time"

	"github.com/kizuna-org/akari/gen/ent"
	"github.com/kizuna-org/akari/pkg/database/domain"
)

func TestFromEntCharacter_ConfigConversion(t *testing.T) {
	t.Parallel()

	now := time.Now()
	nameRegex := "^name$"

	validConfig := &ent.CharacterConfig{
		NameRegexp:          &nameRegex,
		DefaultSystemPrompt: "default-systemPrompt",
	}

	cases := []struct {
		name          string
		entConfig     *ent.CharacterConfig
		wantErrConfig bool
	}{
		{name: "valid config", entConfig: validConfig, wantErrConfig: false},
		{name: "nil config", entConfig: nil, wantErrConfig: true},
		{name: "config with id only", entConfig: &ent.CharacterConfig{ID: 1}, wantErrConfig: false},
		{name: "missing prompts edge", entConfig: &ent.CharacterConfig{ID: 1}, wantErrConfig: false},
	}

	for _, c := range cases {
		testCase := c
		t.Run("config:"+testCase.name, func(t *testing.T) {
			t.Parallel()

			tempChar := &ent.Character{
				ID:        999,
				Name:      "cfg-test",
				CreatedAt: now,
				UpdatedAt: now,
				Edges: ent.CharacterEdges{
					Config:        testCase.entConfig,
					SystemPrompts: []*ent.SystemPrompt{{ID: 1}},
				},
			}

			got, err := domain.FromEntCharacter(tempChar)
			if (err != nil) != testCase.wantErrConfig {
				t.Fatalf("config conversion unexpected error state: %v", err)
			}

			if testCase.wantErrConfig {
				if got != nil {
					t.Fatalf("expected nil config-conversion result on error, got: %+v", got)
				}

				return
			}

			if got == nil {
				t.Fatalf("expected non-nil result from wrapped character conversion")
			}
		})
	}
}

func TestFromEntCharacter_CharacterConversion(t *testing.T) {
	t.Parallel()

	now := time.Now()
	nameRegex := "^name$"

	validConfig := &ent.CharacterConfig{
		NameRegexp:          &nameRegex,
		DefaultSystemPrompt: "default-systemPrompt",
	}

	validCharacter := &ent.Character{
		ID:        1,
		Name:      "character-name",
		CreatedAt: now,
		UpdatedAt: now,
		Edges: ent.CharacterEdges{
			Config: validConfig,
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

	cases := []struct {
		name        string
		entChar     *ent.Character
		wantErrChar bool
	}{
		{name: "valid character", entChar: validCharacter, wantErrChar: false},
		{name: "nil character", entChar: nil, wantErrChar: true},
		{name: "missing config edge", entChar: missingConfig, wantErrChar: true},
		{name: "missing prompts", entChar: missingPrompts, wantErrChar: true},
	}

	for _, c := range cases {
		testCase := c
		t.Run("char:"+testCase.name, func(t *testing.T) {
			t.Parallel()

			got, err := domain.FromEntCharacter(testCase.entChar)
			if (err != nil) != testCase.wantErrChar {
				t.Fatalf("FromEntCharacter unexpected error state: %v", err)
			}

			if testCase.wantErrChar {
				if got != nil {
					t.Fatalf("expected nil character on error, got: %+v", got)
				}

				return
			}

			if got == nil {
				t.Fatalf("expected non-nil character result")
			}

			if got.ID != testCase.entChar.ID || got.Name != testCase.entChar.Name {
				t.Fatalf("ID/Name mismatch: got id=%d name=%s", got.ID, got.Name)
			}

			checkCharacterConversion(t, got, testCase.entChar)
		})
	}
}

func checkCharacterConversion(t *testing.T, character *domain.Character, entCharacter *ent.Character) {
	t.Helper()

	if entCharacter.Edges.Config == nil {
		t.Fatalf("Config missing in ent fixture: %+v", entCharacter.Edges.Config)
	}

	if character.Config == nil {
		t.Fatalf("Config missing in domain.Character")
	}

	if character.Config.DefaultSystemPrompt != entCharacter.Edges.Config.DefaultSystemPrompt {
		t.Fatalf("Config DefaultSystemPrompt mismatch")
	}

	if character.Config.NameRegExp == nil || entCharacter.Edges.Config.NameRegexp == nil {
		t.Fatalf("unexpected nil NameRegExp/NameRegexp pointers")
	}

	if character.Config.NameRegExp != entCharacter.Edges.Config.NameRegexp {
		t.Fatalf("NameRegExp/NameRegexp pointer mismatch")
	}

	for i, systemPrompt := range entCharacter.Edges.SystemPrompts {
		if character.SystemPromptIDs[i] != systemPrompt.ID {
			t.Fatalf("SystemPrompts ID mismatch at index %d", i)
		}
	}
}
