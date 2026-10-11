package insyra

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/xuri/excelize/v2"
)

// A save touches only its own sheet: a workbook's other sheets can never be
// lost to it, and an existing sheet is replaced only when asked.

func excelTestTable(values ...any) *DataTable {
	col := NewDataList(values...).SetName("v")
	return NewDataTable(col)
}

// workbookWith writes a workbook whose sheets each hold one cell naming it.
func workbookWith(t *testing.T, path string, sheets ...string) {
	t.Helper()
	f := excelize.NewFile()
	for i, name := range sheets {
		if i == 0 {
			if err := f.SetSheetName("Sheet1", name); err != nil {
				t.Fatal(err)
			}
		} else if _, err := f.NewSheet(name); err != nil {
			t.Fatal(err)
		}
		if err := f.SetCellValue(name, "A1", "sheet "+name); err != nil {
			t.Fatal(err)
		}
	}
	if err := f.SaveAs(path); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
}

func sheetCells(t *testing.T, path, sheet string) [][]string {
	t.Helper()
	f, err := excelize.OpenFile(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	rows, err := f.GetRows(sheet)
	if err != nil {
		t.Fatal(err)
	}
	return rows
}

func sheetList(t *testing.T, path string) []string {
	t.Helper()
	f, err := excelize.OpenFile(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	return f.GetSheetList()
}

func TestToExcelCreatesAWorkbook(t *testing.T) {
	path := filepath.Join(t.TempDir(), "new.xlsx")
	if err := excelTestTable(1, 2).ToExcel(path, ExcelWriteOptions{SetColNamesToFirstRow: true}); err != nil {
		t.Fatal(err)
	}
	if got := sheetList(t, path); !reflect.DeepEqual(got, []string{"Sheet1"}) {
		t.Fatalf("sheets %v, want [Sheet1]", got)
	}
	if got := sheetCells(t, path, "Sheet1"); !reflect.DeepEqual(got, [][]string{{"v"}, {"1"}, {"2"}}) {
		t.Fatalf("cells %v", got)
	}
}

func TestToExcelAddsASheetAndLeavesTheOthers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "book.xlsx")
	workbookWith(t, path, "2023", "2024")
	if err := excelTestTable(7).ToExcel(path, ExcelWriteOptions{Sheet: "2025"}); err != nil {
		t.Fatal(err)
	}
	if got := sheetList(t, path); !reflect.DeepEqual(got, []string{"2023", "2024", "2025"}) {
		t.Fatalf("sheets %v", got)
	}
	for _, name := range []string{"2023", "2024"} {
		if got := sheetCells(t, path, name); !reflect.DeepEqual(got, [][]string{{"sheet " + name}}) {
			t.Fatalf("sheet %s changed to %v", name, got)
		}
	}
}

func TestToExcelRefusesAnExistingSheetByDefault(t *testing.T) {
	path := filepath.Join(t.TempDir(), "book.xlsx")
	workbookWith(t, path, "2023", "2024")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	err = excelTestTable(7).ToExcel(path, ExcelWriteOptions{Sheet: "2024"})
	if !errors.Is(err, ErrSheetExists) {
		t.Fatalf("got %v, want ErrSheetExists", err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("a refused save changed the file")
	}
}

func TestToExcelReplacesASheetInPlace(t *testing.T) {
	path := filepath.Join(t.TempDir(), "book.xlsx")
	workbookWith(t, path, "2023", "2024", "2025")
	err := excelTestTable(9).ToExcel(path, ExcelWriteOptions{Sheet: "2024", IfSheetExists: SheetExistsReplace})
	if err != nil {
		t.Fatal(err)
	}
	if got := sheetList(t, path); !reflect.DeepEqual(got, []string{"2023", "2024", "2025"}) {
		t.Fatalf("sheet order %v, want the original", got)
	}
	if got := sheetCells(t, path, "2024"); !reflect.DeepEqual(got, [][]string{{"9"}}) {
		t.Fatalf("replaced sheet holds %v, want only the new table", got)
	}
	for _, name := range []string{"2023", "2025"} {
		if got := sheetCells(t, path, name); !reflect.DeepEqual(got, [][]string{{"sheet " + name}}) {
			t.Fatalf("sheet %s changed to %v", name, got)
		}
	}
}

func TestToExcelReplacesTheOnlySheet(t *testing.T) {
	path := filepath.Join(t.TempDir(), "book.xlsx")
	workbookWith(t, path, "only")
	err := excelTestTable(3).ToExcel(path, ExcelWriteOptions{Sheet: "only", IfSheetExists: SheetExistsReplace})
	if err != nil {
		t.Fatal(err)
	}
	if got := sheetList(t, path); !reflect.DeepEqual(got, []string{"only"}) {
		t.Fatalf("sheets %v", got)
	}
	if got := sheetCells(t, path, "only"); !reflect.DeepEqual(got, [][]string{{"3"}}) {
		t.Fatalf("cells %v", got)
	}
}

// A sheet-scoped name defined for a sheet after the replaced one must keep
// pointing at that sheet. excelize's DeleteSheet moves every later sheet's name
// down one index and moving the rebuilt sheet back does not move it back, so
// without the correction Rate would end up belonging to the replaced sheet and
// Last!C1 would read the wrong cell.
func TestToExcelReplaceKeepsTheNamesOfTheSheetsAfterIt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "book.xlsx")
	f := excelize.NewFile()
	if err := f.SetSheetName("Sheet1", "First"); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Target", "Last"} {
		if _, err := f.NewSheet(name); err != nil {
			t.Fatal(err)
		}
	}
	if err := f.SetCellValue("Last", "B2", 7); err != nil {
		t.Fatal(err)
	}
	if err := f.SetCellFormula("Last", "C1", "Rate*10"); err != nil {
		t.Fatal(err)
	}
	if err := f.SetDefinedName(&excelize.DefinedName{Name: "Rate", Scope: "Last", RefersTo: "Last!$B$2"}); err != nil {
		t.Fatal(err)
	}
	if err := f.SaveAs(path); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()

	if err := excelTestTable(9).ToExcel(path, ExcelWriteOptions{Sheet: "Target", IfSheetExists: SheetExistsReplace}); err != nil {
		t.Fatal(err)
	}

	f, err := excelize.OpenFile(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	if got := f.GetSheetList(); !reflect.DeepEqual(got, []string{"First", "Target", "Last"}) {
		t.Fatalf("sheet order %v, want the original", got)
	}
	scopes := map[string]string{}
	for _, dn := range f.GetDefinedName() {
		scopes[dn.Name] = dn.Scope
	}
	if scopes["Rate"] != "Last" {
		t.Fatalf("Rate belongs to %q, want Last", scopes["Rate"])
	}
	got, err := f.CalcCellValue("Last", "C1")
	if err != nil {
		t.Fatal(err)
	}
	if got != "70" {
		t.Fatalf("Rate*10 reads %q, want 70", got)
	}
}

// Whether a sheet is hidden lives in the workbook rather than in the sheet, so
// a sheet replaced by deleting and re-creating it comes back visible unless the
// state is put back. The active sheet, too, must survive the replacement.
func TestToExcelReplaceKeepsAHiddenSheetHidden(t *testing.T) {
	path := filepath.Join(t.TempDir(), "book.xlsx")
	f := excelize.NewFile()
	if err := f.SetSheetName("Sheet1", "Dashboard"); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Data", "Secret"} {
		if _, err := f.NewSheet(name); err != nil {
			t.Fatal(err)
		}
	}
	if err := f.SetSheetVisible("Data", false); err != nil {
		t.Fatal(err)
	}
	if err := f.SetSheetVisible("Secret", false, true); err != nil {
		t.Fatal(err)
	}
	if err := f.SaveAs(path); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()

	if err := excelTestTable(9).ToExcel(path, ExcelWriteOptions{Sheet: "Data", IfSheetExists: SheetExistsReplace}); err != nil {
		t.Fatal(err)
	}

	f, err := excelize.OpenFile(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	if got := f.GetSheetList(); !reflect.DeepEqual(got, []string{"Dashboard", "Data", "Secret"}) {
		t.Fatalf("sheets %v, want the original order", got)
	}
	states := map[string]string{}
	for _, s := range f.WorkBook.Sheets.Sheet {
		states[s.Name] = s.State
	}
	if states["Data"] != "hidden" {
		t.Fatalf("Data state %q, want hidden", states["Data"])
	}
	if states["Secret"] != "veryHidden" {
		t.Fatalf("Secret state %q, want veryHidden", states["Secret"])
	}
	if active := f.GetSheetName(f.GetActiveSheetIndex()); active != "Dashboard" {
		t.Fatalf("active sheet %q, want the one that was active", active)
	}
}

func TestWriteExcelWritesAWorkbook(t *testing.T) {
	var buf bytes.Buffer
	if err := excelTestTable(4, 5).WriteExcel(&buf, ExcelWriteOptions{Sheet: "data", SetColNamesToFirstRow: true}); err != nil {
		t.Fatal(err)
	}
	dt, err := ReadExcel(bytes.NewReader(buf.Bytes()), "data", false, true)
	if err != nil {
		t.Fatal(err)
	}
	if got := dt.ColNames(); !reflect.DeepEqual(got, []string{"v"}) {
		t.Fatalf("columns %v", got)
	}
}

// Sheet names match without regard to case, as in Excel, so "sheet1" is the
// sheet "Sheet1" and is refused like it.
func TestToExcelMatchesSheetNamesWithoutCase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "book.xlsx")
	workbookWith(t, path, "Data")
	err := excelTestTable(1).ToExcel(path, ExcelWriteOptions{Sheet: "data"})
	if !errors.Is(err, ErrSheetExists) {
		t.Fatalf("got %v, want ErrSheetExists", err)
	}
}

// A formula on another sheet names the replaced sheet by its name, so it
// reads the new contents rather than breaking.
func TestToExcelReplaceKeepsFormulasOnOtherSheetsWorking(t *testing.T) {
	path := filepath.Join(t.TempDir(), "book.xlsx")
	workbookWith(t, path, "data", "summary")
	f, err := excelize.OpenFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.SetCellFormula("summary", "B1", "SUM(data!A1:A3)"); err != nil {
		t.Fatal(err)
	}
	if err := f.Save(); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()

	err = excelTestTable(10, 20, 30).ToExcel(path, ExcelWriteOptions{Sheet: "data", IfSheetExists: SheetExistsReplace})
	if err != nil {
		t.Fatal(err)
	}
	f, err = excelize.OpenFile(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	got, err := f.CalcCellValue("summary", "B1")
	if err != nil {
		t.Fatal(err)
	}
	if got != "60" {
		t.Fatalf("formula reads %q, want 60", got)
	}
	if active := f.GetSheetName(f.GetActiveSheetIndex()); active != "data" {
		t.Fatalf("active sheet %q, want the one that was active", active)
	}
}

func TestToExcelWritesRowNamesBesideAHeader(t *testing.T) {
	path := filepath.Join(t.TempDir(), "names.xlsx")
	dt := NewDataTable(
		NewDataList(1, 2).SetName("a"),
		NewDataList("x").SetName("b"),
	)
	dt.SetRowNameByIndex(0, "r1")
	err := dt.ToExcel(path, ExcelWriteOptions{SetColNamesToFirstRow: true, SetRowNamesToFirstCol: true})
	if err != nil {
		t.Fatal(err)
	}
	want := [][]string{{"", "a", "b"}, {"r1", "1", "x"}, {"", "2"}}
	if got := sheetCells(t, path, "Sheet1"); !reflect.DeepEqual(got, want) {
		t.Fatalf("cells %v, want %v", got, want)
	}
}

// A path Excel cannot hold is refused before anything is written.
func TestToExcelRefusesANonWorkbookExtension(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.csv")
	if err := excelTestTable(1).ToExcel(path, ExcelWriteOptions{}); err == nil {
		t.Fatal("saving a workbook as .csv succeeded")
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("a refused save left a file behind: %v", err)
	}
}
