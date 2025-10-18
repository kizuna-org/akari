package entity

type SheetData struct {
	SpreadsheetId string
	SheetName     string
}

type CellRange struct {
	SheetData
	Range string
}

type GridData struct {
	CellRange
	Values [][]string
}
