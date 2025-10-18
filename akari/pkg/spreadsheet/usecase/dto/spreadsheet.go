package dto

type GridDataRequest struct {
	SpreadsheetId string     `json:"spreadsheet_id"`
	SheetName     string     `json:"sheet_name"`
	Range         string     `json:"range,omitempty"`
	Values        [][]string `json:"values,omitempty"`
}

type GridDataResponse struct {
	SpreadsheetID string     `json:"spreadsheet_id"`
	SheetName     string     `json:"sheet_name"`
	Range         string     `json:"range"`
	Values        [][]string `json:"values"`
}

type DeleteRequest struct {
	SpreadsheetId string `json:"spreadsheet_id"`
	SheetName     string `json:"sheet_name"`
	Range         string `json:"range"`
}
