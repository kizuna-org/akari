package domain_test

import (
	"testing"
	"time"

	"github.com/kizuna-org/akari/gen/ent"
	"github.com/kizuna-org/akari/pkg/database/domain"
)

type domainDiscordUserWant struct {
	ID       string
	Username string
}

func TestFromEntDiscordUser_NilAndFields(t *testing.T) {
	t.Parallel()

	now := time.Now()

	valid := &ent.DiscordUser{ID: "user-id", Username: "user-name", Bot: false, CreatedAt: now, UpdatedAt: now}

	tests := []struct {
		name  string
		input *ent.DiscordUser
		want  *domainDiscordUserWant
	}{
		{name: "nil input", input: nil, want: nil},
		{name: "valid input", input: valid, want: &domainDiscordUserWant{ID: valid.ID, Username: valid.Username}},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := domain.FromEntDiscordUser(testCase.input)
			if testCase.want == nil {
				if got != nil {
					t.Fatalf("expected nil result for nil input, got: %+v", got)
				}

				return
			}

			if got == nil {
				t.Fatalf("expected non-nil result")
			}

			if got.ID != testCase.want.ID {
				t.Fatalf("ID mismatch: got=%v want=%v", got.ID, testCase.want.ID)
			}

			if got.Username != testCase.want.Username {
				t.Fatalf("Username mismatch: got=%v want=%v", got.Username, testCase.want.Username)
			}
		})
	}
}
