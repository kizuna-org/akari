package adapter_test

import (
	"errors"
	"testing"

	"github.com/kizuna-org/akari/internal/message/adapter"
	interactorMock "github.com/kizuna-org/akari/pkg/database/usecase/interactor/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestConversationRepository_CreateConversation(t *testing.T) {
	t.Parallel()

	groupID := intPtr(1)

	tests := []struct {
		name                string
		messageID           string
		userID              int
		conversationGroupID *int
		setupMock           func(*interactorMock.MockConversationInteractor)
		wantErr             bool
		errMsg              string
	}{
		{
			name:                "success - with conversation group",
			messageID:           "msg-001",
			userID:              1,
			conversationGroupID: groupID,
			setupMock: func(m *interactorMock.MockConversationInteractor) {
				m.EXPECT().CreateConversation(gomock.Any(), "msg-001", 1, intPtr(1)).Return(nil, nil)
			},
			wantErr: false,
		},
		{
			name:                "success - without conversation group",
			messageID:           "msg-002",
			userID:              2,
			conversationGroupID: nil,
			setupMock: func(m *interactorMock.MockConversationInteractor) {
				m.EXPECT().CreateConversation(gomock.Any(), "msg-002", 2, nil).Return(nil, nil)
			},
			wantErr: false,
		},
		{
			name:                "error - create failed",
			messageID:           "msg-003",
			userID:              3,
			conversationGroupID: groupID,
			setupMock: func(m *interactorMock.MockConversationInteractor) {
				m.EXPECT().CreateConversation(gomock.Any(), "msg-003", 3, intPtr(1)).Return(nil, errors.New("db error"))
			},
			wantErr: true,
			errMsg:  "failed to create conversation",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			t.Cleanup(ctrl.Finish)

			mockInteractor := interactorMock.NewMockConversationInteractor(ctrl)
			testCase.setupMock(mockInteractor)

			repo := adapter.NewConversationRepository(mockInteractor)
			err := repo.CreateConversation(
				t.Context(),
				testCase.messageID,
				testCase.userID,
				testCase.conversationGroupID,
			)

			if testCase.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), testCase.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func intPtr(i int) *int {
	return &i
}
