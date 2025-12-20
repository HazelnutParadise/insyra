package csvxl

import (
	"fmt"
	"os"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/internal/csv"
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
		useEncoding = detected
		insyra.LogInfo("csvxl", "ReadCsvToString", "Auto-detected encoding %s for file %s", useEncoding, filePath)
	}

	result, err := csv.ReadCSVWithEncoding(file, useEncoding)
	if err != nil {
		return "", fmt.Errorf("failed to read CSV file %s: %w", filePath, err)
	}

	return result, nil
}
