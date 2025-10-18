package interactor_test

import (
	"context"
	"testing"

	"github.com/kizuna-org/akari/pkg/spreadsheet/domain/entity"
	"github.com/kizuna-org/akari/pkg/spreadsheet/usecase/dto"
	"github.com/kizuna-org/akari/pkg/spreadsheet/usecase/interactor"
)

type MockSpreadsheetRepository struct {
	gridData *entity.GridData
}

func NewMockSpreadsheetRepository() *MockSpreadsheetRepository {
	return &MockSpreadsheetRepository{
		gridData: nil,
	}
}

func (m *MockSpreadsheetRepository) Read(ctx context.Context, cellRange entity.CellRange) (*entity.GridData, error) {
	if m.gridData != nil {
		return m.gridData, nil
	}
	return &entity.GridData{
		CellRange: cellRange,
		Values:    [][]string{},
	}, nil
}

func (m *MockSpreadsheetRepository) Create(ctx context.Context, gridData *entity.GridData) error {
	m.gridData = gridData
	return nil
}

func (m *MockSpreadsheetRepository) Update(ctx context.Context, gridData *entity.GridData) error {
	m.gridData = gridData
	return nil
}

func (m *MockSpreadsheetRepository) Delete(ctx context.Context, cellRange entity.CellRange) error {
	m.gridData = nil
	return nil
}

func (m *MockSpreadsheetRepository) Clear(ctx context.Context, cellRange entity.CellRange) error {
	return nil
}

func (m *MockSpreadsheetRepository) Append(ctx context.Context, gridData *entity.GridData) error {
	if m.gridData == nil {
		m.gridData = gridData
	} else {
		m.gridData.Values = append(m.gridData.Values, gridData.Values...)
	}
	return nil
}

func TestCreateGrid(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	mockRepo := NewMockSpreadsheetRepository()
	interactor := interactor.NewSpreadsheetInteractor(mockRepo)

	t.Run("CreateGrid", func(t *testing.T) {
		t.Parallel()
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

		err := interactor.Create(ctx, req)
		if err != nil {
			t.Errorf("CreateGrid failed: %v", err)
		}

		if mockRepo.gridData == nil {
			t.Error("Grid data was not created")
		}
	})
}

func TestReadGrid(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	mockRepo := NewMockSpreadsheetRepository()
	interactor := interactor.NewSpreadsheetInteractor(mockRepo)

	t.Run("ReadGrid", func(t *testing.T) {
		t.Parallel()
		mockRepo.gridData = &entity.GridData{
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

		req := &dto.GridDataRequest{
			SpreadsheetId: "test-sheet-id",
			SheetName:     "Sheet1",
			Range:         "A1:C3",
			Values:        nil,
		}

		resp, err := interactor.ReadGrid(ctx, req)
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
	mockRepo := NewMockSpreadsheetRepository()
	interactor := interactor.NewSpreadsheetInteractor(mockRepo)

	t.Run("Update", func(t *testing.T) {
		t.Parallel()
		req := &dto.GridDataRequest{
			SpreadsheetId: "test-sheet-id",
			SheetName:     "Sheet1",
			Range:         "B2:B3",
			Values: [][]string{
				{"26"},
				{"31"},
			},
		}

		err := interactor.Update(ctx, req)
		if err != nil {
			t.Errorf("Update failed: %v", err)
		}
	})
}

func TestDelete(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	mockRepo := NewMockSpreadsheetRepository()
	interactor := interactor.NewSpreadsheetInteractor(mockRepo)

	t.Run("Delete", func(t *testing.T) {
		t.Parallel()
		req := &dto.DeleteRequest{
			SpreadsheetId: "test-sheet-id",
			SheetName:     "Sheet1",
			Range:         "A1:C3",
		}

		err := interactor.Delete(ctx, req)
		if err != nil {
			t.Errorf("Delete failed: %v", err)
		}

		if mockRepo.gridData != nil {
			t.Error("Grid data was not deleted")
		}
	})
}

func TestAppend(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	mockRepo := NewMockSpreadsheetRepository()
	interactor := interactor.NewSpreadsheetInteractor(mockRepo)

	t.Run("Append", func(t *testing.T) {
		t.Parallel()
		mockRepo.gridData = &entity.GridData{
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
			},
		}

		req := &dto.GridDataRequest{
			SpreadsheetId: "test-sheet-id",
			SheetName:     "Sheet1",
			Range:         "A:C",
			Values: [][]string{
				{"Bob", "30", "Osaka"},
			},
		}

		err := interactor.Append(ctx, req)
		if err != nil {
			t.Errorf("Append failed: %v", err)
		}

		if len(mockRepo.gridData.Values) != 3 {
			t.Errorf("Expected 3 rows after append, got %d", len(mockRepo.gridData.Values))
		}
	})
}
