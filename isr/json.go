package isr

type JSON struct {
	// FilePath is the path to the JSON file.
	FilePath string

	// Bytes is the raw JSON data. Only used when FilePath is empty.
	Bytes []byte
}
