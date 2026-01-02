package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/kizuna-org/akari/pkg/database/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepository_CreateConversationGroup_Integration(t *testing.T) {
	t.Parallel()

	_, repo, entClient := setupTestDB(t)
	ctx := context.Background()

	tests := []struct {
		name     string
		setup    func() int
		validate func(t *testing.T, got *domain.ConversationGroup, expectedCharacterID int)
	}{
		{
			name: "success",
			setup: func() int {
				gofakeit.Seed(time.Now().UnixNano())

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
				assert.Greater(t, got.ID, 0)
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

	_, repo, entClient := setupTestDB(t)
	ctx := context.Background()

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
				gofakeit.Seed(time.Now().UnixNano())

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
				assert.Equal(t, expectedID, got.ID)
				assert.Greater(t, got.CharacterID, 0)
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

	_, repo, entClient := setupTestDB(t)
	ctx := context.Background()

	tests := []struct {
		name     string
		setup    func() []int
		validate func(t *testing.T, got []*domain.ConversationGroup, expectedIDs []int)
	}{
		{
			name: "with multiple groups",
			setup: func() []int {
				gofakeit.Seed(time.Now().UnixNano())

				var groupIDs []int

				for i := 0; i < 2; i++ {
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
				assert.GreaterOrEqual(t, len(got), len(expectedIDs))

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

	_, repo, entClient := setupTestDB(t)
	ctx := context.Background()

	tests := []struct {
		name    string
		setup   func() int
		wantErr bool
		errMsg  string
	}{
		{
			name: "success",
			setup: func() int {
				gofakeit.Seed(time.Now().UnixNano())

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
