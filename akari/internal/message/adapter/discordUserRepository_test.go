package adapter_test

import (
	"context"
	"errors"
	"testing"

	"github.com/kizuna-org/akari/gen/ent"
	"github.com/kizuna-org/akari/internal/message/adapter"
	"github.com/kizuna-org/akari/internal/message/domain/entity"
	"github.com/kizuna-org/akari/pkg/database/domain"
	"github.com/kizuna-org/akari/pkg/database/usecase/interactor/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestDiscordUserRepository_CreateIfNotExists(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		user      *entity.DiscordUser
		setupMock func(*mock.MockDiscordUserInteractor, context.Context)
		wantID    string
		wantErr   bool
		errMsg    string
	}{
		{
			name: "user already exists",
			user: &entity.DiscordUser{ID: "user-001", Username: "testuser", Bot: false},
			setupMock: func(m *mock.MockDiscordUserInteractor, ctx context.Context) {
				m.EXPECT().
					GetDiscordUserByID(ctx, "user-001").
					Return(&domain.DiscordUser{ID: "user-001"}, nil)
			},
			wantID:  "user-001",
			wantErr: false,
		},
		{
			name: "user does not exist and creates successfully",
			user: &entity.DiscordUser{ID: "user-002", Username: "newuser", Bot: false},
			setupMock: func(m *mock.MockDiscordUserInteractor, ctx context.Context) {
				m.EXPECT().
					GetDiscordUserByID(ctx, "user-002").
					Return(nil, &ent.NotFoundError{})
				m.EXPECT().
					CreateDiscordUser(ctx, gomock.Any()).
					Return(&domain.DiscordUser{ID: "user-002"}, nil)
			},
			wantID:  "user-002",
			wantErr: false,
		},
		{
			name: "user does not exist and creation fails",
			user: &entity.DiscordUser{ID: "user-003", Username: "failuser", Bot: true},
			setupMock: func(m *mock.MockDiscordUserInteractor, ctx context.Context) {
				m.EXPECT().
					GetDiscordUserByID(ctx, "user-003").
					Return(nil, &ent.NotFoundError{})
				m.EXPECT().
					CreateDiscordUser(ctx, gomock.Any()).
					Return(nil, errors.New("create error"))
			},
			wantID:  "",
			wantErr: true,
			errMsg:  "failed to create discord user",
		},
		{
			name: "get user returns non-not-found error",
			user: &entity.DiscordUser{ID: "user-004", Username: "erruser", Bot: false},
			setupMock: func(m *mock.MockDiscordUserInteractor, ctx context.Context) {
				m.EXPECT().
					GetDiscordUserByID(ctx, "user-004").
					Return(nil, errors.New("database error"))
			},
			wantID:  "",
			wantErr: true,
			errMsg:  "failed to get discord user by id",
		},
		{
			name:      "nil user",
			user:      nil,
			setupMock: func(m *mock.MockDiscordUserInteractor, ctx context.Context) {},
			wantID:    "",
			wantErr:   true,
			errMsg:    "user is required",
		},
		{
			name: "user with empty ID",
			user: &entity.DiscordUser{ID: "", Username: "emptyid", Bot: false},
			setupMock: func(m *mock.MockDiscordUserInteractor, ctx context.Context) {
				m.EXPECT().
					GetDiscordUserByID(ctx, "").
					Return(nil, &ent.NotFoundError{})
				m.EXPECT().
					CreateDiscordUser(ctx, gomock.Any()).
					Return(&domain.DiscordUser{ID: ""}, nil)
			},
			wantID:  "",
			wantErr: false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			t.Cleanup(ctrl.Finish)

			ctx := t.Context()
			mockInteractor := mock.NewMockDiscordUserInteractor(ctrl)
			testCase.setupMock(mockInteractor, ctx)

			repo := adapter.NewDiscordUserRepository(mockInteractor)
			got, err := repo.CreateIfNotExists(ctx, testCase.user)

			if testCase.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), testCase.errMsg)
			} else {
				require.NoError(t, err)
				require.Equal(t, testCase.wantID, got)
			}
		})
	}
}
