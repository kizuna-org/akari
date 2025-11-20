package domain_test

import (
	"testing"
	"time"

	"github.com/kizuna-org/akari/gen/ent"
	"github.com/kizuna-org/akari/gen/ent/systemprompt"
	"github.com/kizuna-org/akari/pkg/database/domain"
)

func TestFromEntSystemPrompt_Converts(t *testing.T) {
	t.Parallel()

	now := time.Now()
	entSystemPrompt := &ent.SystemPrompt{
		ID:        1,
		Title:     "systemPrompt-title",
		Purpose:   systemprompt.PurposeTextChat,
		Prompt:    "systemPrompt",
		CreatedAt: now,
		UpdatedAt: now,
	}

	tests := []struct {
		name    string
		input   *ent.SystemPrompt
		wantErr bool
	}{
		{name: "valid system prompt", input: entSystemPrompt, wantErr: false},
		{name: "nil input", input: nil, wantErr: false}, // FromEntSystemPrompt returns nil rather than error
	}

	runSystemPromptCases(t, tests)
}

func runSystemPromptCases(t *testing.T, tests []struct {
	name    string
	input   *ent.SystemPrompt
	wantErr bool
}) {
	t.Helper()

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := domain.FromEntSystemPrompt(testCase.input)
			if testCase.input == nil {
				if got != nil {
					t.Fatalf("expected nil for nil input")
				}

				return
			}

			validateSystemPromptResult(t, got, testCase.input)
		})
	}
}

func validateSystemPromptResult(t *testing.T, got *domain.SystemPrompt, want *ent.SystemPrompt) {
	t.Helper()

	if got.ID != want.ID {
		t.Fatalf("ID mismatch: got=%d want=%d", got.ID, want.ID)
	}

	if got.Title != want.Title {
		t.Fatalf("Title mismatch: got=%q want=%q", got.Title, want.Title)
	}

	if got.Purpose != string(want.Purpose) {
		t.Fatalf("Purpose mismatch: got=%q want=%q", got.Purpose, want.Purpose)
	}

	if got.Prompt != want.Prompt {
		t.Fatalf("Prompt mismatch: got=%q want=%q", got.Prompt, want.Prompt)
	}

	if !got.CreatedAt.Equal(want.CreatedAt) || !got.UpdatedAt.Equal(want.UpdatedAt) {
		t.Fatalf("timestamps mismatch")
	}
}
