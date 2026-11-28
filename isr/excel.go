package isr

import "github.com/HazelnutParadise/insyra/excel"

// Excel represents a sheet in an Excel file: a source for DT.From and a
// target for (*DataTable).To.
type Excel struct {
	FilePath   string
	SheetName  string // the sheet to read or write; reading defaults to the first sheet, writing to "Sheet1"
	InputOpts  Excel_inOpts
	OutputOpts Excel_outOpts
}

type Excel_inOpts struct {
	NoHeaderRow bool // the first row is data, not column names
	HasRowNames bool // the first column holds row names
	RawStrings  bool // keep every cell as text; skip column type inference
}

type Excel_outOpts struct {
	NoHeaderRow   bool                    // leave out the row of column names
	HasRowNames   bool                    // write the row names as the first column
	IfSheetExists excel.SheetExistsPolicy // zero value refuses to overwrite an existing sheet
}
