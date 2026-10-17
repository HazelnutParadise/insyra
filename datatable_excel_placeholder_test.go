package insyra

import (
	"path/filepath"
	"testing"

	"github.com/xuri/excelize/v2"
)

// A workbook whose only sheet is named like Replace's placeholder used to keep
// its old cells, because the placeholder and the sheet shared a name.
func TestToExcelReplacesAWorkbooksOnlySheetNamedLikeThePlaceholder(t *testing.T) {
	const name = "__insyra_placeholder__"
	path := filepath.Join(t.TempDir(), "w.xlsx")
	f := excelize.NewFile()
	if err := f.SetSheetName("Sheet1", name); err != nil {
		t.Fatal(err)
	}
	if err := f.SetCellValue(name, "A1", "old header"); err != nil {
		t.Fatal(err)
	}
	for _, cell := range []string{"A2", "A3", "A4", "B1"} {
		if err := f.SetCellValue(name, cell, "old cell"); err != nil {
			t.Fatal(err)
		}
	}
	if err := f.SaveAs(path); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()

	dt := excelTestTable("new")
	if err := dt.ToExcel(path, ExcelWriteOptions{Sheet: name, IfSheetExists: SheetExistsReplace}); err != nil {
		t.Fatal(err)
	}
	out, err := excelize.OpenFile(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = out.Close() }()
	if got := out.GetSheetList(); len(got) != 1 || got[0] != name {
		t.Fatalf("sheets = %v, want only %s", got, name)
	}
	rows, err := out.GetRows(name)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || len(rows[0]) != 1 || rows[0][0] != "v" || rows[1][0] != "new" {
		t.Fatalf("rows = %v, want the new table only", rows)
	}
}
