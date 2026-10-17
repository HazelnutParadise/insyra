package isr

// JSON is a JSON source for DT.From and a JSON target for (*DataTable).To.
type JSON struct {
	// FilePath is the path to the JSON file.
	FilePath string

	// Bytes is the raw JSON data. Only used when FilePath is empty.
	Bytes []byte

	// OutputOpts apply when writing with (*DataTable).To.
	OutputOpts JSON_outOpts
}

type JSON_outOpts struct {
	UseColIndex bool // key each value by its column index (A, B, …) instead of its column name
}
