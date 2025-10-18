package interactor

import (
	"context"

	"github.com/kizuna-org/akari/pkg/spreadsheet/domain/entity"
	"github.com/kizuna-org/akari/pkg/spreadsheet/domain/repository"
	"github.com/kizuna-org/akari/pkg/spreadsheet/domain/service"
	"github.com/kizuna-org/akari/pkg/spreadsheet/usecase/dto"
)

type SpreadsheetInteractor struct {
	repo    repository.SpreadsheetRepository
	service *service.SpreadsheetService
}

func NewSpreadsheetInteractor(repo repository.SpreadsheetRepository) *SpreadsheetInteractor {
	return &SpreadsheetInteractor{
		repo:    repo,
		service: service.NewSpreadsheetService(),
	}
}

func (i *SpreadsheetInteractor) ReadGrid(ctx context.Context, req *dto.GridDataRequest) (*dto.GridDataResponse, error) {
	cellRange := entity.CellRange{
		SheetData: entity.SheetData{
			SpreadsheetId: req.SpreadsheetId,
			SheetName:     req.SheetName,
		},
		Range: req.Range,
	}

	gridData, err := i.repo.Read(ctx, cellRange)
	if err != nil {
		return nil, err
	}

	return &dto.GridDataResponse{
		SpreadsheetID: gridData.SpreadsheetId,
		SheetName:     gridData.SheetName,
		Range:         gridData.Range,
		Values:        gridData.Values,
	}, nil
}

func (i *SpreadsheetInteractor) Create(ctx context.Context, req *dto.GridDataRequest) error {
	gridData := &entity.GridData{
		CellRange: entity.CellRange{
			SheetData: entity.SheetData{
				SpreadsheetId: req.SpreadsheetId,
				SheetName:     req.SheetName,
			},
			Range: req.Range,
		},
		Values: req.Values,
	}

	if err := i.service.ValidateGridData(gridData); err != nil {
		return err
	}

	return i.repo.Create(ctx, gridData)
}

func (i *SpreadsheetInteractor) Update(ctx context.Context, req *dto.GridDataRequest) error {
	gridData := &entity.GridData{
		CellRange: entity.CellRange{
			SheetData: entity.SheetData{
				SpreadsheetId: req.SpreadsheetId,
				SheetName:     req.SheetName,
			},
			Range: req.Range,
		},
		Values: req.Values,
	}

	if err := i.service.ValidateGridData(gridData); err != nil {
		return err
	}

	return i.repo.Update(ctx, gridData)
}

func (i *SpreadsheetInteractor) Delete(ctx context.Context, req *dto.DeleteRequest) error {
	cellRange := entity.CellRange{
		SheetData: entity.SheetData{
			SpreadsheetId: req.SpreadsheetId,
			SheetName:     req.SheetName,
		},
		Range: req.Range,
	}

	return i.repo.Delete(ctx, cellRange)
}

func (i *SpreadsheetInteractor) Append(ctx context.Context, req *dto.GridDataRequest) error {
	gridData := &entity.GridData{
		CellRange: entity.CellRange{
			SheetData: entity.SheetData{
				SpreadsheetId: req.SpreadsheetId,
				SheetName:     req.SheetName,
			},
			Range: req.Range,
		},
		Values: req.Values,
	}

	if err := i.service.ValidateGridData(gridData); err != nil {
		return err
	}

	return i.repo.Append(ctx, gridData)
}
