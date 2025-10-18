package interactor_test

import (
	"context"
	"testing"

	"github.com/kizuna-org/akari/pkg/spreadsheet/domain/entity"
	"github.com/kizuna-org/akari/pkg/spreadsheet/usecase/dto"
	"github.com/kizuna-org/akari/pkg/spreadsheet/usecase/interactor"
	"go.uber.org/mock/gomock"
)

func TestCreateGrid(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("CreateGrid", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := NewMockSpreadsheetRepository(ctrl)
		interactor := interactor.NewSpreadsheetInteractor(mockRepo)

		req := &dto.GridDataRequest{
			SpreadsheetId: "test-sheet-id",
			SheetName:     "Sheet1",
			Range:         "A1:C3",
			Values: [][]string{
				{"Name", "Age", "City"},
				{"Alice", "25", "Tokyo"},
				{"Bob", "30", "Osaka"},
			},
		}

		expectedGridData := &entity.GridData{
			CellRange: entity.CellRange{
				SheetData: entity.SheetData{
					SpreadsheetId: "test-sheet-id",
					SheetName:     "Sheet1",
				},
				Range: "A1:C3",
			},
			Values: [][]string{
				{"Name", "Age", "City"},
				{"Alice", "25", "Tokyo"},
				{"Bob", "30", "Osaka"},
			},
		}

		mockRepo.EXPECT().Create(ctx, expectedGridData).Return(nil)

		err := interactor.Create(ctx, req)
		if err != nil {
			t.Errorf("CreateGrid failed: %v", err)
		}
	})
}

func TestReadGrid(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("ReadGrid", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := NewMockSpreadsheetRepository(ctrl)
		interactor := interactor.NewSpreadsheetInteractor(mockRepo)

		expectedCellRange := entity.CellRange{
			SheetData: entity.SheetData{
				SpreadsheetId: "test-sheet-id",
				SheetName:     "Sheet1",
			},
			Range: "A1:C3",
		}

		expectedGridData := &entity.GridData{
			CellRange: expectedCellRange,
			Values: [][]string{
				{"Name", "Age", "City"},
				{"Alice", "25", "Tokyo"},
				{"Bob", "30", "Osaka"},
			},
		}

		mockRepo.EXPECT().Read(ctx, expectedCellRange).Return(expectedGridData, nil)

		req := &dto.GridDataRequest{
			SpreadsheetId: "test-sheet-id",
			SheetName:     "Sheet1",
			Range:         "A1:C3",
			Values:        nil,
		}

		resp, err := interactor.Read(ctx, req)
		if err != nil {
			t.Errorf("Read failed: %v", err)
		}

		if len(resp.Values) != 3 {
			t.Errorf("Expected 3 rows, got %d", len(resp.Values))
		}
	})
}

func TestUpdate(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("Update", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := NewMockSpreadsheetRepository(ctrl)
		interactor := interactor.NewSpreadsheetInteractor(mockRepo)

		req := &dto.GridDataRequest{
			SpreadsheetId: "test-sheet-id",
			SheetName:     "Sheet1",
			Range:         "B2:B3",
			Values: [][]string{
				{"26"},
				{"31"},
			},
		}

		expectedGridData := &entity.GridData{
			CellRange: entity.CellRange{
				SheetData: entity.SheetData{
					SpreadsheetId: "test-sheet-id",
					SheetName:     "Sheet1",
				},
				Range: "B2:B3",
			},
			Values: [][]string{
				{"26"},
				{"31"},
			},
		}

		mockRepo.EXPECT().Update(ctx, expectedGridData).Return(nil)

		err := interactor.Update(ctx, req)
		if err != nil {
			t.Errorf("Update failed: %v", err)
		}
	})
}

func TestDelete(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("Delete", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := NewMockSpreadsheetRepository(ctrl)
		interactor := interactor.NewSpreadsheetInteractor(mockRepo)

		req := &dto.DeleteRequest{
			SpreadsheetId: "test-sheet-id",
			SheetName:     "Sheet1",
			Range:         "A1:C3",
		}

		expectedCellRange := entity.CellRange{
			SheetData: entity.SheetData{
				SpreadsheetId: "test-sheet-id",
				SheetName:     "Sheet1",
			},
			Range: "A1:C3",
		}

		mockRepo.EXPECT().Delete(ctx, expectedCellRange).Return(nil)

		err := interactor.Delete(ctx, req)
		if err != nil {
			t.Errorf("Delete failed: %v", err)
		}
	})
}

func TestAppend(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("Append", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := NewMockSpreadsheetRepository(ctrl)
		interactor := interactor.NewSpreadsheetInteractor(mockRepo)

		req := &dto.GridDataRequest{
			SpreadsheetId: "test-sheet-id",
			SheetName:     "Sheet1",
			Range:         "A:C",
			Values: [][]string{
				{"Bob", "30", "Osaka"},
			},
		}

		expectedGridData := &entity.GridData{
			CellRange: entity.CellRange{
				SheetData: entity.SheetData{
					SpreadsheetId: "test-sheet-id",
					SheetName:     "Sheet1",
				},
				Range: "A:C",
			},
			Values: [][]string{
				{"Bob", "30", "Osaka"},
			},
		}

		mockRepo.EXPECT().Append(ctx, expectedGridData).Return(nil)

		err := interactor.Append(ctx, req)
		if err != nil {
			t.Errorf("Append failed: %v", err)
		}
	})
}
