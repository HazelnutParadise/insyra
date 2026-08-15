package csv

import (
	"bytes"
	"encoding/csv"
	"io"
	"os"
	"strings"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
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

	var reader io.Reader
	switch {
	case strings.Contains(encoding, "utf-8"):
		reader = file
	case strings.Contains(encoding, "big5"):
		reader = transform.NewReader(file, traditionalchinese.Big5.NewDecoder())
	case strings.Contains(encoding, "gb") || strings.Contains(encoding, "gb-"):
		reader = transform.NewReader(file, simplifiedchinese.GB18030.NewDecoder())
	case strings.Contains(encoding, "utf-16") || strings.Contains(encoding, "utf16"):
		reader = transform.NewReader(file, unicode.UTF16(unicode.LittleEndian, unicode.UseBOM).NewDecoder())
	default:
		reader = file
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
