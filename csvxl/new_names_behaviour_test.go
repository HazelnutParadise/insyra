package csvxl

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

// Regression tests for #268 (C-6), written against the new csvxl names. An
// encoding nothing in the shared decoder table handles is refused by that one
// table, so every conversion names the encoding in its error and leaves the
// output alone: no workbook written where there was none, and an existing
// workbook left byte for byte as it was.

func writeNamedCSV(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestConversionsRefuseAnUnknownEncoding(t *testing.T) {
	csvPath := writeNamedCSV(t, t.TempDir(), "a.csv", "x,y\n1,2\n")

	out := filepath.Join(t.TempDir(), "out.xlsx")
	err := CSVToExcel([]string{csvPath}, nil, out, "klingon-1")
	if err == nil {
		t.Fatal("CSVToExcel accepted an encoding no decoder handles")
	}
	if !strings.Contains(err.Error(), "klingon-1") {
		t.Errorf("the error should name the encoding: %v", err)
	}
	if _, statErr := os.Stat(out); !os.IsNotExist(statErr) {
		t.Errorf("no workbook should be written when every file fails: %v", statErr)
	}

	book := filepath.Join(t.TempDir(), "book.xlsx")
	f := excelize.NewFile()
	if err := f.SetCellStr("Sheet1", "A1", "keep"); err != nil {
		t.Fatal(err)
	}
	if err := f.SaveAs(book); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(book)
	if err != nil {
		t.Fatal(err)
	}

	err = AppendCSVToExcel([]string{csvPath}, []string{"Sheet1"}, book, "klingon-1")
	if err == nil {
		t.Fatal("AppendCSVToExcel accepted an encoding no decoder handles")
	}
	if !strings.Contains(err.Error(), "klingon-1") {
		t.Errorf("the error should name the encoding: %v", err)
	}
	after, err := os.ReadFile(book)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(after, before) {
		t.Error("an append that converted nothing rewrote the workbook")
	}

	csvDir := t.TempDir()
	writeNamedCSV(t, csvDir, "a.csv", "x,y\n1,2\n")
	out2 := filepath.Join(t.TempDir(), "dir.xlsx")
	err = CSVDirToExcel(csvDir, out2, "klingon-1")
	if err == nil {
		t.Fatal("CSVDirToExcel accepted an encoding no decoder handles")
	}
	if !strings.Contains(err.Error(), "klingon-1") {
		t.Errorf("the error should name the encoding: %v", err)
	}
	if _, statErr := os.Stat(out2); !os.IsNotExist(statErr) {
		t.Errorf("no workbook should be written when every file fails: %v", statErr)
	}

	if _, err := ReadCSVToString(csvPath, "klingon-1"); err == nil {
		t.Error("ReadCSVToString accepted an encoding no decoder handles")
	} else if !strings.Contains(err.Error(), "klingon-1") {
		t.Errorf("the error should name the encoding: %v", err)
	}
}

// Each old name is a one-line call to its new one, so the workbook it writes
// has to be the same workbook, sheet for sheet and cell for cell.
func TestDeprecatedCSVXLNamesGiveTheSameWorkbook(t *testing.T) {
	dir := t.TempDir()
	csvPath := writeNamedCSV(t, dir, "data.csv", "a,b\n1,2\n3,4\n")
	old := filepath.Join(dir, "old.xlsx")
	current := filepath.Join(dir, "current.xlsx")

	if err := CsvToExcel([]string{csvPath}, nil, old, UTF8); err != nil {
		t.Fatalf("CsvToExcel: %v", err)
	}
	if err := CSVToExcel([]string{csvPath}, nil, current, UTF8); err != nil {
		t.Fatalf("CSVToExcel: %v", err)
	}

	oldBook, err := excelize.OpenFile(old)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = oldBook.Close() }()
	currentBook, err := excelize.OpenFile(current)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = currentBook.Close() }()

	oldSheets := oldBook.GetSheetList()
	currentSheets := currentBook.GetSheetList()
	if !reflect.DeepEqual(oldSheets, currentSheets) {
		t.Fatalf("sheets differ: %q against %q", oldSheets, currentSheets)
	}
	for _, sheet := range oldSheets {
		oldRows, err := oldBook.GetRows(sheet)
		if err != nil {
			t.Fatal(err)
		}
		currentRows, err := currentBook.GetRows(sheet)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(oldRows, currentRows) {
			t.Errorf("rows of sheet %s differ: %q against %q", sheet, oldRows, currentRows)
		}
	}
}

// A directory is a sheet list: one sheet per CSV, named after the file.
func TestCSVDirToExcelWritesOneSheetPerFile(t *testing.T) {
	dir := t.TempDir()
	writeNamedCSV(t, dir, "a.csv", "h\n1\n")
	writeNamedCSV(t, dir, "b.csv", "h\n2\n")
	out := filepath.Join(t.TempDir(), "out.xlsx")

	if err := CSVDirToExcel(dir, out); err != nil {
		t.Fatalf("CSVDirToExcel: %v", err)
	}

	book, err := excelize.OpenFile(out)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = book.Close() }()

	sheets := book.GetSheetList()
	if !reflect.DeepEqual(sheets, []string{"a", "b"}) {
		t.Fatalf("sheets = %q; want [a b]", sheets)
	}
	for sheet, want := range map[string][][]string{
		"a": {{"h"}, {"1"}},
		"b": {{"h"}, {"2"}},
	} {
		rows, err := book.GetRows(sheet)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(rows, want) {
			t.Errorf("rows of sheet %s = %q; want %q", sheet, rows, want)
		}
	}
}

// A name no decoder handles is refused before any file is looked at, so an
// empty file list cannot hide it and the workbook is not written or rewritten.
func TestAnUnknownEncodingIsRefusedWithNoFiles(t *testing.T) {
	out := filepath.Join(t.TempDir(), "out.xlsx")
	err := CSVToExcel(nil, nil, out, "klingon-1")
	if err == nil {
		t.Fatal("CSVToExcel accepted an encoding no decoder handles")
	}
	if !strings.Contains(err.Error(), "klingon-1") {
		t.Errorf("the error should name the encoding: %v", err)
	}
	if _, statErr := os.Stat(out); !os.IsNotExist(statErr) {
		t.Errorf("no workbook should be written for an unknown encoding: %v", statErr)
	}

	emptyDir := t.TempDir()
	out2 := filepath.Join(t.TempDir(), "dir.xlsx")
	err = CSVDirToExcel(emptyDir, out2, "klingon-1")
	if err == nil {
		t.Fatal("CSVDirToExcel accepted an encoding no decoder handles")
	}
	if !strings.Contains(err.Error(), "klingon-1") {
		t.Errorf("the error should name the encoding: %v", err)
	}
	if _, statErr := os.Stat(out2); !os.IsNotExist(statErr) {
		t.Errorf("no workbook should be written for an unknown encoding: %v", statErr)
	}

	book := filepath.Join(t.TempDir(), "book.xlsx")
	f := excelize.NewFile()
	if err := f.SetCellStr("Sheet1", "A1", "keep"); err != nil {
		t.Fatal(err)
	}
	if err := f.SaveAs(book); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(book)
	if err != nil {
		t.Fatal(err)
	}

	err = AppendCSVToExcel(nil, nil, book, "klingon-1")
	if err == nil {
		t.Fatal("AppendCSVToExcel accepted an encoding no decoder handles")
	}
	if !strings.Contains(err.Error(), "klingon-1") {
		t.Errorf("the error should name the encoding: %v", err)
	}
	after, err := os.ReadFile(book)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(after, before) {
		t.Error("an append with an unknown encoding rewrote the workbook")
	}
}

// The encoding is one argument, so it is one mistake however many files it
// would have been applied to.
func TestAnUnknownEncodingIsNamedOnce(t *testing.T) {
	dir := t.TempDir()
	var files []string
	for _, name := range []string{"a.csv", "b.csv", "c.csv"} {
		files = append(files, writeNamedCSV(t, dir, name, "x\n1\n"))
	}
	out := filepath.Join(t.TempDir(), "out.xlsx")

	err := CSVToExcel(files, nil, out, "klingon-1")
	if err == nil {
		t.Fatal("CSVToExcel accepted an encoding no decoder handles")
	}
	if n := strings.Count(err.Error(), "klingon-1"); n != 1 {
		t.Errorf("the error names the encoding %d times, want once: %v", n, err)
	}
}

// Checking the encoding first must not change what a valid one does with no
// files: the workbook is still written.
func TestAKnownEncodingWithNoFilesStillWritesAWorkbook(t *testing.T) {
	out := filepath.Join(t.TempDir(), "out.xlsx")
	if err := CSVToExcel(nil, nil, out, "big5"); err != nil {
		t.Fatalf("CSVToExcel: %v", err)
	}
	if _, err := os.Stat(out); err != nil {
		t.Errorf("the workbook should be written: %v", err)
	}
}
