package interactor

import (
	"context"

	"github.com/kizuna-org/akari/pkg/spreadsheet/domain"
)

type SpreadsheetInteractor interface {
	Read(spreadsheetId, range_ string) ([][]string, error)
	Update(spreadsheetId, range_ string, values [][]string) error
	Append(spreadsheetId, range_ string, values [][]string) error
	Clear(spreadsheetId, range_ string) error
}

type SpreadsheetInteractorImpl struct {
	spreadsheetRepository domain.SpreadsheetRepository
}

func NewSpreadsheetInteractor(
	spreadsheetRepository domain.SpreadsheetRepository,
) SpreadsheetInteractor {
	return &SpreadsheetInteractorImpl{
		spreadsheetRepository: spreadsheetRepository,
	}
}

func (s *SpreadsheetInteractorImpl) Read(spreadsheetId, range_ string) ([][]string, error) {
	return s.spreadsheetRepository.Read(
		context.Background(),
		spreadsheetId,
		range_,
	)
}

func (s *SpreadsheetInteractorImpl) Update(spreadsheetId, range_ string, values [][]string) error {
	return s.spreadsheetRepository.Update(
		context.Background(),
		spreadsheetId,
		range_,
		values,
	)
}

func (s *SpreadsheetInteractorImpl) Append(spreadsheetId, range_ string, values [][]string) error {
	return s.spreadsheetRepository.Append(
		context.Background(),
		spreadsheetId,
		range_,
		values,
	)
}

func (s *SpreadsheetInteractorImpl) Clear(spreadsheetId, range_ string) error {
	return s.spreadsheetRepository.Clear(
		context.Background(),
		spreadsheetId,
		range_,
	)
}
