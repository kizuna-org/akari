//go:build integration

package repository_test

import (
	"context"
	"errors"
	"testing"

	"github.com/kizuna-org/akari/pkg/database/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepository_HealthCheck_Integration(t *testing.T) {
	t.Parallel()

	_, repo, _ := setupTestDB(t)

	ctx := context.Background()

	err := repo.HealthCheck(ctx)
	assert.NoError(t, err)
}

func TestRepository_WithTransaction_Integration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		fn      func(ctx context.Context, tx *domain.Tx) error
		wantErr bool
	}{
		{
			name: "success",
			fn: func(ctx context.Context, tx *domain.Tx) error {
				return nil
			},
			wantErr: false,
		},
		{
			name: "failure with rollback",
			fn: func(ctx context.Context, tx *domain.Tx) error {
				return errors.New("transaction error")
			},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			_, repo, _ := setupTestDB(t)

			ctx := context.Background()

			err := repo.WithTransaction(ctx, testCase.fn)

			if testCase.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "transaction error")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
