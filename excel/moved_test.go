package excel

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/xuri/excelize/v2"
)

func TestReadFromMatchesRead(t *testing.T) {
	book := excelize.NewFile()
	for r, row := range [][]any{{"name", "qty"}, {"apple", 3}, {"banana", 12}} {
		for c, v := range row {
			cell, _ := excelize.CoordinatesToCellName(c+1, r+1)
			if err := book.SetCellValue("Sheet1", cell, v); err != nil {
				t.Fatal(err)
			}
		}
	}
	path := filepath.Join(t.TempDir(), "book.xlsx")
	if err := book.SaveAs(path); err != nil {
		t.Fatal(err)
	}
	_ = book.Close()

	fromFile, err := Read(path, ReadOptions{Sheet: "Sheet1"})
	if err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	fromReader, err := ReadFrom(bytes.NewReader(content), ReadOptions{Sheet: "Sheet1"})
	if err != nil {
		t.Fatal(err)
	}
	tablesEqual(t, "ReadFrom", fromReader, fromFile)
}

func TestWriteOptionsWriteTheHeaderByDefault(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.xlsx")
	if err := Write(insyra.NewDataTable(insyra.NewDataList(1).SetName("a")), path, WriteOptions{}); err != nil {
		t.Fatal(err)
	}
	if got := sheetCells(t, path, "Sheet1"); !reflect.DeepEqual(got, [][]string{{"a"}, {"1"}}) {
		t.Fatalf("cells %v, want the header then the value", got)
	}
}

func tablesEqual(t *testing.T, label string, got, want *insyra.DataTable) {
	t.Helper()
	if !reflect.DeepEqual(got.ColNames(), want.ColNames()) {
		t.Fatalf("%s: column names %v, want %v", label, got.ColNames(), want.ColNames())
	}
	if !reflect.DeepEqual(got.To2DSlice(), want.To2DSlice()) {
		t.Fatalf("%s: cells %v, want %v", label, got.To2DSlice(), want.To2DSlice())
	}
}
