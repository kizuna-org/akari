package dto

type GridDataRequest struct {
	SpreadsheetId string     `json:"spreadsheetId"`
	SheetName     string     `json:"sheetName"`
	Range         string     `json:"range,omitempty"`
	Values        [][]string `json:"values,omitempty"`
}

type GridDataResponse struct {
	SpreadsheetID string     `json:"spreadsheetId"`
	SheetName     string     `json:"sheetName"`
	Range         string     `json:"range"`
	Values        [][]string `json:"values"`
}

type DeleteRequest struct {
	SpreadsheetId string `json:"spreadsheetId"`
	SheetName     string `json:"sheetName"`
	Range         string `json:"range"`
}
