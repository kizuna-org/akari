package service

import (
	"fmt"
	"strings"

	"github.com/kizuna-org/akari/pkg/spreadsheet/domain/entity"
)

type SpreadsheetService struct{}

func NewSpreadsheetService() *SpreadsheetService {
	return &SpreadsheetService{}
}

func (s *SpreadsheetService) ValidateGridData(gridData *entity.GridData) error {
	if gridData.SpreadsheetId == "" {
		return fmt.Errorf("spreadsheet ID is required")
	}
	if gridData.SheetName == "" {
		return fmt.Errorf("sheet name is required")
	}
	if len(gridData.Values) == 0 {
		return fmt.Errorf("values cannot be empty")
	}
	return nil
}

func (s *SpreadsheetService) BuildRange(sheetName, rangeStr string) string {
	if rangeStr == "" {
		return sheetName
	}
	return fmt.Sprintf("%s!%s", sheetName, rangeStr)
}

func (s *SpreadsheetService) ParseRange(fullRange string) (sheetName, rangeStr string) {
	parts := strings.Split(fullRange, "!")
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return fullRange, ""
}
