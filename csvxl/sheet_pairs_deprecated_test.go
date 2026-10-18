package csvxl

import (
	"path/filepath"
	"reflect"
	"testing"
)

// The deprecated names keep their released behaviour on the 0.3.x line:
// positional pairing, and no refusal of two CSVs for one sheet or two sheets
// for one file. Only the new names refuse those (#270, C-11).

func TestDeprecatedNamesKeepPositionalPairing(t *testing.T) {
	dir := t.TempDir()
	a := writeNamedCSV(t, dir, "a.csv", "a\n")
	b := writeNamedCSV(t, dir, "b.csv", "b\n")
	c := writeNamedCSV(t, dir, "c.csv", "c\n")
	out := filepath.Join(dir, "out.xlsx")
	if err := CsvToExcel([]string{a, b, c}, []string{"Sales", "Costs"}, out); err != nil {
		t.Fatal(err)
	}
	if names, _ := bookSheets(t, out); !reflect.DeepEqual(names, []string{"Sales", "Costs", "c"}) {
		t.Errorf("CsvToExcel sheets = %v, want [Sales Costs c]", names)
	}

	csvDir := filepath.Join(t.TempDir(), "out")
	if err := ExcelToCsv(threeSheetBook(t), csvDir, []string{"first", "second"}); err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"first.csv": "Q1\n", "second.csv": "Q2\n", "Q3.csv": "Q3\n"}
	if got := dirFiles(t, csvDir); !reflect.DeepEqual(got, want) {
		t.Errorf("ExcelToCsv files = %v, want %v", got, want)
	}
}

func TestDeprecatedNamesStillAcceptADuplicateTarget(t *testing.T) {
	dir := t.TempDir()
	jan := writeNamedCSV(t, dir, "jan.csv", "jan1\njan2\n")
	feb := writeNamedCSV(t, dir, "feb.csv", "feb1\n")
	out := filepath.Join(dir, "out.xlsx")
	if err := CsvToExcel([]string{jan, feb}, []string{"Sales", "Sales"}, out); err != nil {
		t.Fatalf("CsvToExcel: %v", err)
	}
	if names, _ := bookSheets(t, out); !reflect.DeepEqual(names, []string{"Sales"}) {
		t.Errorf("CsvToExcel sheets = %v, want [Sales]", names)
	}
	if err := AppendCsvToExcel([]string{jan, feb}, []string{"Z", "Z"}, out); err != nil {
		t.Fatalf("AppendCsvToExcel: %v", err)
	}

	csvDir := filepath.Join(t.TempDir(), "out")
	if err := ExcelToCsv(threeSheetBook(t), csvDir, []string{"same", "same"}); err != nil {
		t.Fatalf("ExcelToCsv: %v", err)
	}
	if got := dirFiles(t, csvDir); got["same.csv"] != "Q2\n" {
		t.Errorf("ExcelToCsv files = %v, want same.csv holding Q2", got)
	}
}
