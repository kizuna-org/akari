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

func TestNewConversationGroupInteractor(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := mock.NewMockConversationGroupRepository(ctrl)
	i := interactor.NewConversationGroupInteractor(m)

	assert.NotNil(t, i)
}

func TestConversationGroupInteractor_CreateConversationGroup(t *testing.T) {
	t.Parallel()

	characterID := 1

	tests := []struct {
		name      string
		mockSetup func(*mock.MockConversationGroupRepository, context.Context)
		wantErr   bool
	}{
		{
			name: "success",
			mockSetup: func(m *mock.MockConversationGroupRepository, ctx context.Context) {
				m.EXPECT().CreateConversationGroup(ctx, characterID).Return(
					&domain.ConversationGroup{ID: 1, CreatedAt: time.Now()},
					nil,
				)
			},
			wantErr: false,
		},
		{
			name: "failure",
			mockSetup: func(m *mock.MockConversationGroupRepository, ctx context.Context) {
				m.EXPECT().CreateConversationGroup(ctx, characterID).Return(nil, errors.New("create failed"))
			},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := mock.NewMockConversationGroupRepository(ctrl)
			i := interactor.NewConversationGroupInteractor(m)

			ctx := t.Context()
			testCase.mockSetup(m, ctx)

			res, err := i.CreateConversationGroup(ctx, characterID)

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

func TestConversationGroupInteractor_GetConversationGroupByID(t *testing.T) {
	t.Parallel()

	conversationGroupID := 1

	tests := []struct {
		name      string
		mockSetup func(*mock.MockConversationGroupRepository, context.Context)
		wantErr   bool
	}{
		{
			name: "success",
			mockSetup: func(m *mock.MockConversationGroupRepository, ctx context.Context) {
				m.EXPECT().GetConversationGroupByID(
					ctx,
					conversationGroupID,
				).Return(&domain.ConversationGroup{ID: conversationGroupID, CreatedAt: time.Now()}, nil)
			},
			wantErr: false,
		},
		{
			name: "not found",
			mockSetup: func(m *mock.MockConversationGroupRepository, ctx context.Context) {
				m.EXPECT().GetConversationGroupByID(ctx, conversationGroupID).Return(nil, errors.New("not found"))
			},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := mock.NewMockConversationGroupRepository(ctrl)
			i := interactor.NewConversationGroupInteractor(m)

			ctx := t.Context()
			testCase.mockSetup(m, ctx)

			res, err := i.GetConversationGroupByID(ctx, conversationGroupID)

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

func TestConversationGroupInteractor_GetConversationGroupByCharacterID(t *testing.T) {
	t.Parallel()

	characterID := 1

	tests := []struct {
		name      string
		mockSetup func(*mock.MockConversationGroupRepository, context.Context)
		wantErr   bool
	}{
		{
			name: "success",
			mockSetup: func(m *mock.MockConversationGroupRepository, ctx context.Context) {
				m.EXPECT().GetConversationGroupByCharacterID(ctx, characterID).Return(
					&domain.ConversationGroup{ID: 1, CreatedAt: time.Now()},
					nil,
				)
			},
			wantErr: false,
		},
		{
			name: "not found",
			mockSetup: func(m *mock.MockConversationGroupRepository, ctx context.Context) {
				m.EXPECT().GetConversationGroupByCharacterID(ctx, characterID).Return(nil, errors.New("not found"))
			},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := mock.NewMockConversationGroupRepository(ctrl)
			i := interactor.NewConversationGroupInteractor(m)

			ctx := t.Context()
			testCase.mockSetup(m, ctx)

			res, err := i.GetConversationGroupByCharacterID(ctx, characterID)

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

func TestConversationGroupInteractor_ListConversationGroups(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		mockSetup func(*mock.MockConversationGroupRepository, context.Context)
		wantErr   bool
	}{
		{
			name: "success",
			mockSetup: func(m *mock.MockConversationGroupRepository, ctx context.Context) {
				m.EXPECT().ListConversationGroups(ctx).Return([]*domain.ConversationGroup{{ID: 1, CreatedAt: time.Now()}}, nil)
			},
			wantErr: false,
		},
		{
			name: "failure",
			mockSetup: func(m *mock.MockConversationGroupRepository, ctx context.Context) {
				m.EXPECT().ListConversationGroups(ctx).Return(nil, errors.New("list failed"))
			},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := mock.NewMockConversationGroupRepository(ctrl)
			i := interactor.NewConversationGroupInteractor(m)

			ctx := t.Context()
			testCase.mockSetup(m, ctx)

			res, err := i.ListConversationGroups(ctx)

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

func TestConversationGroupInteractor_DeleteConversationGroup(t *testing.T) {
	t.Parallel()

	conversationGroupID := 1

	tests := []struct {
		name      string
		mockSetup func(*mock.MockConversationGroupRepository, context.Context)
		wantErr   bool
	}{
		{
			name: "success",
			mockSetup: func(m *mock.MockConversationGroupRepository, ctx context.Context) {
				m.EXPECT().DeleteConversationGroup(ctx, conversationGroupID).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "failure",
			mockSetup: func(m *mock.MockConversationGroupRepository, ctx context.Context) {
				m.EXPECT().DeleteConversationGroup(ctx, conversationGroupID).Return(errors.New("delete failed"))
			},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := mock.NewMockConversationGroupRepository(ctrl)
			i := interactor.NewConversationGroupInteractor(m)

			ctx := t.Context()
			testCase.mockSetup(m, ctx)

			err := i.DeleteConversationGroup(ctx, conversationGroupID)

			if testCase.wantErr {
				require.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
