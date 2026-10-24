package csvxl

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestExcelToCSVOptionsAndFormulaGuard(t *testing.T) {
	dir := t.TempDir()
	xlsx := filepath.Join(dir, "input.xlsx")
	f := excelize.NewFile()
	if err := f.SetCellValue("Sheet1", "A1", "=1+1"); err != nil {
		t.Fatal(err)
	}
	if _, err := f.NewSheet("Other"); err != nil {
		t.Fatal(err)
	}
	if err := f.SetCellValue("Other", "A1", "@SUM(A1)"); err != nil {
		t.Fatal(err)
	}
	if err := f.SaveAs(xlsx); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()

	guarded := filepath.Join(dir, "guarded")
	if err := ExcelToCSV(xlsx, guarded); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(guarded, "Sheet1.csv"))
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "'=1+1\n" {
		t.Fatalf("guarded output = %q", b)
	}

	allowed := filepath.Join(dir, "allowed")
	if err := ExcelToCSV(xlsx, allowed, ExcelToCSVOptions{AllowFormulas: true}); err != nil {
		t.Fatal(err)
	}
	b, err = os.ReadFile(filepath.Join(allowed, "Sheet1.csv"))
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "=1+1\n" {
		t.Fatalf("allowed output = %q", b)
	}

	selected := filepath.Join(dir, "selected")
	if err := ExcelToCSV(xlsx, selected, ExcelToCSVOptions{Sheets: []string{"Other"}, FileNames: map[string]string{"Other": "chosen"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(selected, "chosen.csv")); err != nil {
		t.Fatal(err)
	}
	if err := ExcelToCSV(xlsx, dir, ExcelToCSVOptions{Sheets: []string{"missing"}}); err == nil {
		t.Fatal("missing sheet accepted")
	}
	if err := ExcelToCSV(xlsx, dir, ExcelToCSVOptions{}, ExcelToCSVOptions{}); err == nil {
		t.Fatal("multiple options accepted")
	}
	legacy := filepath.Join(dir, "legacy")
	if err := ExcelToCsv(xlsx, legacy, nil); err != nil {
		t.Fatal(err)
	}
	b, err = os.ReadFile(filepath.Join(legacy, "Sheet1.csv"))
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "=1+1\n" {
		t.Fatalf("legacy output changed: %q", b)
	}
}
