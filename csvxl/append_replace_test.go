package csvxl

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
)

func writeCSV(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(p, []byte(content), 0o644))
	return p
}

func sheetRows(t *testing.T, xlsx, sheet string) [][]string {
	t.Helper()
	f, err := excelize.OpenFile(xlsx)
	require.NoError(t, err)
	defer func() { _ = f.Close() }()
	rows, err := f.GetRows(sheet)
	require.NoError(t, err)
	return rows
}

func TestAppendCsvToExcelReplacesExistingSheet(t *testing.T) {
	dir := t.TempDir()
	big := writeCSV(t, dir, "data.csv", "a,b\n1,2\n3,4\n5,6")
	other := writeCSV(t, dir, "other.csv", "x\n1")
	xlsx := filepath.Join(dir, "out.xlsx")
	require.NoError(t, CsvToExcel([]string{big, other}, nil, xlsx, UTF8))
	require.Len(t, sheetRows(t, xlsx, "data"), 4)

	small := writeCSV(t, dir, "small.csv", "a\n9")
	require.NoError(t, AppendCsvToExcel([]string{small}, []string{"data"}, xlsx, UTF8))
	rows := sheetRows(t, xlsx, "data")
	require.Equal(t, [][]string{{"a"}, {"9"}}, rows, "stale cells must not survive")
}

func TestAppendCsvToExcelReplacesOnlySheet(t *testing.T) {
	dir := t.TempDir()
	big := writeCSV(t, dir, "data.csv", "a,b\n1,2\n3,4")
	xlsx := filepath.Join(dir, "out.xlsx")
	require.NoError(t, CsvToExcel([]string{big}, nil, xlsx, UTF8))

	small := writeCSV(t, dir, "small.csv", "a\n9")
	require.NoError(t, AppendCsvToExcel([]string{small}, []string{"data"}, xlsx, UTF8))

	f, err := excelize.OpenFile(xlsx)
	require.NoError(t, err)
	defer func() { _ = f.Close() }()
	require.Equal(t, []string{"data"}, f.GetSheetList())
	rows, err := f.GetRows("data")
	require.NoError(t, err)
	require.Equal(t, [][]string{{"a"}, {"9"}}, rows)
}
