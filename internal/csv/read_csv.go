package csv

import (
	"bytes"
	"encoding/csv"
	"io"
	"os"
	"strings"
)

func ReadCSVWithEncoding(file *os.File, encoding string) (string, error) {
	records, err := ReadCSVRecordsWithEncodingOptions(file, encoding, false, false)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	if err := w.WriteAll(records); err != nil {
		return "", err
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// ReadCSVRecordsWithEncodingOptions decodes a CSV file and returns its parsed
// records with the requested reader tolerance settings applied. It returns
// records rather than re-encoded text: a serialize-reparse round trip would
// drop any record holding a single empty field (csv.Writer emits it as a blank
// line, which csv.Reader skips).
func ReadCSVRecordsWithEncodingOptions(file *os.File, encoding string, allowRaggedRows bool, trimLeadingSpace bool) ([][]string, error) {
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	reader, err := DecodingReader(file, encoding)
	if err != nil {
		return nil, err
	}

	csvReader := csv.NewReader(reader)
	if allowRaggedRows {
		csvReader.FieldsPerRecord = -1
	}
	csvReader.TrimLeadingSpace = trimLeadingSpace
	records, err := csvReader.ReadAll()
	if err != nil {
		return nil, err
	}

	// Trim UTF-8 BOM if present
	if len(records) > 0 && len(records[0]) > 0 {
		records[0][0] = strings.TrimPrefix(records[0][0], "\uFEFF")
	}

	return records, nil
}
