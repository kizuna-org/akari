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

func TestNewDiscordChannelInteractor(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := mock.NewMockDiscordChannelRepository(ctrl)
	i := interactor.NewDiscordChannelInteractor(m)

	assert.NotNil(t, i)
}

func TestDiscordChannelInteractor_CreateDiscordChannel(t *testing.T) {
	t.Parallel()

	params := &domain.DiscordChannel{
		ID:      "m1",
		Name:    "channel1",
		GuildID: "g1",
	}

	tests := []struct {
		name      string
		mockSetup func(*mock.MockDiscordChannelRepository, context.Context)
		wantErr   bool
	}{
		{
			name: "success",
			mockSetup: func(m *mock.MockDiscordChannelRepository, ctx context.Context) {
				m.EXPECT().CreateDiscordChannel(ctx, *params).Return(params, nil)
			},
			wantErr: false,
		},
		{
			name: "failure",
			mockSetup: func(m *mock.MockDiscordChannelRepository, ctx context.Context) {
				m.EXPECT().CreateDiscordChannel(ctx, *params).Return(nil, errors.New("create failed"))
			},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := mock.NewMockDiscordChannelRepository(ctrl)
			i := interactor.NewDiscordChannelInteractor(m)

			ctx := t.Context()
			testCase.mockSetup(m, ctx)

			res, err := i.CreateDiscordChannel(ctx, *params)

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

func TestDiscordChannelInteractor_GetDiscordChannelByID(t *testing.T) {
	t.Parallel()

	channelID := "m1"

	tests := []struct {
		name      string
		mockSetup func(*mock.MockDiscordChannelRepository, context.Context)
		wantErr   bool
	}{
		{
			name: "success",
			mockSetup: func(m *mock.MockDiscordChannelRepository, ctx context.Context) {
				m.EXPECT().GetDiscordChannelByID(ctx, channelID).
					Return(&domain.DiscordChannel{ID: channelID, CreatedAt: time.Now()}, nil)
			},
			wantErr: false,
		},
		{
			name: "not found",
			mockSetup: func(m *mock.MockDiscordChannelRepository, ctx context.Context) {
				m.EXPECT().GetDiscordChannelByID(ctx, channelID).Return(nil, errors.New("not found"))
			},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := mock.NewMockDiscordChannelRepository(ctrl)
			i := interactor.NewDiscordChannelInteractor(m)

			ctx := t.Context()
			testCase.mockSetup(m, ctx)

			res, err := i.GetDiscordChannelByID(ctx, channelID)

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

func TestDiscordChannelInteractor_GetDiscordChannelByMessageID(t *testing.T) {
	t.Parallel()

	messageID := "msg1"

	tests := []struct {
		name      string
		mockSetup func(*mock.MockDiscordChannelRepository, context.Context)
		wantErr   bool
	}{
		{
			name: "success",
			mockSetup: func(m *mock.MockDiscordChannelRepository, ctx context.Context) {
				m.EXPECT().GetDiscordChannelByMessageID(ctx, messageID).
					Return(&domain.DiscordChannel{ID: "m1", CreatedAt: time.Now()}, nil)
			},
			wantErr: false,
		},
		{
			name: "not found",
			mockSetup: func(m *mock.MockDiscordChannelRepository, ctx context.Context) {
				m.EXPECT().GetDiscordChannelByMessageID(ctx, messageID).Return(nil, errors.New("not found"))
			},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := mock.NewMockDiscordChannelRepository(ctrl)
			i := interactor.NewDiscordChannelInteractor(m)

			ctx := t.Context()
			testCase.mockSetup(m, ctx)

			res, err := i.GetDiscordChannelByMessageID(ctx, messageID)

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

func TestDiscordChannelInteractor_GetDiscordChannelsByGuildID(t *testing.T) {
	t.Parallel()

	guildID := "g1"

	tests := []struct {
		name      string
		mockSetup func(*mock.MockDiscordChannelRepository, context.Context)
		wantErr   bool
	}{
		{
			name: "success",
			mockSetup: func(m *mock.MockDiscordChannelRepository, ctx context.Context) {
				m.EXPECT().GetDiscordChannelsByGuildID(ctx, guildID).
					Return([]*domain.DiscordChannel{{ID: "m1", GuildID: guildID, CreatedAt: time.Now()}}, nil)
			},
			wantErr: false,
		},
		{
			name: "failure",
			mockSetup: func(m *mock.MockDiscordChannelRepository, ctx context.Context) {
				m.EXPECT().GetDiscordChannelsByGuildID(ctx, guildID).Return(nil, errors.New("list failed"))
			},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := mock.NewMockDiscordChannelRepository(ctrl)
			i := interactor.NewDiscordChannelInteractor(m)

			ctx := t.Context()
			testCase.mockSetup(m, ctx)

			res, err := i.GetDiscordChannelsByGuildID(ctx, guildID)

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

func TestDiscordDiscordChannelInteractor_DeleteDiscordDiscordChannel(t *testing.T) {
	t.Parallel()

	channelID := "m1"

	tests := []struct {
		name      string
		mockSetup func(*mock.MockDiscordChannelRepository, context.Context)
		wantErr   bool
	}{
		{
			name: "success",
			mockSetup: func(m *mock.MockDiscordChannelRepository, ctx context.Context) {
				m.EXPECT().DeleteDiscordChannel(ctx, channelID).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "failure",
			mockSetup: func(m *mock.MockDiscordChannelRepository, ctx context.Context) {
				m.EXPECT().DeleteDiscordChannel(ctx, channelID).Return(errors.New("delete failed"))
			},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := mock.NewMockDiscordChannelRepository(ctrl)
			i := interactor.NewDiscordChannelInteractor(m)

			ctx := t.Context()
			testCase.mockSetup(m, ctx)

			err := i.DeleteDiscordChannel(ctx, channelID)

			if testCase.wantErr {
				require.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
