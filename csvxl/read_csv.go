package csvxl

import (
	"fmt"
	"os"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/internal/csv"
)

// ReadCSVToString reads the CSV file at filePath and returns its content as
// UTF-8 CSV text. encoding is the file's encoding, detected when it is left
// out or is "" or "auto"; a name no decoder handles is an error.
func ReadCSVToString(filePath string, encoding ...string) (string, error) {
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
	if isAutoEncoding(useEncoding) {
		detected, err := insyra.DetectEncoding(filePath)
		if err != nil {
			return "", fmt.Errorf("failed to auto-detect encoding for %s: %w", filePath, err)
		}
		useEncoding = detected
		insyra.LogInfo("csvxl", "ReadCSVToString", "Auto-detected encoding %s for file %s", useEncoding, filePath)
	}

	result, err := csv.ReadCSVWithEncoding(file, useEncoding)
	if err != nil {
		return "", fmt.Errorf("failed to read CSV file %s: %w", filePath, err)
	}

	return result, nil
}

// ReadCsvToString reads a CSV file as UTF-8 CSV text.
//
// Deprecated: use ReadCSVToString, which is the same function. Removed in the
// release after the one that deprecated it.
func ReadCsvToString(filePath string, encoding ...string) (string, error) {
	return ReadCSVToString(filePath, encoding...)
}
