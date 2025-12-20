package isr

// Excel represents a sheet in an Excel file.
type Excel struct {
	FilePath  string
	SheetName string
	InputOpts Excel_inOpts
}

type Excel_inOpts struct {
	FirstCol2RowNames bool
	FirstRow2ColNames bool
}
