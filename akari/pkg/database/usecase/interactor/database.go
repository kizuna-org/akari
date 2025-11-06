package interactor

import (
	"context"

	"github.com/kizuna-org/akari/pkg/database/domain"
)

type DatabaseInteractor interface {
	GetClient() domain.Client
	Connect(ctx context.Context) error
	Disconnect() error
	HealthCheck(ctx context.Context) error
	WithTransaction(ctx context.Context, fn domain.TxFunc) error
}

type databaseInteractorImpl struct {
	repository domain.DatabaseRepository
}

func NewDatabaseInteractor(repository domain.DatabaseRepository) DatabaseInteractor {
	return &databaseInteractorImpl{
		repository: repository,
	}
}

func (d *databaseInteractorImpl) GetClient() domain.Client {
	return d.repository.GetClient()
}

func (d *databaseInteractorImpl) Connect(ctx context.Context) error {
	return d.repository.Connect(ctx)
}

func (d *databaseInteractorImpl) Disconnect() error {
	return d.repository.Disconnect()
}

func (d *databaseInteractorImpl) HealthCheck(ctx context.Context) error {
	return d.repository.HealthCheck(ctx)
}

func (d *databaseInteractorImpl) WithTransaction(ctx context.Context, fn domain.TxFunc) error {
	return d.repository.GetClient().WithTx(ctx, fn)
}
