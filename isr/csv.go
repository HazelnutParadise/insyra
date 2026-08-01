package isr

type CSV struct {
	FilePath   string
	String     string
	InputOpts  CSV_inOpts
	OutputOpts CSV_outOpts
}

type CSV_inOpts struct {
	FirstCol2RowNames bool
	FirstRow2ColNames bool
	Encoding          string // optional, default is "auto", only for FilePath input
	RawStrings        bool   // keep every cell as its original string; skip column type inference
}

type CSV_outOpts struct {
	RowNames2FirstCol bool
	ColNames2FirstRow bool
}
