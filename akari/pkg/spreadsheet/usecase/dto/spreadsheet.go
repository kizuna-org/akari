package dto

type GridDataRequest struct {
	SpreadsheetId string
	SheetName     string
	Range         string
	Values        [][]string
}

type GridDataResponse struct {
	SpreadsheetID string
	SheetName     string
	Range         string
	Values        [][]string
}

type DeleteRequest struct {
	SpreadsheetId string
	SheetName     string
	Range         string
}
