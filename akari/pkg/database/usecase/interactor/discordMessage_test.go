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

func TestNewDiscordMessageInteractor(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := mock.NewMockDiscordMessageRepository(ctrl)
	i := interactor.NewDiscordMessageInteractor(m)

	assert.NotNil(t, i)
}

func TestDiscordMessageInteractor_CreateDiscordMessage(t *testing.T) {
	t.Parallel()

	params := &domain.DiscordMessage{
		ID:        "m1",
		ChannelID: "c1",
		AuthorID:  "a1",
		Content:   "hello",
		Timestamp: time.Now(),
	}

	tests := []struct {
		name      string
		mockSetup func(*mock.MockDiscordMessageRepository, context.Context)
		wantErr   bool
	}{
		{
			name: "success",
			mockSetup: func(m *mock.MockDiscordMessageRepository, ctx context.Context) {
				m.EXPECT().CreateDiscordMessage(ctx, *params).Return(params, nil)
			},
			wantErr: false,
		},
		{
			name: "failure",
			mockSetup: func(m *mock.MockDiscordMessageRepository, ctx context.Context) {
				m.EXPECT().CreateDiscordMessage(ctx, *params).Return(nil, errors.New("create failed"))
			},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := mock.NewMockDiscordMessageRepository(ctrl)
			i := interactor.NewDiscordMessageInteractor(m)

			ctx := t.Context()
			testCase.mockSetup(m, ctx)

			res, err := i.CreateDiscordMessage(ctx, *params)

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

func TestDiscordMessageInteractor_GetDiscordMessageByID(t *testing.T) {
	t.Parallel()

	messageID := "m1"

	tests := []struct {
		name      string
		mockSetup func(*mock.MockDiscordMessageRepository, context.Context)
		wantErr   bool
	}{
		{
			name: "success",
			mockSetup: func(m *mock.MockDiscordMessageRepository, ctx context.Context) {
				m.EXPECT().GetDiscordMessageByID(ctx, messageID).
					Return(&domain.DiscordMessage{ID: messageID, CreatedAt: time.Now()}, nil)
			},
			wantErr: false,
		},
		{
			name: "not found",
			mockSetup: func(m *mock.MockDiscordMessageRepository, ctx context.Context) {
				m.EXPECT().GetDiscordMessageByID(ctx, messageID).Return(nil, errors.New("not found"))
			},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := mock.NewMockDiscordMessageRepository(ctrl)
			i := interactor.NewDiscordMessageInteractor(m)

			ctx := t.Context()
			testCase.mockSetup(m, ctx)

			res, err := i.GetDiscordMessageByID(ctx, messageID)

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

func TestDiscordMessageInteractor_DeleteDiscordMessage(t *testing.T) {
	t.Parallel()

	messageID := "m1"

	tests := []struct {
		name      string
		mockSetup func(*mock.MockDiscordMessageRepository, context.Context)
		wantErr   bool
	}{
		{
			name: "success",
			mockSetup: func(m *mock.MockDiscordMessageRepository, ctx context.Context) {
				m.EXPECT().DeleteDiscordMessage(ctx, messageID).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "failure",
			mockSetup: func(m *mock.MockDiscordMessageRepository, ctx context.Context) {
				m.EXPECT().DeleteDiscordMessage(ctx, messageID).Return(errors.New("delete failed"))
			},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := mock.NewMockDiscordMessageRepository(ctrl)
			i := interactor.NewDiscordMessageInteractor(m)

			ctx := t.Context()
			testCase.mockSetup(m, ctx)

			err := i.DeleteDiscordMessage(ctx, messageID)

			if testCase.wantErr {
				require.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
