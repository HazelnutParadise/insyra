package isr

type CSV struct {
	FilePath  string
	LoadOpts  CSV_inOpts
	OuputOpts CSV_outOpts
}

type CSV_inOpts struct {
	FirstCol2RowNames bool
	FirstRow2ColNames bool
}

type CSV_outOpts struct {
	RowNames2FirstCol bool
	ColNames2FirstRow bool
}
