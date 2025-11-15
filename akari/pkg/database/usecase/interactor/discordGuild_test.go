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

func TestNewDiscordGuildInteractor(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := mock.NewMockDiscordGuildRepository(ctrl)
	i := interactor.NewDiscordGuildInteractor(m)

	assert.NotNil(t, i)
}

func TestDiscordGuildInteractor_CreateDiscordGuild(t *testing.T) {
	t.Parallel()

	params := &domain.DiscordGuild{
		ID:   "m1",
		Name: "guild1",
	}

	tests := []struct {
		name      string
		mockSetup func(*mock.MockDiscordGuildRepository, context.Context)
		wantErr   bool
	}{
		{
			name: "success",
			mockSetup: func(m *mock.MockDiscordGuildRepository, ctx context.Context) {
				m.EXPECT().CreateDiscordGuild(ctx, *params).Return(params, nil)
			},
			wantErr: false,
		},
		{
			name: "failure",
			mockSetup: func(m *mock.MockDiscordGuildRepository, ctx context.Context) {
				m.EXPECT().CreateDiscordGuild(ctx, *params).Return(nil, errors.New("create failed"))
			},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := mock.NewMockDiscordGuildRepository(ctrl)
			i := interactor.NewDiscordGuildInteractor(m)

			ctx := t.Context()
			testCase.mockSetup(m, ctx)

			res, err := i.CreateDiscordGuild(ctx, *params)

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

func TestDiscordGuildInteractor_GetDiscordGuildByID(t *testing.T) {
	t.Parallel()

	guildID := "m1"

	tests := []struct {
		name      string
		mockSetup func(*mock.MockDiscordGuildRepository, context.Context)
		wantErr   bool
	}{
		{
			name: "success",
			mockSetup: func(m *mock.MockDiscordGuildRepository, ctx context.Context) {
				m.EXPECT().GetDiscordGuildByID(ctx, guildID).
					Return(&domain.DiscordGuild{ID: guildID, CreatedAt: time.Now()}, nil)
			},
			wantErr: false,
		},
		{
			name: "not found",
			mockSetup: func(m *mock.MockDiscordGuildRepository, ctx context.Context) {
				m.EXPECT().GetDiscordGuildByID(ctx, guildID).Return(nil, errors.New("not found"))
			},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := mock.NewMockDiscordGuildRepository(ctrl)
			i := interactor.NewDiscordGuildInteractor(m)

			ctx := t.Context()
			testCase.mockSetup(m, ctx)

			res, err := i.GetDiscordGuildByID(ctx, guildID)

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

func TestDiscordGuildInteractor_GetDiscordGuildByChannelID(t *testing.T) {
	t.Parallel()

	channelID := "c1"

	tests := []struct {
		name      string
		mockSetup func(*mock.MockDiscordGuildRepository, context.Context)
		wantErr   bool
	}{
		{
			name: "success",
			mockSetup: func(m *mock.MockDiscordGuildRepository, ctx context.Context) {
				m.EXPECT().GetDiscordGuildByChannelID(ctx, channelID).
					Return(&domain.DiscordGuild{ID: "g1", CreatedAt: time.Now()}, nil)
			},
			wantErr: false,
		},
		{
			name: "not found",
			mockSetup: func(m *mock.MockDiscordGuildRepository, ctx context.Context) {
				m.EXPECT().GetDiscordGuildByChannelID(ctx, channelID).Return(nil, errors.New("not found"))
			},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := mock.NewMockDiscordGuildRepository(ctrl)
			i := interactor.NewDiscordGuildInteractor(m)

			ctx := t.Context()
			testCase.mockSetup(m, ctx)

			res, err := i.GetDiscordGuildByChannelID(ctx, channelID)

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

func TestDiscordGuildInteractor_ListDiscordGuilds(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		mockSetup func(*mock.MockDiscordGuildRepository, context.Context)
		wantErr   bool
	}{
		{
			name: "success",
			mockSetup: func(m *mock.MockDiscordGuildRepository, ctx context.Context) {
				m.EXPECT().ListDiscordGuilds(ctx).Return([]*domain.DiscordGuild{{ID: "m1", CreatedAt: time.Now()}}, nil)
			},
			wantErr: false,
		},
		{
			name: "failure",
			mockSetup: func(m *mock.MockDiscordGuildRepository, ctx context.Context) {
				m.EXPECT().ListDiscordGuilds(ctx).Return(nil, errors.New("list failed"))
			},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := mock.NewMockDiscordGuildRepository(ctrl)
			i := interactor.NewDiscordGuildInteractor(m)

			ctx := t.Context()
			testCase.mockSetup(m, ctx)

			res, err := i.ListDiscordGuilds(ctx)

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

func TestDiscordGuildInteractor_DeleteDiscordGuild(t *testing.T) {
	t.Parallel()

	guildID := "m1"

	tests := []struct {
		name      string
		mockSetup func(*mock.MockDiscordGuildRepository, context.Context)
		wantErr   bool
	}{
		{
			name: "success",
			mockSetup: func(m *mock.MockDiscordGuildRepository, ctx context.Context) {
				m.EXPECT().DeleteDiscordGuild(ctx, guildID).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "failure",
			mockSetup: func(m *mock.MockDiscordGuildRepository, ctx context.Context) {
				m.EXPECT().DeleteDiscordGuild(ctx, guildID).Return(errors.New("delete failed"))
			},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := mock.NewMockDiscordGuildRepository(ctrl)
			i := interactor.NewDiscordGuildInteractor(m)

			ctx := t.Context()
			testCase.mockSetup(m, ctx)

			err := i.DeleteDiscordGuild(ctx, guildID)

			if testCase.wantErr {
				require.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
