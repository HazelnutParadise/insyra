package csvxl

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/xuri/excelize/v2"
)

// threeSheetBook writes a workbook whose sheets Q1, Q2 and Q3 each hold their
// own name in A1.
func threeSheetBook(t *testing.T) string {
	t.Helper()
	return sheetBook(t, "Q1", "Q2", "Q3")
}

func sheetBook(t *testing.T, sheets ...string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "book.xlsx")
	writeSheetBook(t, path, sheets...)
	return path
}

func writeSheetBook(t *testing.T, path string, sheets ...string) {
	t.Helper()
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	for i, s := range sheets {
		if i == 0 {
			if err := f.SetSheetName("Sheet1", s); err != nil {
				t.Fatal(err)
			}
		} else if _, err := f.NewSheet(s); err != nil {
			t.Fatal(err)
		}
		if err := f.SetCellStr(s, "A1", s); err != nil {
			t.Fatal(err)
		}
	}
	if err := f.SaveAs(path); err != nil {
		t.Fatal(err)
	}
}

// dirFiles maps each file name in dir to its content.
func dirFiles(t *testing.T, dir string) map[string]string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, e := range entries {
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		got[e.Name()] = string(b)
	}
	return got
}

// bookSheets returns the sheet names of a workbook and each sheet's rows.
func bookSheets(t *testing.T, path string) ([]string, map[string][][]string) {
	t.Helper()
	f, err := excelize.OpenFile(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	rows := map[string][][]string{}
	for _, s := range f.GetSheetList() {
		r, err := f.GetRows(s)
		if err != nil {
			t.Fatal(err)
		}
		rows[s] = r
	}
	return f.GetSheetList(), rows
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func writeNamedCSV(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}
