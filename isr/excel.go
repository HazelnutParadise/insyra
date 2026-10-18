package isr

import "github.com/HazelnutParadise/insyra"

// Excel represents a sheet in an Excel file: a source for DT.From and a
// target for (*DataTable).To.
type Excel struct {
	FilePath   string
	SheetName  string // the sheet to read or write; writing defaults to "Sheet1"
	InputOpts  Excel_inOpts
	OutputOpts Excel_outOpts
}

type Excel_inOpts struct {
	FirstCol2RowNames bool
	FirstRow2ColNames bool
}

type Excel_outOpts struct {
	RowNames2FirstCol bool
	ColNames2FirstRow bool
	IfSheetExists     insyra.SheetExistsPolicy // zero value refuses to overwrite an existing sheet
}
