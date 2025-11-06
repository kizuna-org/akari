package domain

import (
	"context"
)

type SpreadsheetRepository interface {
	Read(ctx context.Context, spreadsheetId, range_ string) ([][]string, error)
	Update(ctx context.Context, spreadsheetId, range_ string, values [][]string) error
	Append(ctx context.Context, spreadsheetId, range_ string, values [][]string) error
	Clear(ctx context.Context, spreadsheetId, range_ string) error
}
