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

func TestConversationGroupRepository_GetConversationGroupByCharacterID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		charID    int
		setupMock func(*interactorMock.MockConversationGroupInteractor)
		wantErr   bool
		errMsg    string
	}{
		{
			name:   "success",
			charID: 1,
			setupMock: func(m *interactorMock.MockConversationGroupInteractor) {
				m.EXPECT().GetConversationGroupByCharacterID(gomock.Any(), 1).Return(&domain.ConversationGroup{
					ID:          1,
					CharacterID: 1,
				}, nil)
			},
			wantErr: false,
		},
		{
			name:   "not found",
			charID: 999,
			setupMock: func(m *interactorMock.MockConversationGroupInteractor) {
				m.EXPECT().GetConversationGroupByCharacterID(gomock.Any(), 999).Return(nil, errors.New("not found"))
			},
			wantErr: true,
			errMsg:  "failed to get conversation group by character id",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			t.Cleanup(ctrl.Finish)

			mockInteractor := interactorMock.NewMockConversationGroupInteractor(ctrl)
			testCase.setupMock(mockInteractor)

			repo := adapter.NewConversationGroupRepository(mockInteractor)
			result, err := repo.GetConversationGroupByCharacterID(t.Context(), testCase.charID)

			if testCase.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), testCase.errMsg)
				require.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				require.Equal(t, 1, result.ID)
				require.Equal(t, 1, result.CharacterID)
			}
		})
	}
}

func TestConversationGroupRepository_CreateConversationGroup(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		charID    int
		setupMock func(*interactorMock.MockConversationGroupInteractor)
		wantErr   bool
		errMsg    string
	}{
		{
			name:   "success",
			charID: 1,
			setupMock: func(m *interactorMock.MockConversationGroupInteractor) {
				m.EXPECT().CreateConversationGroup(gomock.Any(), 1).Return(&domain.ConversationGroup{
					ID:          2,
					CharacterID: 1,
				}, nil)
			},
			wantErr: false,
		},
		{
			name:   "create error",
			charID: 1,
			setupMock: func(m *interactorMock.MockConversationGroupInteractor) {
				m.EXPECT().CreateConversationGroup(gomock.Any(), 1).Return(nil, errors.New("db error"))
			},
			wantErr: true,
			errMsg:  "failed to create conversation group",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			t.Cleanup(ctrl.Finish)

			mockInteractor := interactorMock.NewMockConversationGroupInteractor(ctrl)
			testCase.setupMock(mockInteractor)

			repo := adapter.NewConversationGroupRepository(mockInteractor)
			result, err := repo.CreateConversationGroup(t.Context(), testCase.charID)

			if testCase.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), testCase.errMsg)
				require.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				require.Equal(t, 2, result.ID)
				require.Equal(t, 1, result.CharacterID)
			}
		})
	}
}
