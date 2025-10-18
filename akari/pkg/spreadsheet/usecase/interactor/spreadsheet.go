package interactor

import (
	"context"
	"fmt"

	"github.com/kizuna-org/akari/pkg/spreadsheet/domain/entity"
	"github.com/kizuna-org/akari/pkg/spreadsheet/domain/service"
	"github.com/kizuna-org/akari/pkg/spreadsheet/usecase/dto"
)

type SpreadsheetInteractor interface {
	Read(ctx context.Context, req *dto.GridDataRequest) (*dto.GridDataResponse, error)
	Create(ctx context.Context, req *dto.GridDataRequest) error
	Update(ctx context.Context, req *dto.GridDataRequest) error
	Delete(ctx context.Context, req *dto.DeleteRequest) error
	Append(ctx context.Context, req *dto.GridDataRequest) error
}

type SpreadsheetInteractorImpl struct {
	repo service.SpreadsheetRepository
}

func NewSpreadsheetInteractor(repo service.SpreadsheetRepository) SpreadsheetInteractor {
	return &SpreadsheetInteractorImpl{
		repo: repo,
	}
}

func (i *SpreadsheetInteractorImpl) Read(ctx context.Context, req *dto.GridDataRequest) (*dto.GridDataResponse, error) {
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

func (i *SpreadsheetInteractorImpl) Create(ctx context.Context, req *dto.GridDataRequest) error {
	if err := i.validateGridDataRequest(req); err != nil {
		return err
	}

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

	return i.repo.Create(ctx, gridData)
}

func (i *SpreadsheetInteractorImpl) Update(ctx context.Context, req *dto.GridDataRequest) error {
	if err := i.validateGridDataRequest(req); err != nil {
		return err
	}

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

	return i.repo.Update(ctx, gridData)
}

func (i *SpreadsheetInteractorImpl) Delete(ctx context.Context, req *dto.DeleteRequest) error {
	cellRange := entity.CellRange{
		SheetData: entity.SheetData{
			SpreadsheetId: req.SpreadsheetId,
			SheetName:     req.SheetName,
		},
		Range: req.Range,
	}

	return i.repo.Delete(ctx, cellRange)
}

func (i *SpreadsheetInteractorImpl) Append(ctx context.Context, req *dto.GridDataRequest) error {
	if err := i.validateGridDataRequest(req); err != nil {
		return err
	}

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

	return i.repo.Append(ctx, gridData)
}

func (i *SpreadsheetInteractorImpl) validateGridDataRequest(req *dto.GridDataRequest) error {
	if req.SpreadsheetId == "" {
		return fmt.Errorf("spreadsheet ID is required")
	}
	if req.SheetName == "" {
		return fmt.Errorf("sheet name is required")
	}
	if len(req.Values) == 0 {
		return fmt.Errorf("values cannot be empty")
	}
	return nil
}
