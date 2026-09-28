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

// Replacing a sheet keeps its place among the sheets and leaves the active
// sheet as it was, and none of the old cells or formulas survive.
func TestAppendCsvToExcelKeepsTheReplacedSheetsPosition(t *testing.T) {
	dir := t.TempDir()
	first := writeCSV(t, dir, "First.csv", "f\n1")
	target := writeCSV(t, dir, "Target.csv", "a,b,c\n1,2,3\n4,5,6")
	last := writeCSV(t, dir, "Last.csv", "l\n1")
	xlsx := filepath.Join(dir, "out.xlsx")
	require.NoError(t, CsvToExcel([]string{first, target, last}, nil, xlsx, UTF8))

	f, err := excelize.OpenFile(xlsx)
	require.NoError(t, err)
	// A formula-only cell past the end of a row and a far, sparse cell must
	// not survive either.
	require.NoError(t, f.SetCellFormula("Target", "E2", "A2*2"))
	require.NoError(t, f.SetCellValue("Target", "H40", "far"))
	f.SetActiveSheet(2)
	require.NoError(t, f.Save())
	require.NoError(t, f.Close())

	small := writeCSV(t, dir, "small.csv", "z\n9")
	require.NoError(t, AppendCsvToExcel([]string{small}, []string{"Target"}, xlsx, UTF8))

	f, err = excelize.OpenFile(xlsx)
	require.NoError(t, err)
	defer func() { _ = f.Close() }()
	require.Equal(t, []string{"First", "Target", "Last"}, f.GetSheetList(), "the sheet must keep its position")
	require.Equal(t, "Last", f.GetSheetName(f.GetActiveSheetIndex()), "the active sheet must not change")
	rows, err := f.GetRows("Target")
	require.NoError(t, err)
	require.Equal(t, [][]string{{"z"}, {"9"}}, rows, "stale cells must not survive")
	formula, err := f.GetCellFormula("Target", "E2")
	require.NoError(t, err)
	require.Empty(t, formula, "a stale formula must not survive")
}

// Clearing values and formulas left everything else a row or a cell carries:
// a new row landing on an old hidden row was invisible in Excel and skipped
// when the sheet was read back, and old comments and links sat on new values.
func TestAppendCsvToExcelLeavesNothingOfTheOldSheet(t *testing.T) {
	dir := t.TempDir()
	target := writeCSV(t, dir, "Target.csv", "a,b\n1,2\n3,4\n5,6")
	xlsx := filepath.Join(dir, "out.xlsx")
	require.NoError(t, CsvToExcel([]string{target}, nil, xlsx, UTF8))

	f, err := excelize.OpenFile(xlsx)
	require.NoError(t, err)
	require.NoError(t, f.SetRowVisible("Target", 2, false))
	require.NoError(t, f.SetRowVisible("Target", 3, false))
	require.NoError(t, f.SetRowHeight("Target", 4, 60))
	require.NoError(t, f.AddComment("Target", excelize.Comment{Cell: "B2", Author: "x", Text: "old note"}))
	require.NoError(t, f.SetCellHyperLink("Target", "B3", "https://example.com/old", "External"))
	require.NoError(t, f.SetColWidth("Target", "A", "A", 40))
	require.NoError(t, f.Save())
	require.NoError(t, f.Close())

	fresh := writeCSV(t, dir, "fresh.csv", "h\nr1\nr2\nr3\nr4")
	require.NoError(t, AppendCsvToExcel([]string{fresh}, []string{"Target"}, xlsx, UTF8))

	f, err = excelize.OpenFile(xlsx)
	require.NoError(t, err)
	for row := 1; row <= 5; row++ {
		visible, err := f.GetRowVisible("Target", row)
		require.NoError(t, err)
		require.True(t, visible, "row %d must be visible", row)
	}
	height, err := f.GetRowHeight("Target", 4)
	require.NoError(t, err)
	require.NotEqual(t, 60.0, height, "an old row height must not survive")
	comments, err := f.GetComments("Target")
	require.NoError(t, err)
	require.Empty(t, comments, "an old comment must not survive")
	linked, _, err := f.GetCellHyperLink("Target", "B3")
	require.NoError(t, err)
	require.False(t, linked, "an old hyperlink must not survive")
	width, err := f.GetColWidth("Target", "A")
	require.NoError(t, err)
	require.NotEqual(t, 40.0, width, "the replaced sheet starts with default column widths")
	require.NoError(t, f.Close())

	out := t.TempDir()
	require.NoError(t, ExcelToCsv(xlsx, out, nil))
	back, err := os.ReadFile(filepath.Join(out, "Target.csv"))
	require.NoError(t, err)
	require.Equal(t, "h\nr1\nr2\nr3\nr4\n", string(back), "every row must read back")
}

func TestAppendCsvToExcelReplacesTheOnlySheetWithAFreshOne(t *testing.T) {
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
	require.NotEqual(t, 33.0, width, "the replaced sheet starts with default column widths")
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

// Replacing a sheet must cost about what v0.3.2 spent writing over it,
// whatever the old sheet held. Clearing it cell by cell made a sheet with a
// value in XFD cost 16,384 cell writes per row: 1.2 s and 2.9 GB against
// v0.3.2's 160 ms and 657 MB.
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
// a sheet in order. Replacing it must still leave nothing of the old sheet.
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

// excelize's DeleteSheet moves every name defined for a sheet after the deleted
// one down one index, and moving the rebuilt sheet back does not move those
// names back, so each of them would end up belonging to the sheet before its
// own. Names belonging to the replaced sheet go with it, which is the point of
// replacing it.
func TestAppendCsvToExcelKeepsTheNamesOfTheSheetsAfterIt(t *testing.T) {
	dir := t.TempDir()
	xlsx := filepath.Join(dir, "out.xlsx")

	f := excelize.NewFile()
	require.NoError(t, f.SetSheetName("Sheet1", "First"))
	_, err := f.NewSheet("Target")
	require.NoError(t, err)
	_, err = f.NewSheet("Last")
	require.NoError(t, err)
	_, err = f.NewSheet("Tail")
	require.NoError(t, err)
	require.NoError(t, f.SetCellValue("Last", "B2", 7))
	require.NoError(t, f.SetCellFormula("Last", "C1", "Rate*10"))
	require.NoError(t, f.SetDefinedName(&excelize.DefinedName{Name: "Rate", Scope: "Last", RefersTo: "Last!$B$2"}))
	require.NoError(t, f.SetDefinedName(&excelize.DefinedName{Name: "TailName", Scope: "Tail", RefersTo: "Tail!$A$1"}))
	require.NoError(t, f.SetDefinedName(&excelize.DefinedName{Name: "Book", RefersTo: "Last!$B$2"}))
	require.NoError(t, f.SetDefinedName(&excelize.DefinedName{Name: "Own", Scope: "Target", RefersTo: "Target!$A$1"}))
	require.NoError(t, f.SaveAs(xlsx))
	require.NoError(t, f.Close())

	small := writeCSV(t, dir, "small.csv", "z\n9")
	require.NoError(t, AppendCsvToExcel([]string{small}, []string{"Target"}, xlsx, UTF8))

	f, err = excelize.OpenFile(xlsx)
	require.NoError(t, err)
	defer func() { _ = f.Close() }()
	scopes := map[string]string{}
	for _, dn := range f.GetDefinedName() {
		scopes[dn.Name] = dn.Scope
	}
	require.Equal(t, "Last", scopes["Rate"], "a name defined for a later sheet must stay with it")
	require.Equal(t, "Tail", scopes["TailName"], "a name defined for the last sheet must stay with it")
	require.Equal(t, "Workbook", scopes["Book"], "a workbook-scoped name must stay workbook-scoped")
	require.NotContains(t, scopes, "Own", "a name belonging to the replaced sheet goes with it")
	got, err := f.CalcCellValue("Last", "C1")
	require.NoError(t, err)
	require.Equal(t, "70", got)
	require.Equal(t, []string{"First", "Target", "Last", "Tail"}, f.GetSheetList(), "the sheet must keep its position")
}

// Whether a sheet is hidden lives in the workbook rather than in the sheet, so
// a sheet replaced by deleting and re-creating it comes back visible unless the
// state is put back. origin/dev, which emptied the sheet in place, kept it.
func TestAppendCsvToExcelKeepsAHiddenSheetHidden(t *testing.T) {
	dir := t.TempDir()
	xlsx := filepath.Join(dir, "out.xlsx")

	f := excelize.NewFile()
	require.NoError(t, f.SetSheetName("Sheet1", "Dashboard"))
	_, err := f.NewSheet("Data")
	require.NoError(t, err)
	_, err = f.NewSheet("Secret")
	require.NoError(t, err)
	require.NoError(t, f.SetSheetVisible("Data", false))
	require.NoError(t, f.SetSheetVisible("Secret", false, true))
	require.NoError(t, f.SaveAs(xlsx))
	require.NoError(t, f.Close())

	small := writeCSV(t, dir, "small.csv", "z\n9")
	require.NoError(t, AppendCsvToExcel([]string{small, small}, []string{"Data", "Secret"}, xlsx, UTF8))

	f, err = excelize.OpenFile(xlsx)
	require.NoError(t, err)
	defer func() { _ = f.Close() }()
	require.Equal(t, []string{"Dashboard", "Data", "Secret"}, f.GetSheetList(), "the sheets must keep their positions")
	require.NotNil(t, f.WorkBook, "the workbook must be read")
	states := map[string]string{}
	for _, s := range f.WorkBook.Sheets.Sheet {
		states[s.Name] = s.State
	}
	require.Equal(t, "hidden", states["Data"], "a hidden sheet must be hidden again")
	require.Equal(t, "veryHidden", states["Secret"], "a veryHidden sheet must be veryHidden again")
	visible, err := f.GetSheetVisible("Data")
	require.NoError(t, err)
	require.False(t, visible, "Data must not be visible")
	visible, err = f.GetSheetVisible("Secret")
	require.NoError(t, err)
	require.False(t, visible, "Secret must not be visible")
	require.Equal(t, "Dashboard", f.GetSheetName(f.GetActiveSheetIndex()), "the active sheet must not change")
	rows, err := f.GetRows("Data")
	require.NoError(t, err)
	require.Equal(t, [][]string{{"z"}, {"9"}}, rows)
}
