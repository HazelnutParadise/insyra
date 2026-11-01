package insyra

import (
	"bytes"
	"math"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

func writeInferWorkbook(t *testing.T, path string) {
	t.Helper()
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	for _, cell := range []struct {
		axis string
		v    any
	}{
		{"A1", "id"}, {"B1", "price"}, {"C1", "name"}, {"D1", "gaps"}, {"E1", "code"},
		{"A2", 1}, {"B2", 1.5}, {"C2", "apple"}, {"D2", 1}, {"E2", "007"},
		{"A3", 2}, {"B3", 2.5}, {"C3", "12"}, {"E3", "x1"},
		{"A4", 3}, {"B4", 3}, {"C4", "pear"},
	} {
		if err := f.SetCellValue("Sheet1", cell.axis, cell.v); err != nil {
			t.Fatal(err)
		}
	}
	if err := f.SaveAs(path); err != nil {
		t.Fatal(err)
	}
}

func TestReadExcelSheetInfersColumnTypes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "infer.xlsx")
	writeInferWorkbook(t, path)
	dt, err := ReadExcelSheet(path, "Sheet1", false, true)
	if err != nil {
		t.Fatal(err)
	}
	if got := dt.GetColByName("id").Data(); got[0] != int64(1) || got[2] != int64(3) {
		t.Errorf("id = %#v, want int64", got)
	}
	if got := dt.GetColByName("price").Data(); got[0] != 1.5 || got[2] != 3.0 {
		t.Errorf("price = %#v, want float64", got)
	}
	// One word keeps the whole column text, the rule CSV reading follows.
	if got := dt.GetColByName("name").Data(); got[1] != "12" {
		t.Errorf("name = %#v, want text", got)
	}
	// Excel leaves out cells past a row's last value, and they read as empty.
	got := dt.GetColByName("gaps").Data()
	if got[0] != 1.0 {
		t.Errorf("gaps[0] = %#v, want 1.0", got[0])
	}
	for _, i := range []int{1, 2} {
		if f, ok := got[i].(float64); !ok || !math.IsNaN(f) {
			t.Errorf("gaps[%d] = %#v, want NaN", i, got[i])
		}
	}
	if err := dt.Err(); err != nil {
		t.Error(err)
	}
}

func TestReadExcelSheetRawStringsKeepsText(t *testing.T) {
	path := filepath.Join(t.TempDir(), "raw.xlsx")
	writeInferWorkbook(t, path)
	dt, err := ReadExcelSheet(path, "Sheet1", false, true, true)
	if err != nil {
		t.Fatal(err)
	}
	if got := dt.GetColByName("id").Data(); got[0] != "1" {
		t.Errorf("id = %#v, want text", got)
	}
	explicit, err := ReadExcelSheet(path, "Sheet1", false, true, false)
	if err != nil {
		t.Fatal(err)
	}
	if got := explicit.GetColByName("id").Data(); got[0] != int64(1) {
		t.Errorf("rawStrings false: id = %#v, want int64", got)
	}
	if _, err := ReadExcelSheet(path, "Sheet1", false, true, true, false); err == nil || !strings.Contains(err.Error(), "rawStrings") {
		t.Errorf("two rawStrings values: err = %v", err)
	}
}

func TestReadExcelInfersLikeReadExcelSheet(t *testing.T) {
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
	content := buf.Bytes()
	dt, err := ReadExcel(bytes.NewReader(content), "Sheet1", false, true)
	if err != nil {
		t.Fatal(err)
	}
	if got := dt.GetColByName("n").Data(); got[0] != int64(7) {
		t.Errorf("n = %#v, want int64", got)
	}
	raw, err := ReadExcel(bytes.NewReader(content), "Sheet1", false, true, true)
	if err != nil {
		t.Fatal(err)
	}
	if got := raw.GetColByName("n").Data(); got[0] != "7" {
		t.Errorf("raw n = %#v, want text", got)
	}
}
