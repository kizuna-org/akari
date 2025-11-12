package interactor_test

import (
	"context"
	"errors"
	"testing"

	"github.com/kizuna-org/akari/gen/ent"
	"github.com/kizuna-org/akari/pkg/database/domain/mock"
	"github.com/kizuna-org/akari/pkg/database/usecase/interactor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestNewCharacterConfigInteractor(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock.NewMockCharacterConfigRepository(ctrl)
	interactor := interactor.NewCharacterConfigInteractor(mockRepo)

	assert.NotNil(t, interactor)
}

func TestCharacterConfigInteractor_CreateCharacterConfig(t *testing.T) {
	t.Parallel()

	defaultPrompt := "default system prompt"
	nameRegexp := "^Test"

	var characterID *int = nil

	tests := []struct {
		name      string
		mockSetup func(*mock.MockCharacterConfigRepository, context.Context)
		wantErr   bool
	}{
		{
			name: "success",
			mockSetup: func(m *mock.MockCharacterConfigRepository, ctx context.Context) {
				m.EXPECT().
					CreateCharacterConfig(ctx, characterID, &nameRegexp, defaultPrompt).
					Return(&ent.CharacterConfig{
						ID:                  1,
						NameRegexp:          &nameRegexp,
						DefaultSystemPrompt: defaultPrompt,
					}, nil)
			},
			wantErr: false,
		},
		{
			name: "failure - database error",
			mockSetup: func(m *mock.MockCharacterConfigRepository, ctx context.Context) {
				m.EXPECT().
					CreateCharacterConfig(ctx, characterID, &nameRegexp, defaultPrompt).
					Return(nil, errors.New("database error"))
			},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := mock.NewMockCharacterConfigRepository(ctrl)
			inter := interactor.NewCharacterConfigInteractor(mockRepo)

			ctx := t.Context()
			testCase.mockSetup(mockRepo, ctx)

			res, err := inter.CreateCharacterConfig(ctx, characterID, &nameRegexp, defaultPrompt)

			if testCase.wantErr {
				require.Error(t, err)
				assert.Nil(t, res)
			} else {
				require.NoError(t, err)
				require.NotNil(t, res)
				assert.Equal(t, defaultPrompt, res.DefaultSystemPrompt)
			}
		})
	}
}

func TestCharacterConfigInteractor_GetCharacterConfigByID(t *testing.T) {
	t.Parallel()

	characterConfigID := 1

	tests := []struct {
		name      string
		mockSetup func(*mock.MockCharacterConfigRepository, context.Context)
		wantErr   bool
	}{
		{
			name: "success",
			mockSetup: func(m *mock.MockCharacterConfigRepository, ctx context.Context) {
				m.EXPECT().
					GetCharacterConfigByID(ctx, characterConfigID).
					Return(&ent.CharacterConfig{ID: characterConfigID, DefaultSystemPrompt: "p"}, nil)
			},
			wantErr: false,
		},
		{
			name: "not found",
			mockSetup: func(m *mock.MockCharacterConfigRepository, ctx context.Context) {
				m.EXPECT().
					GetCharacterConfigByID(ctx, characterConfigID).
					Return(nil, errors.New("not found"))
			},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := mock.NewMockCharacterConfigRepository(ctrl)
			inter := interactor.NewCharacterConfigInteractor(mockRepo)

			ctx := t.Context()
			testCase.mockSetup(mockRepo, ctx)

			res, err := inter.GetCharacterConfigByID(ctx, characterConfigID)

			if testCase.wantErr {
				require.Error(t, err)
				assert.Nil(t, res)
			} else {
				require.NoError(t, err)
				require.NotNil(t, res)
				assert.Equal(t, characterConfigID, res.ID)
			}
		})
	}
}

func TestCharacterConfigInteractor_GetCharacterConfigByCharacterID(t *testing.T) {
	t.Parallel()

	characterID := 1

	tests := []struct {
		name      string
		mockSetup func(*mock.MockCharacterConfigRepository, context.Context)
		wantErr   bool
	}{
		{
			name: "success",
			mockSetup: func(m *mock.MockCharacterConfigRepository, ctx context.Context) {
				m.EXPECT().
					GetCharacterConfigByCharacterID(ctx, characterID).
					Return(&ent.CharacterConfig{ID: 1, DefaultSystemPrompt: "p"}, nil)
			},
			wantErr: false,
		},
		{
			name: "not found",
			mockSetup: func(m *mock.MockCharacterConfigRepository, ctx context.Context) {
				m.EXPECT().
					GetCharacterConfigByCharacterID(ctx, characterID).
					Return(nil, errors.New("not found"))
			},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := mock.NewMockCharacterConfigRepository(ctrl)
			inter := interactor.NewCharacterConfigInteractor(mockRepo)

			ctx := t.Context()
			testCase.mockSetup(mockRepo, ctx)

			res, err := inter.GetCharacterConfigByCharacterID(ctx, characterID)

			if testCase.wantErr {
				require.Error(t, err)
				assert.Nil(t, res)
			} else {
				require.NoError(t, err)
				require.NotNil(t, res)
				assert.Equal(t, 1, res.ID)
			}
		})
	}
}

func TestCharacterConfigInteractor_UpdateCharacterConfig(t *testing.T) {
	t.Parallel()

	characterConfigID := 1
	newPrompt := "updated prompt"

	tests := []struct {
		name      string
		argPrompt *string
		mockSetup func(*mock.MockCharacterConfigRepository, context.Context)
		wantErr   bool
	}{
		{
			name:      "success - update default prompt",
			argPrompt: &newPrompt,
			mockSetup: func(m *mock.MockCharacterConfigRepository, ctx context.Context) {
				m.EXPECT().
					UpdateCharacterConfig(ctx, characterConfigID, nil, nil, &newPrompt).
					Return(&ent.CharacterConfig{ID: characterConfigID, DefaultSystemPrompt: newPrompt}, nil)
			},
			wantErr: false,
		},
		{
			name:      "failure - not found",
			argPrompt: &newPrompt,
			mockSetup: func(m *mock.MockCharacterConfigRepository, ctx context.Context) {
				m.EXPECT().
					UpdateCharacterConfig(ctx, characterConfigID, nil, nil, &newPrompt).
					Return(nil, errors.New("not found"))
			},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := mock.NewMockCharacterConfigRepository(ctrl)
			inter := interactor.NewCharacterConfigInteractor(mockRepo)

			ctx := t.Context()
			testCase.mockSetup(mockRepo, ctx)

			res, err := inter.UpdateCharacterConfig(ctx, characterConfigID, nil, nil, testCase.argPrompt)

			if testCase.wantErr {
				require.Error(t, err)
				assert.Nil(t, res)
			} else {
				require.NoError(t, err)
				require.NotNil(t, res)
				assert.Equal(t, characterConfigID, res.ID)
			}
		})
	}
}

func TestCharacterConfigInteractor_DeleteCharacterConfig(t *testing.T) {
	t.Parallel()

	characterConfigID := 1

	tests := []struct {
		name      string
		mockSetup func(*mock.MockCharacterConfigRepository, context.Context)
		wantErr   bool
	}{
		{
			name: "success",
			mockSetup: func(m *mock.MockCharacterConfigRepository, ctx context.Context) {
				m.EXPECT().DeleteCharacterConfig(ctx, characterConfigID).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "not found",
			mockSetup: func(m *mock.MockCharacterConfigRepository, ctx context.Context) {
				m.EXPECT().DeleteCharacterConfig(ctx, characterConfigID).Return(errors.New("not found"))
			},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := mock.NewMockCharacterConfigRepository(ctrl)
			inter := interactor.NewCharacterConfigInteractor(mockRepo)

			ctx := t.Context()
			testCase.mockSetup(mockRepo, ctx)

			err := inter.DeleteCharacterConfig(ctx, characterConfigID)

			if testCase.wantErr {
				require.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
