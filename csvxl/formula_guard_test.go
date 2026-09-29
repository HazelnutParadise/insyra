package csvxl

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xuri/excelize/v2"
)

// Text that is safe inside a workbook becomes a formula again once it is a
// CSV opened in a spreadsheet, so ExcelToCSV guards it by default, as the
// core CSV writer does (owner's ruling of 2026-09-26, #285).
func guardWorkbook(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "in.xlsx")
	f := excelize.NewFile()
	for i, v := range []string{"=1+1", "-5", "@x", "hello"} {
		cell, _ := excelize.CoordinatesToCellName(1, i+1)
		if err := f.SetCellStr("Sheet1", cell, v); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := f.NewSheet("Other"); err != nil {
		t.Fatal(err)
	}
	if err := f.SetCellStr("Other", "A1", "=2"); err != nil {
		t.Fatal(err)
	}
	if err := f.SaveAs(path); err != nil {
		t.Fatal(err)
	}
	return path
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestExcelToCSVGuardsFormulasByDefault(t *testing.T) {
	dir := t.TempDir()
	xlsx := guardWorkbook(t, dir)

	out := filepath.Join(dir, "guarded")
	if err := ExcelToCSV(xlsx, out, nil); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, filepath.Join(out, "Sheet1.csv")); got != "'=1+1\n-5\n'@x\nhello\n" {
		t.Fatalf("guarded CSV %q", got)
	}
	if got := readFile(t, filepath.Join(out, "Other.csv")); got != "'=2\n" {
		t.Fatalf("every sheet is converted when none is named: %q", got)
	}

	raw := filepath.Join(dir, "raw")
	if err := ExcelToCSV(xlsx, raw, nil, ExcelToCSVOptions{Sheets: []string{"Sheet1"}, AllowFormulas: true}); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, filepath.Join(raw, "Sheet1.csv")); got != "=1+1\n-5\n@x\nhello\n" {
		t.Fatalf("AllowFormulas CSV %q", got)
	}
	if _, err := os.Stat(filepath.Join(raw, "Other.csv")); !os.IsNotExist(err) {
		t.Fatalf("Sheets did not limit the conversion: %v", err)
	}
	if err := ExcelToCSV(xlsx, raw, nil, ExcelToCSVOptions{}, ExcelToCSVOptions{}); err == nil {
		t.Fatal("two options structs were accepted")
	}
}

func TestExcelDirToCSVGuardsFormulasByDefault(t *testing.T) {
	dir := t.TempDir()
	guardWorkbook(t, dir)
	out := filepath.Join(dir, "out")
	if err := ExcelDirToCSV(dir, out); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, filepath.Join(out, "in_Sheet1.csv")); got != "'=1+1\n-5\n'@x\nhello\n" {
		t.Fatalf("guarded CSV %q", got)
	}
	raw := filepath.Join(dir, "raw")
	if err := ExcelDirToCSV(dir, raw, ExcelToCSVOptions{AllowFormulas: true}); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, filepath.Join(raw, "in_Sheet1.csv")); got != "=1+1\n-5\n@x\nhello\n" {
		t.Fatalf("AllowFormulas CSV %q", got)
	}
}
