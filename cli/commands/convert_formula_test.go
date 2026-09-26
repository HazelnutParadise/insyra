package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

// convert xlsx->csv guards formula-like text by default, like save; the
// option turns the guard off, and an argument convert does not understand is
// an error instead of being ignored.
func TestConvert_ExcelToCsvGuardsFormulasByDefault(t *testing.T) {
	dir := t.TempDir()
	xlsx := filepath.Join(dir, "in.xlsx")
	f := excelize.NewFile()
	if err := f.SetCellStr("Sheet1", "A1", "=1+1"); err != nil {
		t.Fatal(err)
	}
	if err := f.SaveAs(xlsx); err != nil {
		t.Fatal(err)
	}
	ctx := newTestExecContext(t)

	guarded := filepath.Join(dir, "guarded.csv")
	if err := runConvertCommand(ctx, []string{xlsx, guarded}); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(guarded); string(b) != "'=1+1\n" {
		t.Fatalf("default convert wrote %q", b)
	}
	raw := filepath.Join(dir, "raw.csv")
	if err := runConvertCommand(ctx, []string{xlsx, raw, "allowformulas", "true"}); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(raw); string(b) != "=1+1\n" {
		t.Fatalf("allowformulas true wrote %q", b)
	}
	if err := runConvertCommand(ctx, []string{xlsx, raw, "bogus"}); err == nil || !strings.Contains(err.Error(), "bogus") {
		t.Fatalf("an unknown argument: got %v", err)
	}
	if err := runConvertCommand(ctx, []string{guarded, filepath.Join(dir, "x.xlsx"), "allowformulas", "true"}); err == nil {
		t.Fatal("allowformulas was accepted for csv->xlsx")
	}
}
