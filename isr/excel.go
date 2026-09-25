package isr

// Excel represents a sheet in an Excel file.
type Excel struct {
	FilePath  string
	SheetName string
	InputOpts Excel_inOpts
}

type Excel_inOpts struct {
	NoHeaderRow bool // the first row is data, not column names
	HasRowNames bool // the first column holds row names
}
