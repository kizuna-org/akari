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

func TestDiscordUserRepository_GetDiscordUserByID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		discordUserID string
		setupMock     func(*interactorMock.MockDiscordUserInteractor, *interactorMock.MockAkariUserInteractor)
		want          int
		wantErr       bool
		errMsg        string
	}{
		{
			name:          "success",
			discordUserID: "discord-001",
			setupMock: func(
				m *interactorMock.MockDiscordUserInteractor,
				akariUserInteractor *interactorMock.MockAkariUserInteractor,
			) {
				m.EXPECT().GetDiscordUserByID(gomock.Any(), "discord-001").Return(&domain.DiscordUser{
					ID:       "discord-001",
					Username: "user1",
					Bot:      false,
				}, nil)
				akariUserInteractor.EXPECT().
					GetAkariUserByDiscordUserID(gomock.Any(), "discord-001").
					Return(&domain.AkariUser{ID: 5}, nil)
			},
			want:    5,
			wantErr: false,
		},
		{
			name:          "discord user not found",
			discordUserID: "discord-999",
			setupMock: func(m *interactorMock.MockDiscordUserInteractor,
				akariUserInteractor *interactorMock.MockAkariUserInteractor) {
				m.EXPECT().GetDiscordUserByID(gomock.Any(), "discord-999").Return(nil,
					errors.New("not found"))
			},
			want:    0,
			wantErr: true,
			errMsg:  "failed to get discord user by id",
		},
		{
			name:          "akari user not found - error",
			discordUserID: "discord-002",
			setupMock: func(m *interactorMock.MockDiscordUserInteractor,
				akariUserInteractor *interactorMock.MockAkariUserInteractor) {
				m.EXPECT().GetDiscordUserByID(gomock.Any(), "discord-002").Return(&domain.DiscordUser{
					ID:       "discord-002",
					Username: "user2",
					Bot:      false,
				}, nil)
				akariUserInteractor.EXPECT().GetAkariUserByDiscordUserID(gomock.Any(),
					"discord-002").Return(nil, errors.New("db error"))
			},
			want:    0,
			wantErr: true,
			errMsg:  "failed to get akari user by discord user id",
		},
		{
			name:          "akari user not found - recover by create",
			discordUserID: "discord-003",
			setupMock: func(m *interactorMock.MockDiscordUserInteractor,
				akariUserInteractor *interactorMock.MockAkariUserInteractor) {
				m.EXPECT().GetDiscordUserByID(gomock.Any(), "discord-003").Return(&domain.DiscordUser{
					ID:       "discord-003",
					Username: "user3",
					Bot:      false,
				}, nil)
				akariUserInteractor.EXPECT().GetAkariUserByDiscordUserID(gomock.Any(),
					"discord-003").Return(nil, &ent.NotFoundError{})
				akariUserInteractor.EXPECT().CreateAkariUser(gomock.Any()).Return(&domain.AkariUser{ID: 3}, nil)
			},
			want:    3,
			wantErr: false,
		},
		{
			name:          "error - create akari user failed (GetDiscordUserByID)",
			discordUserID: "discord-009",
			setupMock: func(m *interactorMock.MockDiscordUserInteractor,
				akariUserInteractor *interactorMock.MockAkariUserInteractor) {
				m.EXPECT().GetDiscordUserByID(gomock.Any(), "discord-009").Return(&domain.DiscordUser{
					ID:       "discord-009",
					Username: "user9",
					Bot:      false,
				}, nil)
				akariUserInteractor.EXPECT().GetAkariUserByDiscordUserID(gomock.Any(),
					"discord-009").Return(nil, &ent.NotFoundError{})
				akariUserInteractor.EXPECT().CreateAkariUser(gomock.Any()).Return(nil,
					errors.New("db error"))
			},
			want:    0,
			wantErr: true,
			errMsg:  "failed to create akari user",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			t.Cleanup(ctrl.Finish)

			mockDiscordUserInteractor := interactorMock.NewMockDiscordUserInteractor(ctrl)
			mockAkariUserInteractor := interactorMock.NewMockAkariUserInteractor(ctrl)
			testCase.setupMock(mockDiscordUserInteractor, mockAkariUserInteractor)

			repo := adapter.NewDiscordUserRepository(mockDiscordUserInteractor, mockAkariUserInteractor)
			result, err := repo.GetDiscordUserByID(t.Context(), testCase.discordUserID)

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

func TestDiscordUserRepository_GetOrCreateDiscordUser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		discordUserID string
		username      string
		isBot         bool
		setupMock     func(*interactorMock.MockDiscordUserInteractor, *interactorMock.MockAkariUserInteractor)
		want          int
		wantErr       bool
		errMsg        string
	}{
		{
			name:          "success - existing user",
			discordUserID: "discord-001",
			username:      "user1",
			isBot:         false,
			setupMock: func(m *interactorMock.MockDiscordUserInteractor,
				akariUserInteractor *interactorMock.MockAkariUserInteractor) {
				m.EXPECT().GetDiscordUserByID(gomock.Any(), "discord-001").Return(&domain.DiscordUser{
					ID:       "discord-001",
					Username: "user1",
					Bot:      false,
				}, nil)
				akariUserInteractor.EXPECT().GetAkariUserByDiscordUserID(gomock.Any(),
					"discord-001").Return(&domain.AkariUser{ID: 1}, nil)
			},
			want:    1,
			wantErr: false,
		},
		{
			name:          "success - create new user",
			discordUserID: "discord-002",
			username:      "user2",
			isBot:         false,
			setupMock: func(m *interactorMock.MockDiscordUserInteractor,
				akariUserInteractor *interactorMock.MockAkariUserInteractor) {
				m.EXPECT().GetDiscordUserByID(gomock.Any(), "discord-002").Return(nil,
					&ent.NotFoundError{})
				m.EXPECT().CreateDiscordUser(gomock.Any(), gomock.Any()).Return(&domain.DiscordUser{
					ID:       "discord-002",
					Username: "user2",
					Bot:      false,
				}, nil)
				akariUserInteractor.EXPECT().CreateAkariUser(gomock.Any()).Return(
					&domain.AkariUser{ID: 2}, nil)
			},
			want:    2,
			wantErr: false,
		},
		{
			name:          "success - discord exists but akari missing",
			discordUserID: "discord-005",
			username:      "user5",
			isBot:         false,
			setupMock: func(m *interactorMock.MockDiscordUserInteractor,
				akariUserInteractor *interactorMock.MockAkariUserInteractor) {
				m.EXPECT().GetDiscordUserByID(gomock.Any(), "discord-005").Return(&domain.DiscordUser{
					ID:       "discord-005",
					Username: "user5",
					Bot:      false,
				}, nil)
				akariUserInteractor.EXPECT().GetAkariUserByDiscordUserID(gomock.Any(),
					"discord-005").Return(nil, &ent.NotFoundError{})
				akariUserInteractor.EXPECT().CreateAkariUser(gomock.Any()).Return(
					&domain.AkariUser{ID: 5}, nil)
			},
			want:    5,
			wantErr: false,
		},
		{
			name:          "error - get discord user failed (non-not-found)",
			discordUserID: "discord-006",
			username:      "user6",
			isBot:         false,
			setupMock: func(m *interactorMock.MockDiscordUserInteractor,
				akariUserInteractor *interactorMock.MockAkariUserInteractor) {
				m.EXPECT().GetDiscordUserByID(gomock.Any(), "discord-006").Return(nil,
					errors.New("db connection error"))
			},
			want:    0,
			wantErr: true,
			errMsg:  "failed to get discord user by id",
		},
		{
			name:          "error - get akari user failed (non-not-found)",
			discordUserID: "discord-007",
			username:      "user7",
			isBot:         false,
			setupMock: func(m *interactorMock.MockDiscordUserInteractor,
				akariUserInteractor *interactorMock.MockAkariUserInteractor) {
				m.EXPECT().GetDiscordUserByID(gomock.Any(), "discord-007").Return(&domain.DiscordUser{
					ID:       "discord-007",
					Username: "user7",
					Bot:      false,
				}, nil)
				akariUserInteractor.EXPECT().GetAkariUserByDiscordUserID(gomock.Any(),
					"discord-007").Return(nil, errors.New("db error"))
			},
			want:    0,
			wantErr: true,
			errMsg:  "failed to get akari user by discord user id",
		},
		{
			name:          "error - create discord user failed",
			discordUserID: "discord-003",
			username:      "user3",
			isBot:         true,
			setupMock: func(m *interactorMock.MockDiscordUserInteractor,
				akariUserInteractor *interactorMock.MockAkariUserInteractor) {
				m.EXPECT().GetDiscordUserByID(gomock.Any(), "discord-003").Return(nil,
					&ent.NotFoundError{})
				m.EXPECT().CreateDiscordUser(gomock.Any(), gomock.Any()).Return(nil,
					errors.New("db error"))
			},
			want:    0,
			wantErr: true,
			errMsg:  "failed to create discord user",
		},
		{
			name:          "error - create akari user failed",
			discordUserID: "discord-004",
			username:      "user4",
			isBot:         false,
			setupMock: func(m *interactorMock.MockDiscordUserInteractor,
				akariUserInteractor *interactorMock.MockAkariUserInteractor) {
				m.EXPECT().GetDiscordUserByID(gomock.Any(), "discord-004").Return(nil,
					&ent.NotFoundError{})
				m.EXPECT().CreateDiscordUser(gomock.Any(), gomock.Any()).Return(
					&domain.DiscordUser{
						ID:       "discord-004",
						Username: "user4",
						Bot:      false,
					}, nil)
				akariUserInteractor.EXPECT().CreateAkariUser(gomock.Any()).Return(nil,
					errors.New("db error"))
			},
			want:    0,
			wantErr: true,
			errMsg:  "failed to create akari user",
		},
		{
			name:          "error - create akari user failed (GetOrCreateDiscordUser after GetDiscordUserByID)",
			discordUserID: "discord-008",
			username:      "user8",
			isBot:         false,
			setupMock: func(m *interactorMock.MockDiscordUserInteractor,
				akariUserInteractor *interactorMock.MockAkariUserInteractor) {
				m.EXPECT().GetDiscordUserByID(gomock.Any(), "discord-008").Return(&domain.DiscordUser{
					ID:       "discord-008",
					Username: "user8",
					Bot:      false,
				}, nil)
				akariUserInteractor.EXPECT().GetAkariUserByDiscordUserID(gomock.Any(),
					"discord-008").Return(nil, &ent.NotFoundError{})
				akariUserInteractor.EXPECT().CreateAkariUser(gomock.Any()).Return(nil,
					errors.New("db error"))
			},
			want:    0,
			wantErr: true,
			errMsg:  "failed to create akari user",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			t.Cleanup(ctrl.Finish)

			mockDiscordUserInteractor := interactorMock.NewMockDiscordUserInteractor(ctrl)
			mockAkariUserInteractor := interactorMock.NewMockAkariUserInteractor(ctrl)
			testCase.setupMock(mockDiscordUserInteractor, mockAkariUserInteractor)

			repo := adapter.NewDiscordUserRepository(mockDiscordUserInteractor, mockAkariUserInteractor)
			result, err := repo.GetOrCreateDiscordUser(
				t.Context(),
				testCase.discordUserID,
				testCase.username,
				testCase.isBot,
			)

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
