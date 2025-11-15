package interactor_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kizuna-org/akari/gen/ent"
	"github.com/kizuna-org/akari/pkg/database/domain/mock"
	"github.com/kizuna-org/akari/pkg/database/usecase/interactor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestNewAkariUserInteractor(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := mock.NewMockAkariUserRepository(ctrl)
	i := interactor.NewAkariUserInteractor(m)

	assert.NotNil(t, i)
}

func TestAkariUserInteractor_CreateAkariUser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		mockSetup func(*mock.MockAkariUserRepository, context.Context)
		wantErr   bool
	}{
		{
			name: "success",
			mockSetup: func(m *mock.MockAkariUserRepository, ctx context.Context) {
				m.EXPECT().CreateAkariUser(ctx).Return(&ent.AkariUser{ID: 1, CreatedAt: time.Now()}, nil)
			},
			wantErr: false,
		},
		{
			name: "failure",
			mockSetup: func(m *mock.MockAkariUserRepository, ctx context.Context) {
				m.EXPECT().CreateAkariUser(ctx).Return(nil, errors.New("db error"))
			},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := mock.NewMockAkariUserRepository(ctrl)
			i := interactor.NewAkariUserInteractor(m)

			ctx := t.Context()
			testCase.mockSetup(m, ctx)

			res, err := i.CreateAkariUser(ctx)

			if testCase.wantErr {
				require.Error(t, err)
				assert.Nil(t, res)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, res)
			}
		})
	}
}

func TestAkariUserInteractor_GetAkariUserByID(t *testing.T) {
	t.Parallel()

	userId := 1

	tests := []struct {
		name      string
		mockSetup func(*mock.MockAkariUserRepository, context.Context)
		wantErr   bool
	}{
		{
			name: "success",
			mockSetup: func(m *mock.MockAkariUserRepository, ctx context.Context) {
				m.EXPECT().GetAkariUserByID(ctx, userId).Return(&ent.AkariUser{ID: userId, CreatedAt: time.Now()}, nil)
			},
			wantErr: false,
		},
		{
			name: "not found",
			mockSetup: func(m *mock.MockAkariUserRepository, ctx context.Context) {
				m.EXPECT().GetAkariUserByID(ctx, userId).Return(nil, errors.New("not found"))
			},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := mock.NewMockAkariUserRepository(ctrl)
			i := interactor.NewAkariUserInteractor(m)

			ctx := t.Context()
			testCase.mockSetup(m, ctx)

			res, err := i.GetAkariUserByID(ctx, userId)

			if testCase.wantErr {
				require.Error(t, err)
				assert.Nil(t, res)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, res)
				assert.Equal(t, userId, res.ID)
			}
		})
	}
}

func TestAkariUserInteractor_ListAkariUsers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		mockSetup func(*mock.MockAkariUserRepository, context.Context)
		wantErr   bool
	}{
		{
			name: "success",
			mockSetup: func(m *mock.MockAkariUserRepository, ctx context.Context) {
				m.EXPECT().ListAkariUsers(ctx).Return([]*ent.AkariUser{{ID: 1, CreatedAt: time.Now()}}, nil)
			},
			wantErr: false,
		},
		{
			name: "failure",
			mockSetup: func(m *mock.MockAkariUserRepository, ctx context.Context) {
				m.EXPECT().ListAkariUsers(ctx).Return(nil, errors.New("list error"))
			},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := mock.NewMockAkariUserRepository(ctrl)
			i := interactor.NewAkariUserInteractor(m)

			ctx := t.Context()
			testCase.mockSetup(m, ctx)

			res, err := i.ListAkariUsers(ctx)

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

func TestAkariUserInteractor_DeleteAkariUser(t *testing.T) {
	t.Parallel()

	userId := 1

	tests := []struct {
		name      string
		mockSetup func(*mock.MockAkariUserRepository, context.Context)
		wantErr   bool
	}{
		{
			name: "success",
			mockSetup: func(m *mock.MockAkariUserRepository, ctx context.Context) {
				m.EXPECT().DeleteAkariUser(ctx, userId).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "failure",
			mockSetup: func(m *mock.MockAkariUserRepository, ctx context.Context) {
				m.EXPECT().DeleteAkariUser(ctx, userId).Return(errors.New("delete failed"))
			},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := mock.NewMockAkariUserRepository(ctrl)
			i := interactor.NewAkariUserInteractor(m)

			ctx := t.Context()
			testCase.mockSetup(m, ctx)

			err := i.DeleteAkariUser(ctx, userId)

			if testCase.wantErr {
				require.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
