package repository_test

import (
	"testing"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/kizuna-org/akari/pkg/database/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepository_CreateConversationGroup_Integration(t *testing.T) {
	t.Parallel()

	repo, entClient := setupTestDB(t)
	ctx := t.Context()

	tests := []struct {
		name     string
		setup    func() int
		validate func(t *testing.T, got *domain.ConversationGroup, expectedCharacterID int)
	}{
		{
			name: "success",
			setup: func() int {
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
			validate: func(t *testing.T, got *domain.ConversationGroup, expectedCharacterID int) {
				t.Helper()
				assert.Positive(t, got.ID)
				assert.Equal(t, expectedCharacterID, got.CharacterID)
				assert.NotZero(t, got.CreatedAt)
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			characterID := testCase.setup()

			got, err := repo.CreateConversationGroup(ctx, characterID)
			require.NoError(t, err)

			if testCase.validate != nil {
				testCase.validate(t, got, characterID)
			}
		})
	}
}

func TestRepository_GetConversationGroupByID_Integration(t *testing.T) {
	t.Parallel()

	repo, entClient := setupTestDB(t)
	ctx := t.Context()

	tests := []struct {
		name     string
		setup    func() int
		wantErr  bool
		errMsg   string
		validate func(t *testing.T, got *domain.ConversationGroup, expectedID int)
	}{
		{
			name: "not found",
			setup: func() int {
				return 99999
			},
			wantErr: true,
			errMsg:  "failed to get conversation group by id",
		},
		{
			name: "success",
			setup: func() int {
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

				created, err := repo.CreateConversationGroup(ctx, character.ID)
				require.NoError(t, err)

				return created.ID
			},
			wantErr: false,
			validate: func(t *testing.T, got *domain.ConversationGroup, expectedID int) {
				t.Helper()
				assert.Equal(t, expectedID, got.ID)
				assert.Positive(t, got.CharacterID)
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			groupID := testCase.setup()

			got, err := repo.GetConversationGroupByID(ctx, groupID)

			if testCase.wantErr {
				require.Error(t, err)

				if testCase.errMsg != "" {
					assert.Contains(t, err.Error(), testCase.errMsg)
				}
			} else {
				require.NoError(t, err)

				if testCase.validate != nil {
					testCase.validate(t, got, groupID)
				}
			}
		})
	}
}

func TestRepository_ListConversationGroups_Integration(t *testing.T) {
	t.Parallel()

	repo, entClient := setupTestDB(t)
	ctx := t.Context()

	tests := []struct {
		name     string
		setup    func() []int
		validate func(t *testing.T, got []*domain.ConversationGroup, expectedIDs []int)
	}{
		{
			name: "with multiple groups",
			setup: func() []int {
				var groupIDs []int

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

					group, err := repo.CreateConversationGroup(ctx, character.ID)
					require.NoError(t, err)

					groupIDs = append(groupIDs, group.ID)
				}

				return groupIDs
			},
			validate: func(t *testing.T, got []*domain.ConversationGroup, expectedIDs []int) {
				t.Helper()
				found := make(map[int]bool)
				for _, id := range expectedIDs {
					found[id] = false
				}

				for _, g := range got {
					if _, exists := found[g.ID]; exists {
						found[g.ID] = true
					}
				}

				for id, wasFound := range found {
					assert.True(t, wasFound, "group %d should be in the list", id)
				}
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			expectedIDs := testCase.setup()

			got, err := repo.ListConversationGroups(ctx)
			require.NoError(t, err)

			if testCase.validate != nil {
				testCase.validate(t, got, expectedIDs)
			}
		})
	}
}

func TestRepository_DeleteConversationGroup_Integration(t *testing.T) {
	t.Parallel()

	repo, entClient := setupTestDB(t)
	ctx := t.Context()

	tests := []struct {
		name    string
		setup   func() int
		wantErr bool
		errMsg  string
	}{
		{
			name: "success",
			setup: func() int {
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

				created, err := repo.CreateConversationGroup(ctx, character.ID)
				require.NoError(t, err)

				return created.ID
			},
			wantErr: false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			groupID := testCase.setup()

			err := repo.DeleteConversationGroup(ctx, groupID)

			if testCase.wantErr {
				require.Error(t, err)

				if testCase.errMsg != "" {
					assert.Contains(t, err.Error(), testCase.errMsg)
				}
			} else {
				require.NoError(t, err)

				_, err = repo.GetConversationGroupByID(ctx, groupID)
				require.Error(t, err)
				assert.Contains(t, err.Error(), "failed to get conversation group by id")
			}
		})
	}
}
