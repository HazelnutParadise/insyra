package csvxl

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// Tests for #270 (C-11): a CSV and its sheet are named together, so nothing
// is paired by position, and a pairing that would lose data is refused before
// anything is written.

func TestCSVToExcelPairsEachCSVWithItsSheet(t *testing.T) {
	dir := t.TempDir()
	jan := writeNamedCSV(t, dir, "jan.csv", "jan1\n")
	feb := writeNamedCSV(t, dir, "feb.csv", "feb1\n")
	out := filepath.Join(dir, "out.xlsx")
	if err := CSVToExcel([]SheetFromCSV{{Path: jan, Sheet: "Sales"}, {Path: feb}}, out); err != nil {
		t.Fatal(err)
	}
	names, rows := bookSheets(t, out)
	if !reflect.DeepEqual(names, []string{"Sales", "feb"}) {
		t.Fatalf("sheets = %v, want [Sales feb]", names)
	}
	if rows["Sales"][0][0] != "jan1" || rows["feb"][0][0] != "feb1" {
		t.Errorf("rows = %v", rows)
	}
}

func TestCSVToExcelRefusesTwoCSVsForOneSheet(t *testing.T) {
	dir := t.TempDir()
	jan := writeNamedCSV(t, dir, "jan.csv", "jan1\njan2\n")
	feb := writeNamedCSV(t, dir, "feb.csv", "feb1\n")
	a := writeNamedCSV(t, dir, "a.csv", "a\n")
	cases := map[string][]SheetFromCSV{
		"same name":                 {{Path: jan, Sheet: "Sales"}, {Path: feb, Sheet: "Sales"}},
		"names differ only in case": {{Path: jan, Sheet: "Sales"}, {Path: feb, Sheet: "SALES"}},
		"a default name collides":   {{Path: a}, {Path: feb, Sheet: "A"}},
	}
	for name, sheets := range cases {
		t.Run(name, func(t *testing.T) {
			out := filepath.Join(t.TempDir(), "out.xlsx")
			err := CSVToExcel(sheets, out)
			if err == nil {
				t.Fatal("two CSVs for one sheet were converted")
				return
			}
			for _, s := range sheets {
				if !strings.Contains(err.Error(), s.Path) {
					t.Errorf("the error should name %s: %v", s.Path, err)
				}
			}
			if _, statErr := os.Stat(out); !os.IsNotExist(statErr) {
				t.Errorf("no workbook should be written: %v", statErr)
			}
		})
	}
}

func TestAppendCSVToExcelRefusesTwoCSVsForOneSheet(t *testing.T) {
	dir := t.TempDir()
	a := writeNamedCSV(t, dir, "a.csv", "a\n")
	c := writeNamedCSV(t, dir, "c.csv", "c\n")
	book := filepath.Join(dir, "book.xlsx")
	if err := CSVToExcel([]SheetFromCSV{{Path: a}}, book); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(book)
	err := AppendCSVToExcel([]SheetFromCSV{{Path: a, Sheet: "Z"}, {Path: c, Sheet: "Z"}}, book)
	if err == nil || !strings.Contains(err.Error(), "Z") {
		t.Fatalf("two CSVs for one sheet: got %v", err)
	}
	if after, _ := os.ReadFile(book); !bytes.Equal(before, after) {
		t.Error("the workbook changed")
	}
}

func TestExcelToCSVNamesAFileBySheet(t *testing.T) {
	out := filepath.Join(t.TempDir(), "out")
	if err := ExcelToCSV(threeSheetBook(t), out, ExcelToCSVOptions{FileNames: map[string]string{"Q2": "second"}}); err != nil {
		t.Fatal(err)
	}
	got := dirFiles(t, out)
	want := map[string]string{"Q1.csv": "Q1\n", "second.csv": "Q2\n", "Q3.csv": "Q3\n"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("files = %v, want %v", got, want)
	}
}

func TestExcelToCSVFileNamesKeepTheirExtension(t *testing.T) {
	out := filepath.Join(t.TempDir(), "out")
	err := ExcelToCSV(threeSheetBook(t), out, ExcelToCSVOptions{
		Sheets:    []string{"Q1", "Q2"},
		FileNames: map[string]string{"Q1": "report.txt", "Q2": "REPORT2.CSV"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := sortedKeys(dirFiles(t, out)); !reflect.DeepEqual(got, []string{"REPORT2.CSV", "report.txt"}) {
		t.Errorf("files = %v", got)
	}
}

func TestExcelToCSVRefusesAFileNameForASheetNotConverted(t *testing.T) {
	out := filepath.Join(t.TempDir(), "out")
	err := ExcelToCSV(threeSheetBook(t), out, ExcelToCSVOptions{
		Sheets:    []string{"Q1"},
		FileNames: map[string]string{"Q2": "x"},
	})
	if err == nil || !strings.Contains(err.Error(), "Q2") {
		t.Fatalf("a name for an unconverted sheet: got %v", err)
	}
	if got := dirFiles(t, out); len(got) != 0 {
		t.Errorf("nothing should be written, got %v", got)
	}
}

func TestExcelToCSVRefusesTwoSheetsForOneFile(t *testing.T) {
	cases := map[string]map[string]string{
		"same name":                 {"Q1": "same", "Q2": "same.csv"},
		"names differ only in case": {"Q1": "Same", "Q2": "SAME"},
		"a default name collides":   {"Q1": "Q2.csv"},
	}
	for name, fileNames := range cases {
		t.Run(name, func(t *testing.T) {
			out := filepath.Join(t.TempDir(), "out")
			err := ExcelToCSV(threeSheetBook(t), out, ExcelToCSVOptions{FileNames: fileNames})
			if err == nil {
				t.Fatal("two sheets for one file were converted")
				return
			}
			if !strings.Contains(err.Error(), "Q1") || !strings.Contains(err.Error(), "Q2") {
				t.Errorf("the error should name both sheets: %v", err)
			}
			if got := dirFiles(t, out); len(got) != 0 {
				t.Errorf("nothing should be written, got %v", got)
			}
		})
	}
}

func TestExcelToCSVChecksEveryNameBeforeWriting(t *testing.T) {
	out := filepath.Join(t.TempDir(), "out")
	err := ExcelToCSV(threeSheetBook(t), out, ExcelToCSVOptions{FileNames: map[string]string{"Q3": "../escape"}})
	if err == nil {
		t.Fatal("a name leaving the output directory was accepted")
	}
	if got := dirFiles(t, out); len(got) != 0 {
		t.Errorf("Q1 and Q2 should not be written before Q3's name is refused, got %v", got)
	}
	if _, statErr := os.Stat(filepath.Join(filepath.Dir(out), "escape")); !os.IsNotExist(statErr) {
		t.Errorf("a file escaped the output directory: %v", statErr)
	}
}

func TestExcelDirToCSVRefusesFileNames(t *testing.T) {
	dir := t.TempDir()
	writeSheetBook(t, filepath.Join(dir, "a.xlsx"), "Q1")
	out := filepath.Join(t.TempDir(), "out")
	if err := ExcelDirToCSV(dir, out, ExcelToCSVOptions{FileNames: map[string]string{"Q1": "x"}}); err == nil {
		t.Fatal("ExcelDirToCSV accepted FileNames")
	}
	if got := dirFiles(t, out); len(got) != 0 {
		t.Errorf("nothing should be written, got %v", got)
	}
}

func TestExcelDirToCSVRefusesTwoWorkbooksForOneFile(t *testing.T) {
	dir := t.TempDir()
	writeSheetBook(t, filepath.Join(dir, "a.xlsx"), "b_c")
	writeSheetBook(t, filepath.Join(dir, "a_b.xlsx"), "c")
	out := filepath.Join(t.TempDir(), "out")
	err := ExcelDirToCSV(dir, out)
	if err == nil || !strings.Contains(err.Error(), "a_b_c.csv") {
		t.Fatalf("two workbooks writing one file: got %v", err)
	}
	if got := dirFiles(t, out); len(got) != 0 {
		t.Errorf("nothing should be written, got %v", got)
	}
}

func TestSheetNamesListsTheSheetsInOrder(t *testing.T) {
	got, err := SheetNames(sheetBook(t, "Q3", "Q1", "Q2"))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, []string{"Q3", "Q1", "Q2"}) {
		t.Errorf("SheetNames = %v", got)
	}
	if _, err := SheetNames(filepath.Join(t.TempDir(), "missing.xlsx")); err == nil || !strings.Contains(err.Error(), "missing.xlsx") {
		t.Errorf("a missing workbook: got %v", err)
	}
}
