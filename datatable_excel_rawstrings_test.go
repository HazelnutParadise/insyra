package insyra

import (
	"bytes"
	"math"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

func writeRawStringsWorkbook(t *testing.T, path string) {
	t.Helper()
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	for _, cell := range []struct {
		axis string
		v    any
	}{
		{"A1", "id"}, {"B1", "name"}, {"C1", "gaps"},
		{"A2", 1}, {"B2", "apple"}, {"C2", 1},
		{"A3", 2}, {"B3", "12"},
	} {
		if err := f.SetCellValue("Sheet1", cell.axis, cell.v); err != nil {
			t.Fatal(err)
		}
	}
	if err := f.SaveAs(path); err != nil {
		t.Fatal(err)
	}
}

func TestReadExcelSheetRawStrings(t *testing.T) {
	path := filepath.Join(t.TempDir(), "raw.xlsx")
	writeRawStringsWorkbook(t, path)

	left, err := ReadExcelSheet(path, "Sheet1", false, true)
	if err != nil {
		t.Fatal(err)
	}
	if got := left.GetColByName("id").Data()[0]; got != "1" {
		t.Errorf("left out: id = %#v, want text", got)
	}
	kept, err := ReadExcelSheet(path, "Sheet1", false, true, true)
	if err != nil {
		t.Fatal(err)
	}
	if got := kept.GetColByName("id").Data()[0]; got != "1" {
		t.Errorf("true: id = %#v, want text", got)
	}
	typed, err := ReadExcelSheet(path, "Sheet1", false, true, false)
	if err != nil {
		t.Fatal(err)
	}
	if got := typed.GetColByName("id").Data()[0]; got != int64(1) {
		t.Errorf("false: id = %#v, want int64", got)
	}
	if got := typed.GetColByName("name").Data()[1]; got != "12" {
		t.Errorf("false: a column with a word stays text, got %#v", got)
	}
	gaps := typed.GetColByName("gaps").Data()
	if gaps[0] != 1.0 {
		t.Errorf("false: gaps[0] = %#v, want 1.0", gaps[0])
	}
	if f, ok := gaps[1].(float64); !ok || !math.IsNaN(f) {
		t.Errorf("false: gaps[1] = %#v, want NaN", gaps[1])
	}
	if _, err := ReadExcelSheet(path, "Sheet1", false, true, true, false); err == nil || !strings.Contains(err.Error(), "rawStrings") {
		t.Errorf("two values: err = %v", err)
	}
}

func TestReadExcelRawStrings(t *testing.T) {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	if err := f.SetSheetRow("Sheet1", "A1", &[]any{"n"}); err != nil {
		t.Fatal(err)
	}
	if err := f.SetCellValue("Sheet1", "A2", 7); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		t.Fatal(err)
	}
	typed, err := ReadExcel(bytes.NewReader(buf.Bytes()), "Sheet1", false, true, false)
	if err != nil {
		t.Fatal(err)
	}
	if got := typed.GetColByName("n").Data()[0]; got != int64(7) {
		t.Errorf("false: n = %#v, want int64", got)
	}
	if _, err := ReadExcel(bytes.NewReader(buf.Bytes()), "Sheet1", false, true, true, true); err == nil {
		t.Error("two values should be an error")
	}
}
