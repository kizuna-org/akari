package repository

import (
	"context"

	"github.com/kizuna-org/akari/pkg/spreadsheet/domain/entity"
)

type SpreadsheetRepository interface {
	Read(ctx context.Context, cellRange entity.CellRange) (*entity.GridData, error)
	Create(ctx context.Context, gridData *entity.GridData) error
	Update(ctx context.Context, gridData *entity.GridData) error
	Delete(ctx context.Context, cellRange entity.CellRange) error

	Clear(ctx context.Context, cellRange entity.CellRange) error
	Append(ctx context.Context, gridData *entity.GridData) error
}
