package interactor_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kizuna-org/akari/pkg/database/domain"
	"github.com/kizuna-org/akari/pkg/database/domain/mock"
	"github.com/kizuna-org/akari/pkg/database/usecase/interactor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestNewDiscordUserInteractor(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := mock.NewMockDiscordUserRepository(ctrl)
	i := interactor.NewDiscordUserInteractor(m)

	assert.NotNil(t, i)
}

func TestDiscordUserInteractor_CreateDiscordUser(t *testing.T) {
	t.Parallel()

	params := &domain.DiscordUser{
		ID:       "u1",
		Username: "user1",
	}

	tests := []struct {
		name      string
		mockSetup func(*mock.MockDiscordUserRepository, context.Context)
		wantErr   bool
	}{
		{
			name: "success",
			mockSetup: func(m *mock.MockDiscordUserRepository, ctx context.Context) {
				m.EXPECT().CreateDiscordUser(ctx, *params).Return(params, nil)
			},
			wantErr: false,
		},
		{
			name: "failure",
			mockSetup: func(m *mock.MockDiscordUserRepository, ctx context.Context) {
				m.EXPECT().CreateDiscordUser(ctx, *params).Return(nil, errors.New("create failed"))
			},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := mock.NewMockDiscordUserRepository(ctrl)
			i := interactor.NewDiscordUserInteractor(m)

			ctx := t.Context()
			testCase.mockSetup(m, ctx)

			res, err := i.CreateDiscordUser(ctx, *params)

			if testCase.wantErr {
				require.Error(t, err)
				assert.Nil(t, res)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, res)
			}
		})
	}

	t.Run("validation-empty-id", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := mock.NewMockDiscordUserRepository(ctrl)
		i := interactor.NewDiscordUserInteractor(m)

		ctx := t.Context()

		_, err := i.CreateDiscordUser(ctx, domain.DiscordUser{ID: ""})
		require.Error(t, err)
	})
}

func TestDiscordUserInteractor_GetDiscordUserByID(t *testing.T) {
	t.Parallel()

	userID := "u1"

	tests := []struct {
		name      string
		mockSetup func(*mock.MockDiscordUserRepository, context.Context)
		wantErr   bool
	}{
		{
			name: "success",
			mockSetup: func(m *mock.MockDiscordUserRepository, ctx context.Context) {
				m.EXPECT().GetDiscordUserByID(ctx, userID).
					Return(&domain.DiscordUser{ID: userID, CreatedAt: time.Now()}, nil)
			},
			wantErr: false,
		},
		{
			name: "not found",
			mockSetup: func(m *mock.MockDiscordUserRepository, ctx context.Context) {
				m.EXPECT().GetDiscordUserByID(ctx, userID).Return(nil, errors.New("not found"))
			},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := mock.NewMockDiscordUserRepository(ctrl)
			i := interactor.NewDiscordUserInteractor(m)

			ctx := t.Context()
			testCase.mockSetup(m, ctx)

			res, err := i.GetDiscordUserByID(ctx, userID)

			if testCase.wantErr {
				require.Error(t, err)
				assert.Nil(t, res)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, res)
			}
		})
	}

	t.Run("validation-empty-userid", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := mock.NewMockDiscordUserRepository(ctrl)
		i := interactor.NewDiscordUserInteractor(m)

		ctx := t.Context()

		_, err := i.GetDiscordUserByID(ctx, "")
		require.Error(t, err)
	})
}

func TestDiscordUserInteractor_ListDiscordUsers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		mockSetup func(*mock.MockDiscordUserRepository, context.Context)
		wantErr   bool
	}{
		{
			name: "success",
			mockSetup: func(m *mock.MockDiscordUserRepository, ctx context.Context) {
				m.EXPECT().ListDiscordUsers(ctx).
					Return([]*domain.DiscordUser{{ID: "u1", CreatedAt: time.Now()}}, nil)
			},
			wantErr: false,
		},
		{
			name: "failure",
			mockSetup: func(m *mock.MockDiscordUserRepository, ctx context.Context) {
				m.EXPECT().ListDiscordUsers(ctx).Return(nil, errors.New("list failed"))
			},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := mock.NewMockDiscordUserRepository(ctrl)
			i := interactor.NewDiscordUserInteractor(m)

			ctx := t.Context()
			testCase.mockSetup(m, ctx)

			res, err := i.ListDiscordUsers(ctx)

			if testCase.wantErr {
				require.Error(t, err)
				assert.Nil(t, res)
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, res)
			}
		})
	}
}

func TestDiscordUserInteractor_DeleteDiscordUser(t *testing.T) {
	t.Parallel()

	userID := "u1"

	tests := []struct {
		name      string
		mockSetup func(*mock.MockDiscordUserRepository, context.Context)
		wantErr   bool
	}{
		{
			name: "success",
			mockSetup: func(m *mock.MockDiscordUserRepository, ctx context.Context) {
				m.EXPECT().DeleteDiscordUser(ctx, userID).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "failure",
			mockSetup: func(m *mock.MockDiscordUserRepository, ctx context.Context) {
				m.EXPECT().DeleteDiscordUser(ctx, userID).Return(errors.New("delete failed"))
			},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := mock.NewMockDiscordUserRepository(ctrl)
			i := interactor.NewDiscordUserInteractor(m)

			ctx := t.Context()
			testCase.mockSetup(m, ctx)

			err := i.DeleteDiscordUser(ctx, userID)

			if testCase.wantErr {
				require.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}

	t.Run("validation-empty-userid", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := mock.NewMockDiscordUserRepository(ctrl)
		i := interactor.NewDiscordUserInteractor(m)

		ctx := t.Context()

		err := i.DeleteDiscordUser(ctx, "")
		require.Error(t, err)
	})
}
