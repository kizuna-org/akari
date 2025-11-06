package postgres

import (
	"context"
	"fmt"

	"github.com/kizuna-org/akari/pkg/database/domain"
)

func (c *client) WithTx(ctx context.Context, txFunc domain.TxFunc) error {
	transaction, err := c.Tx(ctx)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}

	defer func() {
		if v := recover(); v != nil {
			if err := transaction.Rollback(); err != nil {
				panic(fmt.Sprintf("failed to rollback transaction after panic: %v (original panic: %v)", err, v))
			}

			panic(v)
		}
	}()

	if err := txFunc(ctx, transaction); err != nil {
		if rerr := transaction.Rollback(); rerr != nil {
			return fmt.Errorf("failed to rollback transaction: %w (original error: %w)", rerr, err)
		}

		return err
	}

	if err := transaction.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
