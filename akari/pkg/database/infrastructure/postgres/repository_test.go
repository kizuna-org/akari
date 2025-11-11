package postgres_test

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/kizuna-org/akari/pkg/database/domain"
	"github.com/kizuna-org/akari/pkg/database/infrastructure/postgres"
	"github.com/kizuna-org/akari/pkg/database/infrastructure/postgres/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestNewRepository(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock.NewMockClient(ctrl)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	repo := postgres.NewRepository(mockClient, logger)

	assert.NotNil(t, repo)
}

func TestRepository_HealthCheck(t *testing.T) {
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
			mockErr: assert.AnError,
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockClient := mock.NewMockClient(ctrl)
			logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
			repo := postgres.NewRepository(mockClient, logger)

			ctx := t.Context()

			mockClient.EXPECT().Ping(ctx).Return(testCase.mockErr)

			err := repo.HealthCheck(ctx)

			if testCase.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "database health check failed")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRepository_WithTransaction(t *testing.T) {
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
			mockErr: assert.AnError,
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockClient := mock.NewMockClient(ctrl)
			logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
			repo := postgres.NewRepository(mockClient, logger)

			ctx := t.Context()

			mockClient.EXPECT().
				WithTx(gomock.Any(), gomock.Any()).
				DoAndReturn(func(ctx context.Context, fn domain.TxFunc) error {
					return testCase.mockErr
				})

			err := repo.WithTransaction(ctx, func(ctx context.Context, tx *domain.Tx) error {
				return nil
			})

			if testCase.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
