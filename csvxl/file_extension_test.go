package csvxl

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// csvxl adds .csv as a convenience and never over a name the caller wrote.
// On the way in the path is read as given when it exists; on the way out a
// name keeps whatever extension it already has.

// firstCell converts one source into a workbook and returns cell A1 of its
// only sheet, so a test can tell which of two files was read.
func firstCell(t *testing.T, dir, source string) string {
	t.Helper()
	out := filepath.Join(dir, "out.xlsx")
	_ = os.Remove(out)
	require.NoError(t, CsvToExcel([]string{source}, []string{"s"}, out, UTF8))
	return sheetRows(t, out, "s")[0][0]
}

func TestCsvToExcelReadsANameThatDoesNotEndInCsv(t *testing.T) {
	for _, name := range []string{"export.txt", "DATA.CSV", "plain"} {
		dir := t.TempDir()
		writeCSV(t, dir, name, name+"\n1\n")
		require.Equal(t, name, firstCell(t, dir, filepath.Join(dir, name)), name)
	}
}

func TestCsvToExcelAddsCsvWhenThePathIsNotThere(t *testing.T) {
	for _, c := range []struct{ given, onDisk string }{
		{"data", "data.csv"},
		{"export.txt", "export.txt.csv"},
	} {
		dir := t.TempDir()
		writeCSV(t, dir, c.onDisk, c.onDisk+"\n1\n")
		require.Equal(t, c.onDisk, firstCell(t, dir, filepath.Join(dir, c.given)), c.given)
	}
}

func TestCsvToExcelReadsThePathAsWrittenWhenBothExist(t *testing.T) {
	dir := t.TempDir()
	writeCSV(t, dir, "x", "literal\n1\n")
	writeCSV(t, dir, "x.csv", "added\n1\n")
	require.Equal(t, "literal", firstCell(t, dir, filepath.Join(dir, "x")))
}

func TestCsvToExcelSkipsADirectoryOfTheSameName(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, "data"), 0o755))
	writeCSV(t, dir, "data.csv", "file\n1\n")
	require.Equal(t, "file", firstCell(t, dir, filepath.Join(dir, "data")))
}

func TestCsvToExcelNamesBothPathsWhenNeitherExists(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "nope")
	err := CsvToExcel([]string{missing}, nil, filepath.Join(dir, "out.xlsx"), UTF8)
	require.Error(t, err)
	// Quoted, so the bare path is not satisfied by being a prefix of the other.
	require.Contains(t, err.Error(), fmt.Sprintf("%q", missing))
	require.Contains(t, err.Error(), fmt.Sprintf("%q", missing+".csv"))
}

func TestAppendCsvToExcelReadsANameThatDoesNotEndInCsv(t *testing.T) {
	dir := t.TempDir()
	base := writeCSV(t, dir, "base.csv", "a\n1\n")
	xlsx := filepath.Join(dir, "book.xlsx")
	require.NoError(t, CsvToExcel([]string{base}, []string{"base"}, xlsx, UTF8))

	source := writeCSV(t, dir, "export.txt", "appended\n1\n")
	require.NoError(t, AppendCsvToExcel([]string{source}, []string{"more"}, xlsx, UTF8))
	require.Equal(t, "appended", sheetRows(t, xlsx, "more")[0][0])
}

func TestExcelToCsvKeepsTheExtensionItWasGiven(t *testing.T) {
	dir := t.TempDir()
	source := writeCSV(t, dir, "src.csv", "a\n1\n")
	xlsx := filepath.Join(dir, "book.xlsx")
	require.NoError(t, CsvToExcel([]string{source, source, source}, []string{"one", "two", "three"}, xlsx, UTF8))

	outDir := t.TempDir()
	require.NoError(t, ExcelToCsv(xlsx, outDir, []string{"report.txt", "REPORT.CSV", "bare"}))

	entries, err := os.ReadDir(outDir)
	require.NoError(t, err)
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	got := strings.Join(names, ",")
	require.Contains(t, got, "report.txt")
	require.NotContains(t, got, "report.txt.csv")
	require.Contains(t, got, "REPORT.CSV")
	require.NotContains(t, got, "REPORT.CSV.csv")
	require.Contains(t, got, "bare.csv")
}
