package csvxl

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"testing"
	"time"

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

// farRightWorkbook writes a workbook whose sheet "Target" has 200 rows, each
// with a value in column A and one in XFD, the last column Excel has, plus a
// formula-only cell near the right edge.
func farRightWorkbook(t *testing.T, path string) {
	t.Helper()
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	_, err := f.NewSheet("Target")
	require.NoError(t, err)
	for r := 1; r <= 200; r++ {
		a, err := excelize.CoordinatesToCellName(1, r)
		require.NoError(t, err)
		xfd, err := excelize.CoordinatesToCellName(excelize.MaxColumns, r)
		require.NoError(t, err)
		require.NoError(t, f.SetCellValue("Target", a, r))
		require.NoError(t, f.SetCellValue("Target", xfd, r))
	}
	require.NoError(t, f.SetCellFormula("Target", "XFC7", "A7*2"))
	require.NoError(t, f.SaveAs(path))
}

// measure runs fn and reports how long it took and how many bytes it allocated.
func measure(fn func()) (time.Duration, uint64) {
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	start := time.Now()
	fn()
	elapsed := time.Since(start)
	runtime.ReadMemStats(&after)
	return elapsed, after.TotalAlloc - before.TotalAlloc
}

// Clearing a replaced sheet must cost about what v0.3.2 spent writing over it.
// Rows pads each row out to its last cell, and clearing every padded position
// made a sheet with a value in XFD cost 16,384 cell writes per row: 1.2 s and
// 2.9 GB against v0.3.2's 160 ms and 657 MB.
func TestAppendCsvToExcelClearsAFarRightSheetCheaply(t *testing.T) {
	dir := t.TempDir()
	small := writeCSV(t, dir, "small.csv", "z\n9")
	baseline := filepath.Join(dir, "baseline.xlsx")
	replaced := filepath.Join(dir, "replaced.xlsx")
	farRightWorkbook(t, baseline)
	farRightWorkbook(t, replaced)

	// What v0.3.2's AppendCsvToExcel did with an existing sheet: write the CSV
	// over it and save, clearing nothing.
	baseTime, baseAlloc := measure(func() {
		f, err := excelize.OpenFile(baseline)
		require.NoError(t, err)
		_, err = f.NewSheet("Target")
		require.NoError(t, err)
		require.NoError(t, addCsvSheet(f, "Target", small, UTF8))
		require.NoError(t, f.SaveAs(baseline))
		require.NoError(t, f.Close())
	})
	gotTime, gotAlloc := measure(func() {
		require.NoError(t, AppendCsvToExcel([]string{small}, []string{"Target"}, replaced, UTF8))
	})
	t.Logf("v0.3.2 path: %v, %.1f MB; AppendCsvToExcel: %v, %.1f MB",
		baseTime, float64(baseAlloc)/1e6, gotTime, float64(gotAlloc)/1e6)
	require.LessOrEqual(t, gotAlloc, 2*baseAlloc, "clearing the sheet allocated more than twice what v0.3.2 did")
	require.LessOrEqual(t, gotTime, 4*baseTime, "clearing the sheet took more than four times what v0.3.2 did")

	f, err := excelize.OpenFile(replaced)
	require.NoError(t, err)
	defer func() { _ = f.Close() }()
	rows, err := f.GetRows("Target")
	require.NoError(t, err)
	require.Equal(t, [][]string{{"z"}, {"9"}}, rows, "stale cells must not survive")
	formula, err := f.GetCellFormula("Target", "XFC7")
	require.NoError(t, err)
	require.Empty(t, formula, "a stale formula must not survive")
}

// A cell or row element may leave out its r attribute, and excelize reads such
// a sheet in order. Listing the cells by address cannot, so replacing the sheet
// has to fall back to walking the rows.
func TestAppendCsvToExcelReplacesASheetWithoutCellAddresses(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.xlsx")
	f := excelize.NewFile()
	_, err := f.NewSheet("Target")
	require.NoError(t, err)
	require.NoError(t, f.SetSheetRow("Target", "A1", &[]any{"a", "b", "c"}))
	require.NoError(t, f.SetSheetRow("Target", "A2", &[]any{1, 2, 3}))
	require.NoError(t, f.SaveAs(src))
	require.NoError(t, f.Close())

	// Rewrite the package with every r attribute removed from the sheet.
	xlsx := filepath.Join(dir, "noaddress.xlsx")
	zr, err := zip.OpenReader(src)
	require.NoError(t, err)
	defer func() { _ = zr.Close() }()
	out, err := os.Create(xlsx)
	require.NoError(t, err)
	zw := zip.NewWriter(out)
	address := regexp.MustCompile(` r="[^"]*"`)
	stripped := false
	for _, entry := range zr.File {
		rc, err := entry.Open()
		require.NoError(t, err)
		content, err := io.ReadAll(rc)
		require.NoError(t, err)
		require.NoError(t, rc.Close())
		if entry.Name == "xl/worksheets/sheet2.xml" {
			content = address.ReplaceAll(content, nil)
			stripped = true
		}
		w, err := zw.Create(entry.Name)
		require.NoError(t, err)
		_, err = w.Write(content)
		require.NoError(t, err)
	}
	require.NoError(t, zw.Close())
	require.NoError(t, out.Close())
	require.True(t, stripped, "the Target sheet was not found in the package")

	small := writeCSV(t, dir, "small.csv", "z\n9")
	require.NoError(t, AppendCsvToExcel([]string{small}, []string{"Target"}, xlsx, UTF8))
	require.Equal(t, [][]string{{"z"}, {"9"}}, sheetRows(t, xlsx, "Target"), "stale cells must not survive")
}
