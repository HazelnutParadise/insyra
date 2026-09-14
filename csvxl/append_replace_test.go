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

// Replacing a sheet keeps the sheet itself: its place among the sheets and its
// sheet-level settings, such as a column width, as v0.3.2 kept them. Only the
// old cells go.
func TestAppendCsvToExcelKeepsTheReplacedSheetsPositionAndSettings(t *testing.T) {
	dir := t.TempDir()
	first := writeCSV(t, dir, "First.csv", "f\n1")
	target := writeCSV(t, dir, "Target.csv", "a,b,c\n1,2,3\n4,5,6")
	last := writeCSV(t, dir, "Last.csv", "l\n1")
	xlsx := filepath.Join(dir, "out.xlsx")
	require.NoError(t, CsvToExcel([]string{first, target, last}, nil, xlsx, UTF8))

	f, err := excelize.OpenFile(xlsx)
	require.NoError(t, err)
	require.NoError(t, f.SetColWidth("Target", "A", "A", 40))
	// A formula-only cell past the end of a row and a far, sparse cell must
	// not survive either.
	require.NoError(t, f.SetCellFormula("Target", "E2", "A2*2"))
	require.NoError(t, f.SetCellValue("Target", "H40", "far"))
	require.NoError(t, f.Save())
	require.NoError(t, f.Close())

	small := writeCSV(t, dir, "small.csv", "z\n9")
	require.NoError(t, AppendCsvToExcel([]string{small}, []string{"Target"}, xlsx, UTF8))

	f, err = excelize.OpenFile(xlsx)
	require.NoError(t, err)
	defer func() { _ = f.Close() }()
	require.Equal(t, []string{"First", "Target", "Last"}, f.GetSheetList(), "the sheet must keep its position")
	width, err := f.GetColWidth("Target", "A")
	require.NoError(t, err)
	require.Equal(t, 40.0, width, "the sheet must keep its column width")
	rows, err := f.GetRows("Target")
	require.NoError(t, err)
	require.Equal(t, [][]string{{"z"}, {"9"}}, rows, "stale cells must not survive")
	formula, err := f.GetCellFormula("Target", "E2")
	require.NoError(t, err)
	require.Empty(t, formula, "a stale formula must not survive")
}

func TestAppendCsvToExcelKeepsTheOnlySheetsSettings(t *testing.T) {
	dir := t.TempDir()
	big := writeCSV(t, dir, "data.csv", "a,b\n1,2\n3,4")
	xlsx := filepath.Join(dir, "out.xlsx")
	require.NoError(t, CsvToExcel([]string{big}, nil, xlsx, UTF8))

	f, err := excelize.OpenFile(xlsx)
	require.NoError(t, err)
	require.NoError(t, f.SetColWidth("data", "B", "B", 33))
	require.NoError(t, f.Save())
	require.NoError(t, f.Close())

	small := writeCSV(t, dir, "small.csv", "a\n9")
	require.NoError(t, AppendCsvToExcel([]string{small}, []string{"data"}, xlsx, UTF8))

	f, err = excelize.OpenFile(xlsx)
	require.NoError(t, err)
	defer func() { _ = f.Close() }()
	require.Equal(t, []string{"data"}, f.GetSheetList())
	width, err := f.GetColWidth("data", "B")
	require.NoError(t, err)
	require.Equal(t, 33.0, width)
	rows, err := f.GetRows("data")
	require.NoError(t, err)
	require.Equal(t, [][]string{{"a"}, {"9"}}, rows)
}
