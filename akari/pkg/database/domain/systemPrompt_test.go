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

	systemPrompt := domain.FromEntSystemPrompt(entSystemPrompt)
	if systemPrompt == nil {
		t.Fatalf("expected non-nil system prompt")
	}

	if systemPrompt.ID != entSystemPrompt.ID {
		t.Fatalf("ID mismatch: got=%d want=%d", systemPrompt.ID, entSystemPrompt.ID)
	}

	if systemPrompt.Title != entSystemPrompt.Title {
		t.Fatalf("Title mismatch: got=%q want=%q", systemPrompt.Title, entSystemPrompt.Title)
	}

	if systemPrompt.Purpose != string(entSystemPrompt.Purpose) {
		t.Fatalf("Purpose mismatch: got=%q want=%q", systemPrompt.Purpose, entSystemPrompt.Purpose)
	}

	if systemPrompt.Prompt != entSystemPrompt.Prompt {
		t.Fatalf("Prompt mismatch: got=%q want=%q", systemPrompt.Prompt, entSystemPrompt.Prompt)
	}

	if !systemPrompt.CreatedAt.Equal(entSystemPrompt.CreatedAt) {
		t.Fatalf("CreatedAt mismatch: got=%v want=%v", systemPrompt.CreatedAt, entSystemPrompt.CreatedAt)
	}

	if !systemPrompt.UpdatedAt.Equal(entSystemPrompt.UpdatedAt) {
		t.Fatalf("UpdatedAt mismatch: got=%v want=%v", systemPrompt.UpdatedAt, entSystemPrompt.UpdatedAt)
	}
}

func TestFromEntSystemPrompt_Nil(t *testing.T) {
	t.Parallel()

	if domain.FromEntSystemPrompt(nil) != nil {
		t.Fatalf("expected nil when input is nil")
	}
}
