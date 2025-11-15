package interactor_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kizuna-org/akari/gen/ent"
	"github.com/kizuna-org/akari/gen/ent/systemprompt"
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
