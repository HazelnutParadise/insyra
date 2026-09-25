package isr

type CSV struct {
	FilePath   string
	String     string
	InputOpts  CSV_inOpts
	OutputOpts CSV_outOpts
}

type CSV_inOpts struct {
	NoHeaderRow      bool   // the first row is data, not column names
	HasRowNames      bool   // the first column holds row names
	Encoding         string // optional, default is "auto", only for FilePath input
	RawStrings       bool   // keep every cell as its original string; skip column type inference
	AllowRaggedRows  bool   // pad short rows and keep long rows in auto-named columns
	TrimLeadingSpace bool   // ignore leading spaces before CSV fields, including quotes
}

type CSV_outOpts struct {
	NoHeaderRow bool // leave out the row of column names
	HasRowNames bool // write the row names as the first column
}
