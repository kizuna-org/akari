package infrastructure

import (
	"context"
	"fmt"

	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

type SpreadsheetClient struct {
	service *sheets.Service
}

func NewSpreadsheetClient(ctx context.Context, opts ...option.ClientOption) (*SpreadsheetClient, error) {
	srv, err := sheets.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create spreadsheet service: %w", err)
	}

	return &SpreadsheetClient{
		service: srv,
	}, nil
}

func (c *SpreadsheetClient) Read(ctx context.Context, spreadsheetId, range_ string) ([][]string, error) {
	valueRange, err := c.service.Spreadsheets.Values.Get(spreadsheetId, range_).Context(ctx).Do()
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

func (c *SpreadsheetClient) Update(ctx context.Context, spreadsheetId, range_ string, values [][]string) error {
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

	_, err := c.service.Spreadsheets.Values.Update(spreadsheetId, range_, valueRange).
		ValueInputOption("RAW").
		Context(ctx).
		Do()

	if err != nil {
		return fmt.Errorf("failed to update values: %w", err)
	}

	return nil
}

func (c *SpreadsheetClient) Append(ctx context.Context, spreadsheetId, range_ string, values [][]string) error {
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

	_, err := c.service.Spreadsheets.Values.Append(spreadsheetId, range_, valueRange).
		ValueInputOption("RAW").
		InsertDataOption("INSERT_ROWS").
		Context(ctx).
		Do()

	if err != nil {
		return fmt.Errorf("failed to append values: %w", err)
	}

	return nil
}

func (c *SpreadsheetClient) Clear(ctx context.Context, spreadsheetId, range_ string) error {
	_, err := c.service.Spreadsheets.Values.Clear(spreadsheetId, range_, &sheets.ClearValuesRequest{}).
		Context(ctx).
		Do()

	if err != nil {
		return fmt.Errorf("failed to clear values: %w", err)
	}

	return nil
}

func (c *SpreadsheetClient) BatchGet(
	ctx context.Context,
	spreadsheetId string,
	ranges []string,
) (map[string][][]string, error) {
	resp, err := c.service.Spreadsheets.Values.BatchGet(spreadsheetId).
		Ranges(ranges...).
		Context(ctx).
		Do()

	if err != nil {
		return nil, fmt.Errorf("failed to batch get values: %w", err)
	}

	result := make(map[string][][]string)
	for _, valueRange := range resp.ValueRanges {
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
		result[valueRange.Range] = values
	}

	return result, nil
}

func (c *SpreadsheetClient) BatchUpdate(ctx context.Context, spreadsheetID string, data map[string][][]string) error {
	valueRanges := make([]*sheets.ValueRange, 0, len(data))

	for range_, values := range data {
		interfaceValues := make([][]any, len(values))
		for i, row := range values {
			interfaceValues[i] = make([]any, len(row))
			for j, cell := range row {
				interfaceValues[i][j] = cell
			}
		}

		valueRanges = append(valueRanges, &sheets.ValueRange{
			Range:          range_,
			Values:         interfaceValues,
			MajorDimension: "",
			ServerResponse: googleapi.ServerResponse{
				HTTPStatusCode: 0,
				Header:         nil,
			},
			ForceSendFields: nil,
			NullFields:      nil,
		})
	}

	batchUpdateRequest := &sheets.BatchUpdateValuesRequest{
		ValueInputOption:             "RAW",
		Data:                         valueRanges,
		IncludeValuesInResponse:      false,
		ResponseDateTimeRenderOption: "",
		ResponseValueRenderOption:    "",
		ForceSendFields:              nil,
		NullFields:                   nil,
	}

	_, err := c.service.Spreadsheets.Values.BatchUpdate(spreadsheetID, batchUpdateRequest).
		Context(ctx).
		Do()

	if err != nil {
		return fmt.Errorf("failed to batch update values: %w", err)
	}

	return nil
}
