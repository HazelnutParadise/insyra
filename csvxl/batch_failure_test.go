package csvxl

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
)

func sheetList(t *testing.T, xlsx string) []string {
	t.Helper()
	f, err := excelize.OpenFile(xlsx)
	require.NoError(t, err)
	defer func() { _ = f.Close() }()
	return f.GetSheetList()
}

func TestCsvToExcelKeepsGoingPastAMissingFile(t *testing.T) {
	dir := t.TempDir()
	good := writeCSV(t, dir, "good.csv", "a,b\n1,2\n")
	missing := filepath.Join(dir, "missing.csv")
	out := filepath.Join(dir, "out.xlsx")

	err := CsvToExcel([]string{good, missing}, nil, out, UTF8)
	require.Error(t, err)
	require.ErrorIs(t, err, os.ErrNotExist)
	require.Contains(t, err.Error(), missing)
	require.Equal(t, []string{"good"}, sheetList(t, out))
	require.Equal(t, [][]string{{"a", "b"}, {"1", "2"}}, sheetRows(t, out, "good"))
}

// The first file failing must not leave the new workbook's default sheet in
// the output.
func TestCsvToExcelNamesEveryFailedFile(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "first.csv")
	good := writeCSV(t, dir, "good.csv", "a\n1\n")
	second := filepath.Join(dir, "second.csv")
	out := filepath.Join(dir, "out.xlsx")

	err := CsvToExcel([]string{first, good, second}, nil, out, UTF8)
	require.Error(t, err)
	require.Contains(t, err.Error(), first)
	require.Contains(t, err.Error(), second)
	require.Equal(t, []string{"good"}, sheetList(t, out))
}

func TestCsvToExcelReportsASheetNameExcelRejects(t *testing.T) {
	dir := t.TempDir()
	a := writeCSV(t, dir, "a.csv", "x\n1\n")
	b := writeCSV(t, dir, "b.csv", "y\n2\n")
	out := filepath.Join(dir, "out.xlsx")

	err := CsvToExcel([]string{a, b}, []string{"a", "bad:name"}, out, UTF8)
	require.Error(t, err)
	require.Contains(t, err.Error(), b)
	require.Equal(t, []string{"a"}, sheetList(t, out))
}

func TestCsvToExcelWritesNothingWhenEveryFileFails(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "out.xlsx")

	err := CsvToExcel([]string{filepath.Join(dir, "a.csv"), filepath.Join(dir, "b.csv")}, nil, out, UTF8)
	require.Error(t, err)
	_, statErr := os.Stat(out)
	require.True(t, errors.Is(statErr, os.ErrNotExist), "an output file was written although nothing converted")
}

func TestAppendCsvToExcelLeavesASheetAloneWhenItsCSVFails(t *testing.T) {
	dir := t.TempDir()
	data := writeCSV(t, dir, "data.csv", "keep\nme\n")
	other := writeCSV(t, dir, "other.csv", "old\n")
	xlsx := filepath.Join(dir, "book.xlsx")
	require.NoError(t, CsvToExcel([]string{data, other}, nil, xlsx, UTF8))

	missing := filepath.Join(dir, "missing.csv")
	fresh := writeCSV(t, dir, "fresh.csv", "new\n")
	err := AppendCsvToExcel([]string{missing, fresh}, []string{"data", "other"}, xlsx, UTF8)
	require.Error(t, err)
	require.ErrorIs(t, err, os.ErrNotExist)
	require.Contains(t, err.Error(), missing)
	require.Equal(t, [][]string{{"keep"}, {"me"}}, sheetRows(t, xlsx, "data"))
	require.Equal(t, [][]string{{"new"}}, sheetRows(t, xlsx, "other"))
}

func TestAppendCsvToExcelLeavesTheFileAloneWhenEveryCSVFails(t *testing.T) {
	dir := t.TempDir()
	data := writeCSV(t, dir, "data.csv", "keep\n")
	xlsx := filepath.Join(dir, "book.xlsx")
	require.NoError(t, CsvToExcel([]string{data}, nil, xlsx, UTF8))
	before, err := os.ReadFile(xlsx)
	require.NoError(t, err)

	err = AppendCsvToExcel([]string{filepath.Join(dir, "missing.csv")}, []string{"data"}, xlsx, UTF8)
	require.Error(t, err)
	after, err := os.ReadFile(xlsx)
	require.NoError(t, err)
	require.True(t, bytes.Equal(before, after), "the workbook was rewritten although nothing was appended")
}

// A directory named like a CSV matches EachCsvToOneExcel's glob and cannot be
// read as one.
func TestEachCsvToOneExcelKeepsGoingPastAnUnreadableFile(t *testing.T) {
	dir := t.TempDir()
	writeCSV(t, dir, "good.csv", "a\n1\n")
	require.NoError(t, os.Mkdir(filepath.Join(dir, "broken.csv"), 0o755))
	out := filepath.Join(t.TempDir(), "out.xlsx")

	err := EachCsvToOneExcel(dir, out, UTF8)
	require.Error(t, err)
	require.Contains(t, err.Error(), "broken.csv")
	require.Equal(t, []string{"good"}, sheetList(t, out))
}
