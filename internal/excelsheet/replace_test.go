package excelsheet

import (
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestReplaceOnlySheetNamedPlaceholder(t *testing.T) {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	old := f.GetSheetName(0)
	if err := f.SetCellValue(old, "A1", "old"); err != nil {
		t.Fatal(err)
	}
	if err := f.SetSheetName(old, "__insyra_placeholder__"); err != nil {
		t.Fatal(err)
	}
	if err := Replace(f, "__insyra_placeholder__", 0); err != nil {
		t.Fatal(err)
	}
	if f.SheetCount != 1 {
		t.Fatalf("sheet count=%d", f.SheetCount)
	}
	if got, err := f.GetCellValue("__insyra_placeholder__", "A1"); err != nil || got != "" {
		t.Fatalf("cell=%q err=%v", got, err)
	}
}
