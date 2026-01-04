package domain_test

import (
	"testing"
	"time"

	"github.com/kizuna-org/akari/gen/ent"
	"github.com/kizuna-org/akari/pkg/database/domain"
)

type domainDiscordUserWant struct {
	ID          string
	Username    string
	AkariUserID *int
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
		{
			name:  "valid input",
			input: valid,
			want:  &domainDiscordUserWant{ID: valid.ID, Username: valid.Username, AkariUserID: nil},
		},
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

			if got.AkariUserID != testCase.want.AkariUserID {
				t.Fatalf("AkariUserID mismatch: got=%v want=%v", got.AkariUserID, testCase.want.AkariUserID)
			}
		})
	}
}

func TestFromEntDiscordUser_WithAkariUserEdge(t *testing.T) {
	t.Parallel()

	now := time.Now()
	akariUserID := 123
	akariUser := &ent.AkariUser{ID: akariUserID, CreatedAt: now, UpdatedAt: now}

	withAkariUser := &ent.DiscordUser{
		ID:        "user-id",
		Username:  "user-name",
		Bot:       false,
		CreatedAt: now,
		UpdatedAt: now,
		Edges: ent.DiscordUserEdges{
			AkariUser: akariUser,
		},
	}

	withoutAkariUser := &ent.DiscordUser{
		ID:        "user-id-2",
		Username:  "user-name-2",
		Bot:       true,
		CreatedAt: now,
		UpdatedAt: now,
		Edges: ent.DiscordUserEdges{
			AkariUser: nil,
		},
	}

	tests := []struct {
		name          string
		input         *ent.DiscordUser
		wantAkariUser *int
	}{
		{name: "with akari user edge", input: withAkariUser, wantAkariUser: &akariUserID},
		{name: "without akari user edge", input: withoutAkariUser, wantAkariUser: nil},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := domain.FromEntDiscordUser(testCase.input)
			if got == nil {
				t.Fatalf("expected non-nil result")
			}

			if testCase.wantAkariUser == nil {
				if got.AkariUserID != nil {
					t.Fatalf("expected nil AkariUserID, got: %v", got.AkariUserID)
				}
			} else {
				if got.AkariUserID == nil {
					t.Fatalf("expected non-nil AkariUserID, got: nil")
				}

				if *got.AkariUserID != *testCase.wantAkariUser {
					t.Fatalf("AkariUserID mismatch: got=%d want=%d", *got.AkariUserID, *testCase.wantAkariUser)
				}
			}
		})
	}
}
