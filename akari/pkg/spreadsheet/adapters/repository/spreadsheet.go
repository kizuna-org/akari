package repository

import (
	"context"
	"fmt"

	"github.com/kizuna-org/akari/pkg/spreadsheet/domain/entity"
	"github.com/kizuna-org/akari/pkg/spreadsheet/domain/repository"
	"github.com/kizuna-org/akari/pkg/spreadsheet/infrastructure"
)

type SpreadsheetRepository struct {
	client *infrastructure.SpreadsheetClient
}

func NewSpreadsheetRepository(client *infrastructure.SpreadsheetClient) repository.SpreadsheetRepository {
	return &SpreadsheetRepository{
		client: client,
	}
}

func (r *SpreadsheetRepository) Read(ctx context.Context, cellRange entity.CellRange) (*entity.GridData, error) {
	range_ := r.buildRangeString(cellRange.SheetName, cellRange.Range)

	values, err := r.client.Read(ctx, cellRange.SpreadsheetId, range_)
	if err != nil {
		return nil, fmt.Errorf("failed to read grid: %w", err)
	}

	return &entity.GridData{
		CellRange: cellRange,
		Values:    values,
	}, nil
}

func (r *SpreadsheetRepository) Create(ctx context.Context, gridData *entity.GridData) error {
	range_ := r.buildRangeString(gridData.SheetName, gridData.Range)

	if err := r.client.Update(ctx, gridData.SpreadsheetId, range_, gridData.Values); err != nil {
		return fmt.Errorf("failed to create grid: %w", err)
	}

	return nil
}

func (r *SpreadsheetRepository) Update(ctx context.Context, gridData *entity.GridData) error {
	range_ := r.buildRangeString(gridData.SheetName, gridData.Range)

	if err := r.client.Update(ctx, gridData.SpreadsheetId, range_, gridData.Values); err != nil {
		return fmt.Errorf("failed to update grid: %w", err)
	}

	return nil
}

func (r *SpreadsheetRepository) Delete(ctx context.Context, cellRange entity.CellRange) error {
	range_ := r.buildRangeString(cellRange.SheetName, cellRange.Range)

	if err := r.client.Clear(ctx, cellRange.SpreadsheetId, range_); err != nil {
		return fmt.Errorf("failed to delete grid: %w", err)
	}

	return nil
}

func (r *SpreadsheetRepository) Clear(ctx context.Context, cellRange entity.CellRange) error {
	range_ := r.buildRangeString(cellRange.SheetName, cellRange.Range)

	if err := r.client.Clear(ctx, cellRange.SpreadsheetId, range_); err != nil {
		return fmt.Errorf("failed to clear range: %w", err)
	}

	return nil
}

func (r *SpreadsheetRepository) Append(ctx context.Context, gridData *entity.GridData) error {
	range_ := r.buildRangeString(gridData.SheetName, gridData.Range)

	if err := r.client.Append(ctx, gridData.SpreadsheetId, range_, gridData.Values); err != nil {
		return fmt.Errorf("failed to append grid: %w", err)
	}

	return nil
}

func (r *SpreadsheetRepository) buildRangeString(sheetName, range_ string) string {
	if range_ == "" {
		return sheetName
	}
	return fmt.Sprintf("%s!%s", sheetName, range_)
}
