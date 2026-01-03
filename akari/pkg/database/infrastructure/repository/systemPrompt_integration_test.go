package repository_test

import (
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/kizuna-org/akari/pkg/database/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepository_GetSystemPromptByID_Integration(t *testing.T) {
	t.Parallel()

	repo, entClient := setupTestDB(t)
	ctx := t.Context()

	tests := []struct {
		name     string
		setup    func() int
		wantErr  bool
		errMsg   string
		validate func(t *testing.T, got *domain.SystemPrompt, expectedID int)
	}{
		{
			name: "not found",
			setup: func() int {
				return 99999
			},
			wantErr: true,
			errMsg:  "failed to get system prompt by id",
		},
		{
			name: "success",
			setup: func() int {
				_ = gofakeit.Seed(time.Now().UnixNano())

				systemPrompt, err := entClient.SystemPrompt.Create().
					SetTitle(gofakeit.Word()).
					SetPurpose("text_chat").
					SetPrompt(gofakeit.Paragraph(3, 5, 10, "\n")).
					Save(ctx)
				require.NoError(t, err)

				return systemPrompt.ID
			},
			wantErr: false,
			validate: func(t *testing.T, got *domain.SystemPrompt, expectedID int) {
				t.Helper()
				assert.Equal(t, expectedID, got.ID)
				assert.NotEmpty(t, got.Title)
				assert.NotEmpty(t, got.Purpose)
				assert.NotEmpty(t, got.Prompt)
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			systemPromptID := testCase.setup()

			got, err := repo.GetSystemPromptByID(ctx, systemPromptID)

			if testCase.wantErr {
				require.Error(t, err)

				if testCase.errMsg != "" {
					assert.Contains(t, err.Error(), testCase.errMsg)
				}
			} else {
				require.NoError(t, err)

				if testCase.validate != nil {
					testCase.validate(t, got, systemPromptID)
				}
			}
		})
	}
}
