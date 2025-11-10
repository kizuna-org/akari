package interactor

import (
	"context"

	"github.com/kizuna-org/akari/pkg/database/domain"
)

type databaseInteractor interface {
	WithTransaction(ctx context.Context, fn domain.TxFunc) error
}

type databaseInteractorImpl struct {
	repository domain.DatabaseRepository
}

func NewDatabaseInteractor(repository domain.DatabaseRepository) databaseInteractor {
	return &databaseInteractorImpl{
		repository: repository,
	}
}

func (d *databaseInteractorImpl) WithTransaction(ctx context.Context, fn domain.TxFunc) error {
	return d.repository.WithTransaction(ctx, fn)
}
