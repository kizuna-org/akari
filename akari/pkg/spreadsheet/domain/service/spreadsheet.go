package service

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

type SpreadsheetService interface {
	Read(ctx context.Context, spreadsheetId, sheetName, range_ string) (*entity.GridData, error)
	Create(ctx context.Context, spreadsheetId, sheetName, range_ string, values [][]string) error
	Update(ctx context.Context, spreadsheetId, sheetName, range_ string, values [][]string) error
	Delete(ctx context.Context, spreadsheetId, sheetName, range_ string) error
	Clear(ctx context.Context, spreadsheetId, sheetName, range_ string) error
	Append(ctx context.Context, spreadsheetId, sheetName, range_ string, values [][]string) error
}

type SpreadsheetServiceImpl struct {
	repo SpreadsheetRepository
}

func NewSpreadsheetService(repo SpreadsheetRepository) SpreadsheetService {
	return &SpreadsheetServiceImpl{
		repo: repo,
	}
}

func (s *SpreadsheetServiceImpl) Read(ctx context.Context, spreadsheetId, sheetName, range_ string) (*entity.GridData, error) {
	cellRange := entity.CellRange{
		SheetData: entity.SheetData{
			SpreadsheetId: spreadsheetId,
			SheetName:     sheetName,
		},
		Range: range_,
	}

	return s.repo.Read(ctx, cellRange)
}

func (s *SpreadsheetServiceImpl) Create(ctx context.Context, spreadsheetId, sheetName, range_ string, values [][]string) error {
	gridData := &entity.GridData{
		CellRange: entity.CellRange{
			SheetData: entity.SheetData{
				SpreadsheetId: spreadsheetId,
				SheetName:     sheetName,
			},
			Range: range_,
		},
		Values: values,
	}

	return s.repo.Create(ctx, gridData)
}

func (s *SpreadsheetServiceImpl) Update(ctx context.Context, spreadsheetId, sheetName, range_ string, values [][]string) error {
	gridData := &entity.GridData{
		CellRange: entity.CellRange{
			SheetData: entity.SheetData{
				SpreadsheetId: spreadsheetId,
				SheetName:     sheetName,
			},
			Range: range_,
		},
		Values: values,
	}

	return s.repo.Update(ctx, gridData)
}

func (s *SpreadsheetServiceImpl) Delete(ctx context.Context, spreadsheetId, sheetName, range_ string) error {
	cellRange := entity.CellRange{
		SheetData: entity.SheetData{
			SpreadsheetId: spreadsheetId,
			SheetName:     sheetName,
		},
		Range: range_,
	}

	return s.repo.Delete(ctx, cellRange)
}

func (s *SpreadsheetServiceImpl) Clear(ctx context.Context, spreadsheetId, sheetName, range_ string) error {
	cellRange := entity.CellRange{
		SheetData: entity.SheetData{
			SpreadsheetId: spreadsheetId,
			SheetName:     sheetName,
		},
		Range: range_,
	}

	return s.repo.Clear(ctx, cellRange)
}

func (s *SpreadsheetServiceImpl) Append(ctx context.Context, spreadsheetId, sheetName, range_ string, values [][]string) error {
	gridData := &entity.GridData{
		CellRange: entity.CellRange{
			SheetData: entity.SheetData{
				SpreadsheetId: spreadsheetId,
				SheetName:     sheetName,
			},
			Range: range_,
		},
		Values: values,
	}

	return s.repo.Append(ctx, gridData)
}
