package domain_test

import (
	"testing"
	"time"

	"github.com/kizuna-org/akari/gen/ent"
	"github.com/kizuna-org/akari/pkg/database/domain"
)

func TestFromEntAkariUser_Converts(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := []struct {
		name    string
		ent     *ent.AkariUser
		wantErr bool
	}{
		{
			name:    "valid ent user",
			ent:     &ent.AkariUser{ID: 1, CreatedAt: now, UpdatedAt: now},
			wantErr: false,
		},
		{
			name:    "nil ent user",
			ent:     nil,
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, err := domain.FromEntAkariUser(testCase.ent)
			if (err != nil) != testCase.wantErr {
				t.Fatalf("unexpected error state: %v", err)
			}

			if testCase.wantErr {
				if got != nil {
					t.Fatalf("expected nil result when error, got non-nil")
				}

				return
			}

			if got == nil {
				t.Fatalf("expected non-nil result")
			}

			if got.ID != testCase.ent.ID {
				t.Fatalf("ID mismatch: got=%d want=%d", got.ID, testCase.ent.ID)
			}

			if !got.CreatedAt.Equal(testCase.ent.CreatedAt) || !got.UpdatedAt.Equal(testCase.ent.UpdatedAt) {
				ca, ua, ea, eb := got.CreatedAt, got.UpdatedAt, testCase.ent.CreatedAt, testCase.ent.UpdatedAt
				t.Fatalf("timestamps mismatch: got=%v/%v want=%v/%v", ca, ua, ea, eb)
			}
		})
	}
}
