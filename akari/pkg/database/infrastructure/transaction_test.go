package infrastructure_test

import (
	"context"
	"testing"

	"github.com/kizuna-org/akari/gen/ent"
	"github.com/kizuna-org/akari/pkg/database/domain"
	"github.com/kizuna-org/akari/pkg/database/infrastructure/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestClient_WithTx(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		txFuncErr  error
		mockErr    error
		wantErr    bool
		errMessage string
	}{
		{
			name:       "success",
			txFuncErr:  nil,
			mockErr:    nil,
			wantErr:    false,
			errMessage: "",
		},
		{
			name:       "transaction_function_error",
			txFuncErr:  assert.AnError,
			mockErr:    nil,
			wantErr:    true,
			errMessage: "assert.AnError",
		},
		{
			name:       "start_transaction_error",
			txFuncErr:  nil,
			mockErr:    assert.AnError,
			wantErr:    true,
			errMessage: "assert.AnError",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockClient := mock.NewMockClient(ctrl)
			ctx := t.Context()

			txFunc := func(ctx context.Context, tx *domain.Tx) error {
				return testCase.txFuncErr
			}

			if testCase.mockErr != nil {
				mockClient.EXPECT().
					WithTx(ctx, gomock.Any()).
					Return(testCase.mockErr)
			} else {
				mockClient.EXPECT().
					WithTx(ctx, gomock.Any()).
					DoAndReturn(func(ctx context.Context, fn domain.TxFunc) error {
						err := fn(ctx, &ent.Tx{SystemPrompt: nil})

						return err
					})
			}

			err := mockClient.WithTx(ctx, txFunc)

			if testCase.wantErr {
				require.Error(t, err)

				if testCase.errMessage != "" {
					assert.Contains(t, err.Error(), testCase.errMessage)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
