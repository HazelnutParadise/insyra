package csvxl

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// Regression tests for #270 (C-11) through the deprecated names, which keep
// their positional pairing: a pairing that would put two CSVs in one sheet,
// or two sheets in one CSV, is refused before anything is written instead of
// mixing or overwriting data.

func TestDeprecatedCsvToExcelRefusesTwoCSVsForOneSheet(t *testing.T) {
	dir := t.TempDir()
	jan := writeNamedCSV(t, dir, "jan.csv", "jan1\njan2\n")
	feb := writeNamedCSV(t, dir, "feb.csv", "feb1\n")
	out := filepath.Join(dir, "out.xlsx")
	err := CsvToExcel([]string{jan, feb}, []string{"Sales", "Sales"}, out)
	if err == nil || !strings.Contains(err.Error(), "Sales") {
		t.Fatalf("two CSVs for one sheet: got %v", err)
	}
	if _, statErr := os.Stat(out); !os.IsNotExist(statErr) {
		t.Errorf("no workbook should be written: %v", statErr)
	}
}

func TestDeprecatedAppendCsvToExcelRefusesTwoCSVsForOneSheet(t *testing.T) {
	dir := t.TempDir()
	a := writeNamedCSV(t, dir, "a.csv", "a\n")
	c := writeNamedCSV(t, dir, "c.csv", "c\n")
	book := filepath.Join(dir, "book.xlsx")
	if err := CsvToExcel([]string{a}, nil, book); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(book)
	if err := AppendCsvToExcel([]string{a, c}, []string{"Z", "z"}, book); err == nil {
		t.Fatal("two CSVs for one sheet were appended")
	}
	if after, _ := os.ReadFile(book); !bytes.Equal(before, after) {
		t.Error("the workbook changed")
	}
}

func TestDeprecatedExcelToCsvRefusesTwoSheetsForOneFile(t *testing.T) {
	book := threeSheetBook(t)
	out := filepath.Join(t.TempDir(), "out")
	err := ExcelToCsv(book, out, []string{"same", "same"})
	if err == nil || !strings.Contains(err.Error(), "same.csv") {
		t.Fatalf("two sheets for one file: got %v", err)
	}
	if entries, _ := os.ReadDir(out); len(entries) != 0 {
		t.Errorf("nothing should be written, got %d files", len(entries))
	}
}

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
	if err := ExcelToCsv(threeSheetBook(t), csvDir, []string{"x"}, ExcelToCsvOptions{FileNames: map[string]string{"Q1": "y"}}); err == nil {
		t.Error("ExcelToCsv accepted file names in both csvNames and FileNames")
	}
}
