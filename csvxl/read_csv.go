package csvxl

import (
	"fmt"
	"os"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/internal/csv"
)

// ReadCsvToString reads a CSV file and returns its content as UTF-8.
//
// Deprecated: use ReadCSVToString, which has the same arguments and behavior. Removed in 0.4.
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
	if isAutoEncoding(useEncoding) {
		detected, err := insyra.DetectEncoding(filePath)
		if err != nil {
			return "", fmt.Errorf("failed to auto-detect encoding for %s: %w", filePath, err)
		}
		useEncoding = detected
		insyra.LogInfo("csvxl", "ReadCsvToString", "Auto-detected encoding %s for file %s", useEncoding, filePath)
	}

	// The content is promised as UTF-8, so an encoding nothing here decodes is
	// an error rather than the file's raw bytes.
	if err := csv.CheckDecodable(useEncoding); err != nil {
		return "", fmt.Errorf("failed to read CSV file %s: %w", filePath, err)
	}

	result, err := csv.ReadCSVWithEncoding(file, useEncoding)
	if err != nil {
		return "", fmt.Errorf("failed to read CSV file %s: %w", filePath, err)
	}

	return result, nil
}

// ReadCSVToString reads a CSV file as a string using the same options as ReadCsvToString.
func ReadCSVToString(filePath string, encoding ...string) (string, error) {
	return ReadCsvToString(filePath, encoding...)
}
