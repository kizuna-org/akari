package adapter_test

import (
	"errors"
	"testing"

	"github.com/kizuna-org/akari/gen/ent"
	"github.com/kizuna-org/akari/internal/message/adapter"
	"github.com/kizuna-org/akari/pkg/database/domain"
	interactorMock "github.com/kizuna-org/akari/pkg/database/usecase/interactor/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func newNotFoundError() error {
	return &ent.NotFoundError{}
}

func TestAkariUserRepository_GetOrCreateAkariUserByDiscordUserID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		userID    string
		setupMock func(*interactorMock.MockAkariUserInteractor)
		want      int
		wantErr   bool
		errMsg    string
	}{
		{
			name:   "success - existing user",
			userID: "user-001",
			setupMock: func(m *interactorMock.MockAkariUserInteractor) {
				m.EXPECT().GetAkariUserByDiscordUserID(gomock.Any(), "user-001").Return(&domain.AkariUser{ID: 1}, nil)
			},
			want:    1,
			wantErr: false,
		},
		{
			name:   "success - create new user",
			userID: "user-002",
			setupMock: func(m *interactorMock.MockAkariUserInteractor) {
				m.EXPECT().GetAkariUserByDiscordUserID(gomock.Any(), "user-002").Return(nil, newNotFoundError())
				m.EXPECT().CreateAkariUser(gomock.Any()).Return(&domain.AkariUser{ID: 2}, nil)
			},
			want:    2,
			wantErr: false,
		},
		{
			name:   "error - create user failed",
			userID: "user-003",
			setupMock: func(m *interactorMock.MockAkariUserInteractor) {
				m.EXPECT().GetAkariUserByDiscordUserID(gomock.Any(), "user-003").Return(nil, newNotFoundError())
				m.EXPECT().CreateAkariUser(gomock.Any()).Return(nil, errors.New("db error"))
			},
			want:    0,
			wantErr: true,
			errMsg:  "failed to create akari user",
		},
		{
			name:   "error - get akari user failed with non-not-found error",
			userID: "user-004",
			setupMock: func(m *interactorMock.MockAkariUserInteractor) {
				m.EXPECT().GetAkariUserByDiscordUserID(gomock.Any(), "user-004").Return(nil, errors.New("database error"))
			},
			want:    0,
			wantErr: true,
			errMsg:  "failed to get akari user by discord user id",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			t.Cleanup(ctrl.Finish)

			mockInteractor := interactorMock.NewMockAkariUserInteractor(ctrl)
			testCase.setupMock(mockInteractor)

			repo := adapter.NewAkariUserRepository(mockInteractor)
			result, err := repo.GetOrCreateAkariUserByDiscordUserID(t.Context(), testCase.userID)

			if testCase.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), testCase.errMsg)
			} else {
				require.NoError(t, err)
				require.Equal(t, testCase.want, result)
			}
		})
	}
}
