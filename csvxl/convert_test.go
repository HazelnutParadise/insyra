package csvxl

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectEncoding(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()

	// Test UTF-8 file
	utf8File := filepath.Join(tempDir, "test_utf8.csv")
	utf8Content := "name,age,city\nJohn,25,New York\nJane,30,San Francisco"
	err := os.WriteFile(utf8File, []byte(utf8Content), 0644)
	require.NoError(t, err)

	encoding, err := DetectEncoding(utf8File)
	assert.NoError(t, err)
	assert.Equal(t, UTF8, encoding)

	// Test UTF-8 file with Chinese characters
	utf8ChineseFile := filepath.Join(tempDir, "test_utf8_chinese.csv")
	utf8ChineseContent := "姓名,年齡,城市\n張三,25,台北\n李四,30,高雄"
	err = os.WriteFile(utf8ChineseFile, []byte(utf8ChineseContent), 0644)
	require.NoError(t, err)

	encoding, err = DetectEncoding(utf8ChineseFile)
	assert.NoError(t, err)
	assert.Equal(t, UTF8, encoding)

	// Test empty file
	emptyFile := filepath.Join(tempDir, "empty.csv")
	err = os.WriteFile(emptyFile, []byte(""), 0644)
	require.NoError(t, err)

	encoding, err = DetectEncoding(emptyFile)
	// Should still work, likely detecting as UTF-8
	assert.NoError(t, err)
	assert.Contains(t, []string{UTF8}, encoding)

	// Test non-existent file
	_, err = DetectEncoding(filepath.Join(tempDir, "nonexistent.csv"))
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
	CsvToExcel([]string{csvFile1, csvFile2}, nil, outputFile)

	// Check if the Excel file was created
	_, err = os.Stat(outputFile)
	assert.NoError(t, err, "Excel file should be created")

	// Test explicit auto encoding
	outputFile2 := filepath.Join(tempDir, "output_explicit_auto.xlsx")
	CsvToExcel([]string{csvFile1, csvFile2}, nil, outputFile2, Auto)

	// Check if the Excel file was created
	_, err = os.Stat(outputFile2)
	assert.NoError(t, err, "Excel file should be created with explicit auto encoding")

	// Test explicit UTF-8 encoding
	outputFile3 := filepath.Join(tempDir, "output_utf8.xlsx")
	CsvToExcel([]string{csvFile1, csvFile2}, nil, outputFile3, UTF8)

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
	CsvToExcel([]string{csvFile1}, nil, outputFile)

	// Create additional CSV file to append
	csvFile2 := filepath.Join(tempDir, "append.csv")
	csvContent2 := "product,price\nLaptop,1000"
	err = os.WriteFile(csvFile2, []byte(csvContent2), 0644)
	require.NoError(t, err)

	// Test appending with auto-detection (default)
	AppendCsvToExcel([]string{csvFile2}, nil, outputFile)

	// Check if the Excel file still exists
	_, err = os.Stat(outputFile)
	assert.NoError(t, err, "Excel file should exist after append")

	// Test appending with explicit auto encoding
	csvFile3 := filepath.Join(tempDir, "append2.csv")
	csvContent3 := "country,population\nUSA,331000000"
	err = os.WriteFile(csvFile3, []byte(csvContent3), 0644)
	require.NoError(t, err)

	AppendCsvToExcel([]string{csvFile3}, nil, outputFile, Auto)

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

	// Test converting all CSV files in directory with auto-detection
	outputFile := filepath.Join(tempDir, "directory_output.xlsx")
	EachCsvToOneExcel(tempDir, outputFile)

	// Check if the Excel file was created
	_, err = os.Stat(outputFile)
	assert.NoError(t, err, "Excel file should be created from directory")

	// Test with explicit auto encoding
	outputFile2 := filepath.Join(tempDir, "directory_output_explicit.xlsx")
	EachCsvToOneExcel(tempDir, outputFile2, Auto)

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