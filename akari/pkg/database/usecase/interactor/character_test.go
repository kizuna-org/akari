package interactor_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kizuna-org/akari/gen/ent"
	"github.com/kizuna-org/akari/pkg/database/domain/mock"
	"github.com/kizuna-org/akari/pkg/database/usecase/interactor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestNewCharacterInteractor(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock.NewMockCharacterRepository(ctrl)
	interactor := interactor.NewCharacterInteractor(mockRepo)

	assert.NotNil(t, interactor)
}

func TestCharacterInteractor_CreateCharacter(t *testing.T) {
	t.Parallel()

	name := "Test Character"

	tests := []struct {
		name      string
		mockSetup func(*mock.MockCharacterRepository, context.Context)
		wantErr   bool
	}{
		{
			name: "success",
			mockSetup: func(m *mock.MockCharacterRepository, ctx context.Context) {
				m.EXPECT().
					CreateCharacter(ctx, name).
					Return(&ent.Character{
						ID:        1,
						Name:      name,
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					}, nil)
			},
			wantErr: false,
		},
		{
			name: "failure - database error",
			mockSetup: func(m *mock.MockCharacterRepository, ctx context.Context) {
				m.EXPECT().
					CreateCharacter(ctx, name).
					Return(nil, errors.New("database error"))
			},
			wantErr: true,
		},
		{
			name: "failure - duplicate name",
			mockSetup: func(m *mock.MockCharacterRepository, ctx context.Context) {
				m.EXPECT().
					CreateCharacter(ctx, name).
					Return(nil, errors.New("unique constraint violation"))
			},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := mock.NewMockCharacterRepository(ctrl)
			interactor := interactor.NewCharacterInteractor(mockRepo)

			ctx := t.Context()
			testCase.mockSetup(mockRepo, ctx)

			result, err := interactor.CreateCharacter(ctx, name)

			if testCase.wantErr {
				require.Error(t, err)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, name, result.Name)
			}
		})
	}
}

func TestCharacterInteractor_GetCharacterByID(t *testing.T) {
	t.Parallel()

	characterID := 1

	tests := []struct {
		name      string
		mockSetup func(*mock.MockCharacterRepository, context.Context)
		wantErr   bool
	}{
		{
			name: "success",
			mockSetup: func(m *mock.MockCharacterRepository, ctx context.Context) {
				m.EXPECT().
					GetCharacterByID(ctx, characterID).
					Return(&ent.Character{
						ID:        characterID,
						Name:      "Test Character",
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					}, nil)
			},
			wantErr: false,
		},
		{
			name: "not found",
			mockSetup: func(m *mock.MockCharacterRepository, ctx context.Context) {
				m.EXPECT().
					GetCharacterByID(ctx, characterID).
					Return(nil, errors.New("not found"))
			},
			wantErr: true,
		},
		{
			name: "database error",
			mockSetup: func(m *mock.MockCharacterRepository, ctx context.Context) {
				m.EXPECT().
					GetCharacterByID(ctx, characterID).
					Return(nil, errors.New("database connection error"))
			},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := mock.NewMockCharacterRepository(ctrl)
			interactor := interactor.NewCharacterInteractor(mockRepo)

			ctx := t.Context()
			testCase.mockSetup(mockRepo, ctx)

			result, err := interactor.GetCharacterByID(ctx, characterID)

			if testCase.wantErr {
				require.Error(t, err)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, characterID, result.ID)
			}
		})
	}
}

func TestCharacterInteractor_GetCharacterWithSystemPromptByID(t *testing.T) {
	t.Parallel()

	characterID := 1

	tests := []struct {
		name      string
		mockSetup func(*mock.MockCharacterRepository, context.Context)
		wantErr   bool
	}{
		{
			name: "success",
			mockSetup: func(m *mock.MockCharacterRepository, ctx context.Context) {
				m.EXPECT().
					GetCharacterWithSystemPromptByID(ctx, characterID).
					Return(&ent.Character{
						ID:        characterID,
						Name:      "Test Character",
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
						Edges: ent.CharacterEdges{
							SystemPrompts: []*ent.SystemPrompt{
								{
									ID:        1,
									Title:     "Test Prompt",
									Prompt:    "System prompt content",
									CreatedAt: time.Now(),
									UpdatedAt: time.Now(),
								},
							},
						},
					}, nil)
			},
			wantErr: false,
		},
		{
			name: "not found",
			mockSetup: func(m *mock.MockCharacterRepository, ctx context.Context) {
				m.EXPECT().
					GetCharacterWithSystemPromptByID(ctx, characterID).
					Return(nil, errors.New("not found"))
			},
			wantErr: true,
		},
		{
			name: "database error",
			mockSetup: func(m *mock.MockCharacterRepository, ctx context.Context) {
				m.EXPECT().
					GetCharacterWithSystemPromptByID(ctx, characterID).
					Return(nil, errors.New("failed to load edges"))
			},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := mock.NewMockCharacterRepository(ctrl)
			interactor := interactor.NewCharacterInteractor(mockRepo)

			ctx := t.Context()
			testCase.mockSetup(mockRepo, ctx)

			result, err := interactor.GetCharacterWithSystemPromptByID(ctx, characterID)

			if testCase.wantErr {
				require.Error(t, err)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, characterID, result.ID)
				assert.NotEmpty(t, result.Edges.SystemPrompts)
			}
		})
	}
}

func TestCharacterInteractor_ListCharacters(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		mockSetup func(*mock.MockCharacterRepository, context.Context)
		wantErr   bool
		wantCount int
	}{
		{
			name: "success - multiple characters",
			mockSetup: func(m *mock.MockCharacterRepository, ctx context.Context) {
				m.EXPECT().
					ListCharacters(ctx).
					Return([]*ent.Character{
						{
							ID:        1,
							Name:      "Character 1",
							CreatedAt: time.Now(),
							UpdatedAt: time.Now(),
						},
						{
							ID:        2,
							Name:      "Character 2",
							CreatedAt: time.Now(),
							UpdatedAt: time.Now(),
						},
						{
							ID:        3,
							Name:      "Character 3",
							CreatedAt: time.Now(),
							UpdatedAt: time.Now(),
						},
					}, nil)
			},
			wantErr:   false,
			wantCount: 3,
		},
		{
			name: "success - empty list",
			mockSetup: func(m *mock.MockCharacterRepository, ctx context.Context) {
				m.EXPECT().
					ListCharacters(ctx).
					Return([]*ent.Character{}, nil)
			},
			wantErr:   false,
			wantCount: 0,
		},
		{
			name: "failure - database error",
			mockSetup: func(m *mock.MockCharacterRepository, ctx context.Context) {
				m.EXPECT().
					ListCharacters(ctx).
					Return(nil, errors.New("database connection error"))
			},
			wantErr:   true,
			wantCount: 0,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := mock.NewMockCharacterRepository(ctrl)
			interactor := interactor.NewCharacterInteractor(mockRepo)

			ctx := t.Context()
			testCase.mockSetup(mockRepo, ctx)

			result, err := interactor.ListCharacters(ctx)

			if testCase.wantErr {
				require.Error(t, err)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)
				assert.Len(t, result, testCase.wantCount)
			}
		})
	}
}

func TestCharacterInteractor_UpdateCharacter(t *testing.T) {
	t.Parallel()

	characterID := 1
	newName := "Updated Character"

	tests := []struct {
		name      string
		nameParam *string
		mockSetup func(*mock.MockCharacterRepository, context.Context)
		wantErr   bool
	}{
		{
			name:      "success - update name",
			nameParam: &newName,
			mockSetup: func(m *mock.MockCharacterRepository, ctx context.Context) {
				m.EXPECT().
					UpdateCharacter(ctx, characterID, &newName).
					Return(&ent.Character{
						ID:        characterID,
						Name:      newName,
						CreatedAt: time.Now().Add(-24 * time.Hour),
						UpdatedAt: time.Now(),
					}, nil)
			},
			wantErr: false,
		},
		{
			name:      "success - no update (nil name)",
			nameParam: nil,
			mockSetup: func(m *mock.MockCharacterRepository, ctx context.Context) {
				m.EXPECT().
					UpdateCharacter(ctx, characterID, nil).
					Return(&ent.Character{
						ID:        characterID,
						Name:      "Original Name",
						CreatedAt: time.Now().Add(-24 * time.Hour),
						UpdatedAt: time.Now(),
					}, nil)
			},
			wantErr: false,
		},
		{
			name:      "failure - not found",
			nameParam: &newName,
			mockSetup: func(m *mock.MockCharacterRepository, ctx context.Context) {
				m.EXPECT().
					UpdateCharacter(ctx, characterID, &newName).
					Return(nil, errors.New("not found"))
			},
			wantErr: true,
		},
		{
			name:      "failure - duplicate name",
			nameParam: &newName,
			mockSetup: func(m *mock.MockCharacterRepository, ctx context.Context) {
				m.EXPECT().
					UpdateCharacter(ctx, characterID, &newName).
					Return(nil, errors.New("unique constraint violation"))
			},
			wantErr: true,
		},
		{
			name:      "failure - database error",
			nameParam: &newName,
			mockSetup: func(m *mock.MockCharacterRepository, ctx context.Context) {
				m.EXPECT().
					UpdateCharacter(ctx, characterID, &newName).
					Return(nil, errors.New("database connection error"))
			},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := mock.NewMockCharacterRepository(ctrl)
			interactor := interactor.NewCharacterInteractor(mockRepo)

			ctx := t.Context()
			testCase.mockSetup(mockRepo, ctx)

			result, err := interactor.UpdateCharacter(ctx, characterID, testCase.nameParam)

			if testCase.wantErr {
				require.Error(t, err)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, characterID, result.ID)
			}
		})
	}
}

func TestCharacterInteractor_DeleteCharacter(t *testing.T) {
	t.Parallel()

	characterID := 1

	tests := []struct {
		name      string
		mockSetup func(*mock.MockCharacterRepository, context.Context)
		wantErr   bool
	}{
		{
			name: "success",
			mockSetup: func(m *mock.MockCharacterRepository, ctx context.Context) {
				m.EXPECT().DeleteCharacter(ctx, characterID).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "not found",
			mockSetup: func(m *mock.MockCharacterRepository, ctx context.Context) {
				m.EXPECT().DeleteCharacter(ctx, characterID).Return(errors.New("not found"))
			},
			wantErr: true,
		},
		{
			name: "delete failed - database error",
			mockSetup: func(m *mock.MockCharacterRepository, ctx context.Context) {
				m.EXPECT().DeleteCharacter(ctx, characterID).Return(errors.New("database error"))
			},
			wantErr: true,
		},
		{
			name: "delete failed - foreign key constraint",
			mockSetup: func(m *mock.MockCharacterRepository, ctx context.Context) {
				m.EXPECT().
					DeleteCharacter(ctx, characterID).
					Return(errors.New("foreign key constraint violation"))
			},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := mock.NewMockCharacterRepository(ctrl)
			interactor := interactor.NewCharacterInteractor(mockRepo)

			ctx := t.Context()
			testCase.mockSetup(mockRepo, ctx)

			err := interactor.DeleteCharacter(ctx, characterID)

			if testCase.wantErr {
				require.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
