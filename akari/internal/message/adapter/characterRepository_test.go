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

func TestCharacterRepository_GetCharacterByID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		charID    int
		setupMock func(*interactorMock.MockCharacterInteractor)
		wantErr   bool
		errMsg    string
	}{
		{
			name:   "success",
			charID: 1,
			setupMock: func(m *interactorMock.MockCharacterInteractor) {
				m.EXPECT().GetCharacterByID(gomock.Any(), 1).Return(&domain.Character{
					ID:              1,
					Name:            "Alice",
					SystemPromptIDs: []int{1, 2},
				}, nil)
			},
			wantErr: false,
		},
		{
			name:   "character not found",
			charID: 999,
			setupMock: func(m *interactorMock.MockCharacterInteractor) {
				m.EXPECT().GetCharacterByID(gomock.Any(), 999).Return(nil, errors.New("not found"))
			},
			wantErr: true,
			errMsg:  "failed to get character by id",
		},
		{
			name:   "database error",
			charID: 2,
			setupMock: func(m *interactorMock.MockCharacterInteractor) {
				m.EXPECT().GetCharacterByID(gomock.Any(), 2).Return(nil, errors.New("db error"))
			},
			wantErr: true,
			errMsg:  "failed to get character by id",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			t.Cleanup(ctrl.Finish)

			mockInteractor := interactorMock.NewMockCharacterInteractor(ctrl)
			testCase.setupMock(mockInteractor)

			repo := adapter.NewCharacterRepository(mockInteractor)
			result, err := repo.Get(t.Context(), testCase.charID)

			if testCase.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), testCase.errMsg)
				require.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				require.Equal(t, 1, result.ID)
				require.Equal(t, "Alice", result.Name)
				require.Equal(t, []int{1, 2}, result.SystemPromptIDs)
			}
		})
	}
}
