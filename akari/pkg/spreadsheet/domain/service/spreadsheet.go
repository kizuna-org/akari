package service

import (
	"fmt"
	"strings"

	"github.com/kizuna-org/akari/pkg/spreadsheet/domain/entity"
)

const rangePartsCount = 2

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

func (s *SpreadsheetService) BuildRange(sheetName, range_ string) string {
	if range_ == "" {
		return sheetName
	}
	return fmt.Sprintf("%s!%s", sheetName, range_)
}

func (s *SpreadsheetService) ParseRange(fullRange string) (string, string) {
	parts := strings.Split(fullRange, "!")
	if len(parts) == rangePartsCount {
		return parts[0], parts[1]
	}
	return fullRange, ""
}
