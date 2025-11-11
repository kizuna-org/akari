package interactor_test

import (
	"context"
	"errors"
	"testing"

	"github.com/kizuna-org/akari/gen/ent"
	"github.com/kizuna-org/akari/pkg/database/domain"
	"github.com/kizuna-org/akari/pkg/database/domain/mock"
	"github.com/kizuna-org/akari/pkg/database/usecase/interactor"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestNewDatabaseInteractor(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock.NewMockDatabaseRepository(ctrl)
	interactor := interactor.NewDatabaseInteractor(mockRepo)

	assert.NotNil(t, interactor)
}

func TestDatabaseInteractor_WithTransaction(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mockErr error
		wantErr bool
	}{
		{
			name:    "success",
			mockErr: nil,
			wantErr: false,
		},
		{
			name:    "failure",
			mockErr: errors.New("transaction failed"),
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := mock.NewMockDatabaseRepository(ctrl)
			interactor := interactor.NewDatabaseInteractor(mockRepo)

			ctx := t.Context()

			if testCase.mockErr == nil {
				mockRepo.EXPECT().
					WithTransaction(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, fn domain.TxFunc) error {
						return fn(ctx, &ent.Tx{SystemPrompt: nil})
					})
			} else {
				mockRepo.EXPECT().
					WithTransaction(gomock.Any(), gomock.Any()).
					Return(testCase.mockErr)
			}

			err := interactor.WithTransaction(ctx, func(ctx context.Context, tx *domain.Tx) error {
				return testCase.mockErr
			})

			if testCase.wantErr {
				assert.Equal(t, testCase.mockErr, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
