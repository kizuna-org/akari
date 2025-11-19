package domain_test

import (
	"testing"

	"github.com/kizuna-org/akari/gen/ent"
	"github.com/kizuna-org/akari/pkg/database/domain"
)

func TestFromEntCharacterConfig_Converts(t *testing.T) {
	t.Parallel()

	nameRegex := "^name$"
	valid := &ent.CharacterConfig{
		NameRegexp:          &nameRegex,
		DefaultSystemPrompt: "default-systemPrompt",
	}

	tests := []struct {
		name    string
		ent     *ent.CharacterConfig
		wantErr bool
	}{
		{name: "valid config", ent: valid, wantErr: false},
		{name: "nil config", ent: nil, wantErr: true},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, err := domain.FromEntCharacterConfig(testCase.ent)
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

			if got.DefaultSystemPrompt != testCase.ent.DefaultSystemPrompt {
				t.Fatalf("DefaultSystemPrompt mismatch: got=%q want=%q", got.DefaultSystemPrompt, testCase.ent.DefaultSystemPrompt)
			}

			if got.NameRegexp == nil || testCase.ent.NameRegexp == nil {
				t.Fatalf("unexpected nil NameRegexp pointers")
			}

			if got.NameRegexp != testCase.ent.NameRegexp {
				t.Fatalf("NameRegexp pointer mismatch")
			}
		})
	}
}
