package domain_test

import (
	"testing"
	"time"

	"github.com/kizuna-org/akari/gen/ent"
	"github.com/kizuna-org/akari/pkg/database/domain"
)

func TestFromEntConversationGroup_TableDriven(t *testing.T) {
	t.Parallel()

	now := time.Now()
	entCharacter := &ent.Character{ID: 1, Name: "name", CreatedAt: now, UpdatedAt: now}

	withChar := &ent.ConversationGroup{
		ID:        3,
		CreatedAt: now,
		Edges:     ent.ConversationGroupEdges{Character: entCharacter},
	}

	withoutChar := &ent.ConversationGroup{
		ID:        4,
		CreatedAt: now,
		Edges:     ent.ConversationGroupEdges{Character: nil},
	}

	tests := []struct {
		name    string
		input   *ent.ConversationGroup
		wantErr bool
	}{
		{name: "with character edge", input: withChar, wantErr: false},
		{name: "missing character edge", input: withoutChar, wantErr: true},
		{name: "nil input", input: nil, wantErr: true},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, err := domain.FromEntConversationGroup(testCase.input)
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

			if got.ID != testCase.input.ID {
				t.Fatalf("ID mismatch: got=%d want=%d", got.ID, testCase.input.ID)
			}

			if got.CharacterID != testCase.input.Edges.Character.ID {
				t.Fatalf("CharacterID mismatch: got=%d want=%d", got.CharacterID, testCase.input.Edges.Character.ID)
			}
		})
	}
}
