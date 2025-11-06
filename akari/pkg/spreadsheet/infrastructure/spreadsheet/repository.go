package infrastructure

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/kizuna-org/akari/pkg/spreadsheet/domain"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/sheets/v4"
)

type repositoryImpl struct {
	service *sheets.Service
	logger  *slog.Logger
}

func NewRepository(clientOption domain.ClientOption, logger *slog.Logger) (domain.SpreadsheetRepository, error) {
	ctx := context.Background()

	srv, err := sheets.NewService(ctx, clientOption)
	if err != nil {
		return nil, fmt.Errorf("failed to create spreadsheet service: %w", err)
	}

	return &repositoryImpl{
		service: srv,
		logger:  logger.With("component", "spreadsheet_repository"),
	}, nil
}

func (r *repositoryImpl) Read(ctx context.Context, spreadsheetId, range_ string) ([][]string, error) {
	valueRange, err := r.service.Spreadsheets.Values.Get(spreadsheetId, range_).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to read values: %w", err)
	}

	values := make([][]string, len(valueRange.Values))
	for i, row := range valueRange.Values {
		values[i] = make([]string, len(row))

		for j, cell := range row {
			if cell != nil {
				values[i][j] = fmt.Sprintf("%v", cell)
			} else {
				values[i][j] = ""
			}
		}
	}

	return values, nil
}

func (r *repositoryImpl) Update(ctx context.Context, spreadsheetId, range_ string, values [][]string) error {
	interfaceValues := make([][]any, len(values))
	for i, row := range values {
		interfaceValues[i] = make([]any, len(row))
		for j, cell := range row {
			interfaceValues[i][j] = cell
		}
	}

	valueRange := &sheets.ValueRange{
		Values:         interfaceValues,
		MajorDimension: "",
		Range:          "",
		ServerResponse: googleapi.ServerResponse{
			HTTPStatusCode: 0,
			Header:         nil,
		},
		ForceSendFields: nil,
		NullFields:      nil,
	}

	_, err := r.service.Spreadsheets.Values.Update(spreadsheetId, range_, valueRange).
		ValueInputOption("RAW").
		Context(ctx).
		Do()
	if err != nil {
		return fmt.Errorf("failed to update values: %w", err)
	}

	return nil
}

func (r *repositoryImpl) Append(ctx context.Context, spreadsheetId, range_ string, values [][]string) error {
	interfaceValues := make([][]any, len(values))
	for i, row := range values {
		interfaceValues[i] = make([]any, len(row))
		for j, cell := range row {
			interfaceValues[i][j] = cell
		}
	}

	valueRange := &sheets.ValueRange{
		Values:         interfaceValues,
		MajorDimension: "",
		Range:          "",
		ServerResponse: googleapi.ServerResponse{
			HTTPStatusCode: 0,
			Header:         nil,
		},
		ForceSendFields: nil,
		NullFields:      nil,
	}

	_, err := r.service.Spreadsheets.Values.Append(spreadsheetId, range_, valueRange).
		ValueInputOption("RAW").
		InsertDataOption("INSERT_ROWS").
		Context(ctx).
		Do()
	if err != nil {
		return fmt.Errorf("failed to append values: %w", err)
	}

	return nil
}

func (r *repositoryImpl) Clear(ctx context.Context, spreadsheetId, range_ string) error {
	_, err := r.service.Spreadsheets.Values.Clear(spreadsheetId, range_, &sheets.ClearValuesRequest{}).
		Context(ctx).
		Do()
	if err != nil {
		return fmt.Errorf("failed to clear values: %w", err)
	}

	return nil
}
