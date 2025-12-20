package csvxl

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/HazelnutParadise/insyra"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

func ReadCsvToString(filePath string, encoding ...string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open CSV file %s: %w", filePath, err)
	}
	defer func() { _ = file.Close() }()

	if len(encoding) == 0 {
		encoding = []string{Auto}
	} else if len(encoding) > 1 {
		return "", fmt.Errorf("too many arguments for encoding")
	}

	useEncoding := encoding[0]
	if useEncoding == Auto {
		detected, err := insyra.DetectEncoding(filePath)
		if err != nil {
			return "", fmt.Errorf("failed to auto-detect encoding for %s: %v", filePath, err)
		}
		useEncoding = strings.ToLower(detected)
		insyra.LogInfo("csvxl", "ReadCsvToString", "Auto-detected encoding %s for file %s", useEncoding, filePath)
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("failed to seek file %s: %w", filePath, err)
	}

	var reader io.Reader
	switch {
	case strings.Contains(useEncoding, "utf-8"):
		reader = file
	case strings.Contains(useEncoding, "big5"):
		reader = transform.NewReader(file, traditionalchinese.Big5.NewDecoder())
	case strings.Contains(useEncoding, "gb") || strings.Contains(useEncoding, "gb-"):
		reader = transform.NewReader(file, simplifiedchinese.GB18030.NewDecoder())
	case strings.Contains(useEncoding, "utf-16") || strings.Contains(useEncoding, "utf16"):
		reader = transform.NewReader(file, unicode.UTF16(unicode.LittleEndian, unicode.UseBOM).NewDecoder())
	default:
		insyra.LogWarning("csvxl", "ReadCsvToString", "unsupported encoding %s for file %s, reading as utf-8", useEncoding, filePath)
		reader = file
	}

	csvReader := csv.NewReader(reader)
	records, err := csvReader.ReadAll()
	if err != nil {
		return "", fmt.Errorf("failed to read CSV file %s: %w", filePath, err)
	}

	// Trim UTF-8 BOM if present
	if len(records) > 0 && len(records[0]) > 0 {
		records[0][0] = strings.TrimPrefix(records[0][0], "\uFEFF")
	}

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	if err := w.WriteAll(records); err != nil {
		return "", fmt.Errorf("failed to write CSV to string: %w", err)
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return "", err
	}

	return buf.String(), nil
}
