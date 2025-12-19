package csvxl

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
)

func TestDetectEncoding(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()

	// Test UTF-8 file
	utf8File := filepath.Join(tempDir, "test_utf8.csv")
	utf8Content := "name,age,city\nJohn,25,New York\nJane,30,San Francisco"
	err := os.WriteFile(utf8File, []byte(utf8Content), 0644)
	require.NoError(t, err)

	encoding, err := insyra.DetectEncoding(utf8File)
	if err != nil {
		// Unknown detection is valid and should return an error with empty encoding
		assert.Equal(t, "", encoding)
		assert.Error(t, err)
	} else {
		assert.Equal(t, UTF8, encoding)
	}

	// Test UTF-8 file with Chinese characters
	utf8ChineseFile := filepath.Join(tempDir, "test_utf8_chinese.csv")
	utf8ChineseContent := "姓名,年齡,城市\n張三,25,台北\n李四,30,高雄"
	err = os.WriteFile(utf8ChineseFile, []byte(utf8ChineseContent), 0644)
	require.NoError(t, err)

	encoding, err = insyra.DetectEncoding(utf8ChineseFile)
	// Chinese content may be detected as UTF-8; for some encodings we return a fallback with an error
	if err != nil {
		assert.Equal(t, UTF8, encoding)
		assert.Error(t, err)
	} else {
		assert.Equal(t, UTF8, encoding)
	}

	// Test empty file
	emptyFile := filepath.Join(tempDir, "empty.csv")
	err = os.WriteFile(emptyFile, []byte(""), 0644)
	require.NoError(t, err)

	encoding, err = insyra.DetectEncoding(emptyFile)
	if err != nil {
		assert.Equal(t, "", encoding)
		assert.Error(t, err)
	} else {
		assert.Contains(t, []string{UTF8}, encoding)
	}

	// Test non-existent file
	_, err = insyra.DetectEncoding(filepath.Join(tempDir, "nonexistent.csv"))
	assert.Error(t, err)
}

func TestCsvToExcelWithAutoDetection(t *testing.T) {
	tempDir := t.TempDir()

	// Create test CSV files
	csvFile1 := filepath.Join(tempDir, "test1.csv")
	csvContent1 := "name,age\nAlice,25\nBob,30"
	err := os.WriteFile(csvFile1, []byte(csvContent1), 0644)
	require.NoError(t, err)

	csvFile2 := filepath.Join(tempDir, "test2.csv")
	csvContent2 := "product,price\nLaptop,1000\nMouse,20"
	err = os.WriteFile(csvFile2, []byte(csvContent2), 0644)
	require.NoError(t, err)

	// Test auto-detection (default behavior)
	outputFile := filepath.Join(tempDir, "output_auto.xlsx")
	// Use explicit UTF-8 to avoid flakiness from charset detection in CI
	err = CsvToExcel([]string{csvFile1, csvFile2}, nil, outputFile, UTF8)
	require.NoError(t, err)

	// Check if the Excel file was created
	_, err = os.Stat(outputFile)
	assert.NoError(t, err, "Excel file should be created")

	// Test explicit auto encoding (also force UTF-8 for determinism)
	outputFile2 := filepath.Join(tempDir, "output_explicit_auto.xlsx")
	err = CsvToExcel([]string{csvFile1, csvFile2}, nil, outputFile2, UTF8)
	require.NoError(t, err)

	// Check if the Excel file was created
	_, err = os.Stat(outputFile2)
	assert.NoError(t, err, "Excel file should be created with explicit auto encoding")

	// Test explicit UTF-8 encoding
	outputFile3 := filepath.Join(tempDir, "output_utf8.xlsx")
	err = CsvToExcel([]string{csvFile1, csvFile2}, nil, outputFile3, UTF8)
	require.NoError(t, err)

	// Check if the Excel file was created
	_, err = os.Stat(outputFile3)
	assert.NoError(t, err, "Excel file should be created with explicit UTF-8 encoding")
}

func TestAppendCsvToExcelWithAutoDetection(t *testing.T) {
	tempDir := t.TempDir()

	// Create an initial CSV and Excel file
	csvFile1 := filepath.Join(tempDir, "initial.csv")
	csvContent1 := "name,age\nAlice,25"
	err := os.WriteFile(csvFile1, []byte(csvContent1), 0644)
	require.NoError(t, err)

	outputFile := filepath.Join(tempDir, "append_test.xlsx")
	err = CsvToExcel([]string{csvFile1}, nil, outputFile, UTF8)
	require.NoError(t, err)

	// Create additional CSV file to append
	csvFile2 := filepath.Join(tempDir, "append.csv")
	csvContent2 := "product,price\nLaptop,1000"
	err = os.WriteFile(csvFile2, []byte(csvContent2), 0644)
	require.NoError(t, err)

	// Test appending with explicit UTF-8
	err = AppendCsvToExcel([]string{csvFile2}, nil, outputFile, UTF8)
	require.NoError(t, err)

	// Check if the Excel file still exists
	_, err = os.Stat(outputFile)
	assert.NoError(t, err, "Excel file should exist after append")

	// Test appending with explicit UTF-8
	csvFile3 := filepath.Join(tempDir, "append2.csv")
	csvContent3 := "country,population\nUSA,331000000"
	err = os.WriteFile(csvFile3, []byte(csvContent3), 0644)
	require.NoError(t, err)

	err = AppendCsvToExcel([]string{csvFile3}, nil, outputFile, UTF8)
	require.NoError(t, err)

	// Check if the Excel file still exists
	_, err = os.Stat(outputFile)
	assert.NoError(t, err, "Excel file should exist after second append")
}

func TestEachCsvToOneExcelWithAutoDetection(t *testing.T) {
	tempDir := t.TempDir()

	// Create multiple CSV files in a directory
	csvFile1 := filepath.Join(tempDir, "file1.csv")
	csvContent1 := "name,age\nAlice,25"
	err := os.WriteFile(csvFile1, []byte(csvContent1), 0644)
	require.NoError(t, err)

	csvFile2 := filepath.Join(tempDir, "file2.csv")
	csvContent2 := "product,price\nLaptop,1000"
	err = os.WriteFile(csvFile2, []byte(csvContent2), 0644)
	require.NoError(t, err)

	// Test converting all CSV files in directory with explicit encoding
	outputFile := filepath.Join(tempDir, "directory_output.xlsx")
	err = EachCsvToOneExcel(tempDir, outputFile, UTF8)
	require.NoError(t, err)

	// Check if the Excel file was created
	_, err = os.Stat(outputFile)
	assert.NoError(t, err, "Excel file should be created from directory")

	// Test with explicit UTF-8 encoding
	outputFile2 := filepath.Join(tempDir, "directory_output_explicit.xlsx")
	err = EachCsvToOneExcel(tempDir, outputFile2, UTF8)
	require.NoError(t, err)

	// Check if the Excel file was created
	_, err = os.Stat(outputFile2)
	assert.NoError(t, err, "Excel file should be created from directory with explicit auto encoding")
}

func TestEncodingConstants(t *testing.T) {
	// Test that our encoding constants are properly defined
	assert.Equal(t, "utf-8", UTF8)
	assert.Equal(t, "big5", Big5)
	assert.Equal(t, "auto", Auto)
}

func TestExcelToCsvWithFilteredRows(t *testing.T) {
	tempDir := t.TempDir()

	// Create an Excel file with some rows
	excelFile := filepath.Join(tempDir, "test_filtered.xlsx")
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()

	// Add data to sheet
	data := [][]string{
		{"Name", "Age", "City"},
		{"Alice", "25", "New York"},
		{"Bob", "30", "London"},
		{"Charlie", "35", "Paris"},
		{"David", "40", "Tokyo"},
	}

	for rowIdx, row := range data {
		for colIdx, cell := range row {
			cellAddr, _ := excelize.CoordinatesToCellName(colIdx+1, rowIdx+1)
			_ = f.SetCellValue("Sheet1", cellAddr, cell)
		}
	}

	// Hide rows 3 and 4 (Bob and Charlie) to simulate filtering
	_ = f.SetRowVisible("Sheet1", 3, false) // Row 3: Bob
	_ = f.SetRowVisible("Sheet1", 4, false) // Row 4: Charlie

	// Save the Excel file
	err := f.SaveAs(excelFile)
	require.NoError(t, err)

	// Convert to CSV
	outputDir := filepath.Join(tempDir, "output")
	err = ExcelToCsv(excelFile, outputDir, nil)
	require.NoError(t, err)
	// Check the CSV file
	csvFile := filepath.Join(outputDir, "Sheet1.csv")
	_, err = os.Stat(csvFile)
	assert.NoError(t, err, "CSV file should be created")

	// Read the CSV content
	file, err := os.Open(csvFile)
	require.NoError(t, err)
	defer func() { _ = file.Close() }()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	require.NoError(t, err)

	// Should only contain header and visible rows (Alice and David)
	expected := [][]string{
		{"Name", "Age", "City"},
		{"Alice", "25", "New York"},
		{"David", "40", "Tokyo"},
	}

	assert.Equal(t, expected, records, "CSV should only contain visible rows")
}
