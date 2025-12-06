package adapter_test

import (
	"errors"
	"testing"

	"github.com/kizuna-org/akari/internal/message/adapter"
	llmMock "github.com/kizuna-org/akari/pkg/llm/usecase/interactor/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestLLMRepository_GenerateResponse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		systemPrompt string
		userMessage  string
		setupMock    func(*llmMock.MockLLMInteractor)
		want         string
		wantErr      bool
	}{
		{
			name:         "success",
			systemPrompt: "system prompt",
			userMessage:  "Hello user",
			setupMock: func(m *llmMock.MockLLMInteractor) {
				response := "Hello response"
				responses := []*string{&response}
				m.EXPECT().SendChatMessage(gomock.Any(), "system prompt", nil, "Hello user", nil).
					Return(responses, nil, nil)
			},
			want:    "Hello response",
			wantErr: false,
		},
		{
			name:         "no response",
			systemPrompt: "system prompt",
			userMessage:  "Hello user",
			setupMock: func(m *llmMock.MockLLMInteractor) {
				responses := []*string{}
				m.EXPECT().SendChatMessage(gomock.Any(), "system prompt", nil, "Hello user", nil).
					Return(responses, nil, nil)
			},
			want:    "",
			wantErr: true,
		},
		{
			name:         "llm error",
			systemPrompt: "system prompt",
			userMessage:  "Hello user",
			setupMock: func(m *llmMock.MockLLMInteractor) {
				m.EXPECT().SendChatMessage(gomock.Any(), "system prompt", nil, "Hello user", nil).
					Return(nil, nil, errors.New("llm error"))
			},
			want:    "",
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			t.Cleanup(ctrl.Finish)

			mockLLMInteractor := llmMock.NewMockLLMInteractor(ctrl)
			testCase.setupMock(mockLLMInteractor)

			repo := adapter.NewLLMRepository(mockLLMInteractor)
			result, err := repo.GenerateResponse(t.Context(), testCase.systemPrompt, testCase.userMessage)

			if testCase.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, testCase.want, result)
			}
		})
	}
}
