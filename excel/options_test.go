package excel

import (
	"bytes"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/xuri/excelize/v2"
)

// A workbook whose first sheet is not "Sheet1", so reading the first sheet
// by default cannot pass by accident.
func twoSheetBook(t *testing.T) string {
	t.Helper()
	f := excelize.NewFile()
	if err := f.SetSheetName("Sheet1", "Q3"); err != nil {
		t.Fatal(err)
	}
	if _, err := f.NewSheet("Q1"); err != nil {
		t.Fatal(err)
	}
	for sheet, v := range map[string]int{"Q3": 3, "Q1": 1} {
		if err := f.SetSheetRow(sheet, "A1", &[]any{"v"}); err != nil {
			t.Fatal(err)
		}
		if err := f.SetSheetRow(sheet, "A2", &[]any{v}); err != nil {
			t.Fatal(err)
		}
	}
	path := filepath.Join(t.TempDir(), "book.xlsx")
	if err := f.SaveAs(path); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	return path
}

func TestReadWithoutOptionsReadsTheFirstSheet(t *testing.T) {
	dt, err := Read(twoSheetBook(t))
	if err != nil {
		t.Fatal(err)
	}
	if got := dt.GetColByName("v").Data(); !reflect.DeepEqual(got, []any{int64(3)}) {
		t.Fatalf("v = %#v, want the first sheet's [3]", got)
	}
}

func TestReadNamesASheet(t *testing.T) {
	dt, err := Read(twoSheetBook(t), ReadOptions{Sheet: "Q1"})
	if err != nil {
		t.Fatal(err)
	}
	if got := dt.GetColByName("v").Data(); !reflect.DeepEqual(got, []any{int64(1)}) {
		t.Fatalf("v = %#v, want Q1's [1]", got)
	}
}

func TestSheetNamesListsTheSheetsInOrder(t *testing.T) {
	got, err := SheetNames(twoSheetBook(t))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, []string{"Q3", "Q1"}) {
		t.Errorf("SheetNames = %v", got)
	}
	if _, err := SheetNames(filepath.Join(t.TempDir(), "missing.xlsx")); err == nil || !strings.Contains(err.Error(), "missing.xlsx") {
		t.Errorf("a missing workbook: got %v", err)
	}
}

func TestMoreThanOneOptionsValueIsRefused(t *testing.T) {
	dt := insyra.NewDataTable(insyra.NewDataList(1).SetName("v"))
	path := filepath.Join(t.TempDir(), "out.xlsx")
	if err := Write(dt, path, WriteOptions{}, WriteOptions{}); err == nil || !strings.Contains(err.Error(), "WriteOptions") {
		t.Errorf("Write: got %v", err)
	}
	var buf bytes.Buffer
	if err := WriteTo(dt, &buf, WriteOptions{}, WriteOptions{}); err == nil || !strings.Contains(err.Error(), "WriteOptions") {
		t.Errorf("WriteTo: got %v", err)
	}
	if _, err := ReadFrom(&buf, ReadOptions{}, ReadOptions{}); err == nil || !strings.Contains(err.Error(), "ReadOptions") {
		t.Errorf("ReadFrom: got %v", err)
	}
}

func TestANilTableIsRefused(t *testing.T) {
	var dt *insyra.DataTable
	if err := Write(dt, filepath.Join(t.TempDir(), "out.xlsx")); err == nil {
		t.Error("Write accepted a nil table")
	}
	if err := WriteTo(dt, &bytes.Buffer{}); err == nil {
		t.Error("WriteTo accepted a nil table")
	}
}

func TestWriteThenReadKeepsTheTable(t *testing.T) {
	dt := insyra.NewDataTable(
		insyra.NewDataList(1, 2).SetName("a"),
		insyra.NewDataList("x", "y").SetName("b"),
	)
	dt.SetRowNames([]string{"r1", "r2"})
	path := filepath.Join(t.TempDir(), "out.xlsx")
	if err := Write(dt, path, WriteOptions{HasRowNames: true}); err != nil {
		t.Fatal(err)
	}
	back, err := Read(path, ReadOptions{HasRowNames: true})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(back.ColNames(), []string{"a", "b"}) {
		t.Fatalf("columns %v", back.ColNames())
	}
	if !reflect.DeepEqual(back.To2DSlice(), [][]any{{int64(1), "x"}, {int64(2), "y"}}) {
		t.Fatalf("cells %v", back.To2DSlice())
	}
	if name, _ := back.GetRowNameByIndex(1); name != "r2" {
		t.Fatalf("row name %q, want r2", name)
	}
}
