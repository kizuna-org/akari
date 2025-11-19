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

func TestNewConversationInteractor(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := mock.NewMockConversationRepository(ctrl)
	i := interactor.NewConversationInteractor(m)

	assert.NotNil(t, i)
}

func TestConversationInteractor_CreateConversation(t *testing.T) {
	t.Parallel()

	messageID := "message123"
	groupID := 1

	tests := []struct {
		name      string
		mockSetup func(*mock.MockConversationRepository, context.Context)
		wantErr   bool
	}{
		{
			name: "success",
			mockSetup: func(m *mock.MockConversationRepository, ctx context.Context) {
				m.EXPECT().CreateConversation(ctx, messageID, &groupID).Return(&domain.Conversation{
					ID:        1,
					CreatedAt: time.Now(),
				}, nil)
			},
			wantErr: false,
		},
		{
			name: "failure",
			mockSetup: func(m *mock.MockConversationRepository, ctx context.Context) {
				m.EXPECT().CreateConversation(ctx, messageID, &groupID).Return(nil, errors.New("db error"))
			},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := mock.NewMockConversationRepository(ctrl)
			i := interactor.NewConversationInteractor(m)

			ctx := t.Context()
			testCase.mockSetup(m, ctx)

			res, err := i.CreateConversation(ctx, messageID, &groupID)

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

func TestConversationInteractor_GetConversationByID(t *testing.T) {
	t.Parallel()

	conversationID := 1

	tests := []struct {
		name      string
		mockSetup func(*mock.MockConversationRepository, context.Context)
		wantErr   bool
	}{
		{
			name: "success",
			mockSetup: func(m *mock.MockConversationRepository, ctx context.Context) {
				m.EXPECT().GetConversationByID(
					ctx,
					conversationID,
				).Return(&domain.Conversation{ID: conversationID, CreatedAt: time.Now()}, nil)
			},
			wantErr: false,
		},
		{
			name: "not found",
			mockSetup: func(m *mock.MockConversationRepository, ctx context.Context) {
				m.EXPECT().GetConversationByID(ctx, conversationID).Return(nil, errors.New("not found"))
			},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := mock.NewMockConversationRepository(ctrl)
			i := interactor.NewConversationInteractor(m)

			ctx := t.Context()
			testCase.mockSetup(m, ctx)

			res, err := i.GetConversationByID(ctx, conversationID)

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

func TestConversationInteractor_ListConversations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		mockSetup func(*mock.MockConversationRepository, context.Context)
		wantErr   bool
	}{
		{
			name: "success",
			mockSetup: func(m *mock.MockConversationRepository, ctx context.Context) {
				m.EXPECT().ListConversations(ctx).Return([]*domain.Conversation{{ID: 1, CreatedAt: time.Now()}}, nil)
			},
			wantErr: false,
		},
		{
			name: "failure",
			mockSetup: func(m *mock.MockConversationRepository, ctx context.Context) {
				m.EXPECT().ListConversations(ctx).Return(nil, errors.New("list error"))
			},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := mock.NewMockConversationRepository(ctrl)
			i := interactor.NewConversationInteractor(m)

			ctx := t.Context()
			testCase.mockSetup(m, ctx)

			res, err := i.ListConversations(ctx)

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

func TestConversationInteractor_DeleteConversation(t *testing.T) {
	t.Parallel()

	conversationID := 1

	tests := []struct {
		name      string
		mockSetup func(*mock.MockConversationRepository, context.Context)
		wantErr   bool
	}{
		{
			name: "success",
			mockSetup: func(m *mock.MockConversationRepository, ctx context.Context) {
				m.EXPECT().DeleteConversation(ctx, conversationID).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "failure",
			mockSetup: func(m *mock.MockConversationRepository, ctx context.Context) {
				m.EXPECT().DeleteConversation(ctx, conversationID).Return(errors.New("delete failed"))
			},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := mock.NewMockConversationRepository(ctrl)
			i := interactor.NewConversationInteractor(m)

			ctx := t.Context()
			testCase.mockSetup(m, ctx)

			err := i.DeleteConversation(ctx, conversationID)

			if testCase.wantErr {
				require.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
