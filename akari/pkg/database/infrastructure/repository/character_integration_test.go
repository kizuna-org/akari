package repository_test

import (
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/kizuna-org/akari/pkg/database/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepository_GetCharacterByID_Integration(t *testing.T) {
	t.Parallel()

	repo, entClient := setupTestDB(t)
	ctx := t.Context()

	tests := []struct {
		name     string
		setup    func() int
		wantErr  bool
		errMsg   string
		validate func(t *testing.T, got *domain.Character)
	}{
		{
			name: "not found",
			setup: func() int {
				return 99999
			},
			wantErr: true,
			errMsg:  "failed to get character",
		},
		{
			name: "success",
			setup: func() int {
				_ = gofakeit.Seed(time.Now().UnixNano())

				config, err := entClient.CharacterConfig.Create().
					SetDefaultSystemPrompt(gofakeit.Sentence(10)).
					Save(ctx)
				require.NoError(t, err)

				systemPrompt, err := entClient.SystemPrompt.Create().
					SetTitle(gofakeit.Word()).
					SetPurpose("text_chat").
					SetPrompt(gofakeit.Paragraph(3, 5, 10, "\n")).
					Save(ctx)
				require.NoError(t, err)

				character, err := entClient.Character.Create().
					SetName(gofakeit.Name()).
					SetConfig(config).
					AddSystemPrompts(systemPrompt).
					Save(ctx)
				require.NoError(t, err)

				return character.ID
			},
			wantErr: false,
			validate: func(t *testing.T, got *domain.Character) {
				t.Helper()
				assert.Positive(t, got.ID)
				assert.NotEmpty(t, got.Name)
				assert.NotNil(t, got.Config)
				assert.GreaterOrEqual(t, len(got.SystemPromptIDs), 1)
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			characterID := testCase.setup()

			got, err := repo.GetCharacterByID(ctx, characterID)

			if testCase.wantErr {
				require.Error(t, err)

				if testCase.errMsg != "" {
					assert.Contains(t, err.Error(), testCase.errMsg)
				}
			} else {
				require.NoError(t, err)

				if testCase.validate != nil {
					testCase.validate(t, got)
				}
			}
		})
	}
}

func TestRepository_ListCharacters_Integration(t *testing.T) {
	t.Parallel()

	repo, entClient := setupTestDB(t)
	ctx := t.Context()

	tests := []struct {
		name     string
		setup    func() []int
		validate func(t *testing.T, got []*domain.Character, expectedIDs []int)
	}{
		{
			name: "empty",
			setup: func() []int {
				return []int{}
			},
			validate: func(t *testing.T, got []*domain.Character, expectedIDs []int) {
				t.Helper()
				assert.Empty(t, got)
			},
		},
		{
			name: "with data",
			setup: func() []int {
				_ = gofakeit.Seed(time.Now().UnixNano())

				var characterIDs []int

				for range 2 {
					config, err := entClient.CharacterConfig.Create().
						SetDefaultSystemPrompt(gofakeit.Sentence(10)).
						Save(ctx)
					require.NoError(t, err)

					systemPrompt, err := entClient.SystemPrompt.Create().
						SetTitle(gofakeit.Word()).
						SetPurpose("text_chat").
						SetPrompt(gofakeit.Paragraph(3, 5, 10, "\n")).
						Save(ctx)
					require.NoError(t, err)

					character, err := entClient.Character.Create().
						SetName(gofakeit.Name()).
						SetConfig(config).
						AddSystemPrompts(systemPrompt).
						Save(ctx)
					require.NoError(t, err)

					characterIDs = append(characterIDs, character.ID)
				}

				return characterIDs
			},
			validate: func(t *testing.T, got []*domain.Character, expectedIDs []int) {
				t.Helper()
				assert.GreaterOrEqual(t, len(got), len(expectedIDs))

				found := make(map[int]bool)

				for _, c := range got {
					if _, exists := found[c.ID]; exists {
						found[c.ID] = true
					}
				}

				for id, wasFound := range found {
					assert.True(t, wasFound, "character %d should be in the list", id)
				}
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			expectedIDs := testCase.setup()

			got, err := repo.ListCharacters(ctx)
			require.NoError(t, err)

			if testCase.validate != nil {
				testCase.validate(t, got, expectedIDs)
			}
		})
	}
}
