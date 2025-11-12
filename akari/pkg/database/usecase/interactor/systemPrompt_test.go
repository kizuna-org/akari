package interactor_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kizuna-org/akari/gen/ent"
	"github.com/kizuna-org/akari/gen/ent/systemprompt"
	"github.com/kizuna-org/akari/pkg/database/domain"
	"github.com/kizuna-org/akari/pkg/database/domain/mock"
	"github.com/kizuna-org/akari/pkg/database/usecase/interactor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestNewSystemPromptInteractor(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock.NewMockSystemPromptRepository(ctrl)
	interactor := interactor.NewSystemPromptInteractor(mockRepo)

	assert.NotNil(t, interactor)
}

func TestSystemPromptInteractor_CreateSystemPrompt(t *testing.T) {
	t.Parallel()

	title := "Test Prompt"
	prompt := "This is a test prompt"
	purpose := systemprompt.PurposeTextChat

	tests := []struct {
		name      string
		mockSetup func(*mock.MockSystemPromptRepository, context.Context)
		wantErr   bool
	}{
		{
			name: "success",
			mockSetup: func(m *mock.MockSystemPromptRepository, ctx context.Context) {
				m.EXPECT().
					CreateSystemPrompt(ctx, title, prompt, purpose).
					Return(&ent.SystemPrompt{
						ID:        1,
						Title:     title,
						Prompt:    prompt,
						Purpose:   purpose,
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					}, nil)
			},
			wantErr: false,
		},
		{
			name: "failure",
			mockSetup: func(m *mock.MockSystemPromptRepository, ctx context.Context) {
				m.EXPECT().
					CreateSystemPrompt(ctx, title, prompt, purpose).
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

			mockRepo := mock.NewMockSystemPromptRepository(ctrl)
			interactor := interactor.NewSystemPromptInteractor(mockRepo)

			ctx := t.Context()
			testCase.mockSetup(mockRepo, ctx)

			result, err := interactor.CreateSystemPrompt(ctx, title, prompt, purpose)

			if testCase.wantErr {
				require.Error(t, err)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

func TestSystemPromptInteractor_GetSystemPromptByID(t *testing.T) {
	t.Parallel()

	promptID := 1

	tests := []struct {
		name      string
		mockSetup func(*mock.MockSystemPromptRepository, context.Context)
		wantErr   bool
	}{
		{
			name: "success",
			mockSetup: func(m *mock.MockSystemPromptRepository, ctx context.Context) {
				m.EXPECT().
					GetSystemPromptByID(ctx, promptID).
					Return(&ent.SystemPrompt{
						ID:        promptID,
						Title:     "Test Prompt",
						Prompt:    "This is a test prompt",
						Purpose:   systemprompt.PurposeTextChat,
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					}, nil)
			},
			wantErr: false,
		},
		{
			name: "not found",
			mockSetup: func(m *mock.MockSystemPromptRepository, ctx context.Context) {
				m.EXPECT().
					GetSystemPromptByID(ctx, promptID).
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

			mockRepo := mock.NewMockSystemPromptRepository(ctrl)
			interactor := interactor.NewSystemPromptInteractor(mockRepo)

			ctx := t.Context()
			testCase.mockSetup(mockRepo, ctx)

			result, err := interactor.GetSystemPromptByID(ctx, promptID)

			if testCase.wantErr {
				require.Error(t, err)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

func TestSystemPromptInteractor_UpdateSystemPrompt(t *testing.T) {
	t.Parallel()

	promptID := 1
	newTitle := "Updated Title"
	newPrompt := "Updated prompt"
	newPurpose := systemprompt.PurposeTextChat

	tests := []struct {
		name      string
		title     *string
		prompt    *string
		purpose   *domain.SystemPromptPurpose
		mockSetup func(*mock.MockSystemPromptRepository, context.Context)
		wantErr   bool
	}{
		{
			name:    "full update success",
			title:   &newTitle,
			prompt:  &newPrompt,
			purpose: &newPurpose,
			mockSetup: func(m *mock.MockSystemPromptRepository, ctx context.Context) {
				m.EXPECT().
					UpdateSystemPrompt(ctx, promptID, &newTitle, &newPrompt, &newPurpose).
					Return(&ent.SystemPrompt{
						ID:        promptID,
						Title:     newTitle,
						Prompt:    newPrompt,
						Purpose:   newPurpose,
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					}, nil)
			},
			wantErr: false,
		},
		{
			name:    "partial update - title only",
			title:   &newTitle,
			prompt:  nil,
			purpose: nil,
			mockSetup: func(m *mock.MockSystemPromptRepository, ctx context.Context) {
				m.EXPECT().
					UpdateSystemPrompt(ctx, promptID, &newTitle, nil, nil).
					Return(&ent.SystemPrompt{
						ID:        promptID,
						Title:     newTitle,
						Purpose:   systemprompt.PurposeTextChat,
						Prompt:    "Original prompt",
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					}, nil)
			},
			wantErr: false,
		},
		{
			name:    "update failure",
			title:   &newTitle,
			prompt:  nil,
			purpose: nil,
			mockSetup: func(m *mock.MockSystemPromptRepository, ctx context.Context) {
				m.EXPECT().
					UpdateSystemPrompt(ctx, promptID, &newTitle, nil, nil).
					Return(nil, errors.New("update failed"))
			},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := mock.NewMockSystemPromptRepository(ctrl)
			interactor := interactor.NewSystemPromptInteractor(mockRepo)

			ctx := t.Context()
			testCase.mockSetup(mockRepo, ctx)

			result, err := interactor.UpdateSystemPrompt(ctx, promptID, testCase.title, testCase.prompt, testCase.purpose)

			if testCase.wantErr {
				require.Error(t, err)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

func TestSystemPromptInteractor_DeleteSystemPrompt(t *testing.T) {
	t.Parallel()

	promptID := 1

	tests := []struct {
		name      string
		mockSetup func(*mock.MockSystemPromptRepository, context.Context)
		wantErr   bool
	}{
		{
			name: "success",
			mockSetup: func(m *mock.MockSystemPromptRepository, ctx context.Context) {
				m.EXPECT().DeleteSystemPrompt(ctx, promptID).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "not found",
			mockSetup: func(m *mock.MockSystemPromptRepository, ctx context.Context) {
				m.EXPECT().DeleteSystemPrompt(ctx, promptID).Return(errors.New("not found"))
			},
			wantErr: true,
		},
		{
			name: "delete failed",
			mockSetup: func(m *mock.MockSystemPromptRepository, ctx context.Context) {
				m.EXPECT().DeleteSystemPrompt(ctx, promptID).Return(errors.New("delete failed"))
			},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := mock.NewMockSystemPromptRepository(ctrl)
			interactor := interactor.NewSystemPromptInteractor(mockRepo)

			ctx := t.Context()
			testCase.mockSetup(mockRepo, ctx)

			err := interactor.DeleteSystemPrompt(ctx, promptID)

			if testCase.wantErr {
				require.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
