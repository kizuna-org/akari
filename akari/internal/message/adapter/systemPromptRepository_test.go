package adapter_test

import (
	"errors"
	"testing"

	"github.com/kizuna-org/akari/internal/message/adapter"
	"github.com/kizuna-org/akari/pkg/database/domain"
	interactorMock "github.com/kizuna-org/akari/pkg/database/usecase/interactor/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestSystemPromptRepository_GetSystemPromptByID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		promptID  int
		setupMock func(*interactorMock.MockSystemPromptInteractor)
		wantErr   bool
		errMsg    string
	}{
		{
			name:     "success",
			promptID: 1,
			setupMock: func(m *interactorMock.MockSystemPromptInteractor) {
				m.EXPECT().GetSystemPromptByID(gomock.Any(), 1).Return(&domain.SystemPrompt{
					ID:     1,
					Prompt: "You are a helpful assistant.",
				}, nil)
			},
			wantErr: false,
		},
		{
			name:     "prompt not found",
			promptID: 999,
			setupMock: func(m *interactorMock.MockSystemPromptInteractor) {
				m.EXPECT().GetSystemPromptByID(gomock.Any(), 999).Return(nil, errors.New("not found"))
			},
			wantErr: true,
			errMsg:  "failed to get system prompt by id",
		},
		{
			name:     "database error",
			promptID: 2,
			setupMock: func(m *interactorMock.MockSystemPromptInteractor) {
				m.EXPECT().GetSystemPromptByID(gomock.Any(), 2).Return(nil, errors.New("db error"))
			},
			wantErr: true,
			errMsg:  "failed to get system prompt by id",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			t.Cleanup(ctrl.Finish)

			mockInteractor := interactorMock.NewMockSystemPromptInteractor(ctrl)
			testCase.setupMock(mockInteractor)

			repo := adapter.NewSystemPromptRepository(mockInteractor)
			result, err := repo.Get(t.Context(), testCase.promptID)

			if testCase.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), testCase.errMsg)
				require.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				require.Equal(t, 1, result.ID)
				require.Equal(t, "You are a helpful assistant.", result.Prompt)
			}
		})
	}
}
