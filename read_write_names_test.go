package insyra

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// The file readers and writers take their settings in one optional options
// struct, and the zero value is the common file: a header row naming the
// columns and no row names (#213, ruled 2026-09-25).

func writeTempCSV(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "in.csv")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestReadCSVFileDefaultsToAHeaderRow(t *testing.T) {
	path := writeTempCSV(t, "name,age\nAmy,30\nBen,40\n")

	dt, err := ReadCSVFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := dt.ColNames(); !reflect.DeepEqual(got, []string{"name", "age"}) {
		t.Fatalf("columns %v, want the header row's names", got)
	}
	if rows, _ := dt.Size(); rows != 2 {
		t.Fatalf("%d rows, want 2", rows)
	}

	raw, err := ReadCSVFile(path, CSVReadOptions{NoHeaderRow: true})
	if err != nil {
		t.Fatal(err)
	}
	if rows, _ := raw.Size(); rows != 3 {
		t.Fatalf("NoHeaderRow read %d rows, want 3", rows)
	}

	named, err := ReadCSVFile(writeTempCSV(t, ",x\nr1,1\nr2,2\n"), CSVReadOptions{HasRowNames: true})
	if err != nil {
		t.Fatal(err)
	}
	if name, _ := named.GetRowNameByIndex(1); name != "r2" {
		t.Fatalf("row name %q, want r2", name)
	}
	if _, err := ReadCSVFile(path, CSVReadOptions{}, CSVReadOptions{}); err == nil {
		t.Fatal("two options structs were accepted")
	}
}

func TestReadCSVStringDefaultsToAHeaderRow(t *testing.T) {
	dt, err := ReadCSVString("name,age\nAmy,30\n")
	if err != nil {
		t.Fatal(err)
	}
	if got := dt.ColNames(); !reflect.DeepEqual(got, []string{"name", "age"}) {
		t.Fatalf("columns %v", got)
	}
	raw, err := ReadCSVString("1,2\n3,4\n", CSVReadOptions{NoHeaderRow: true})
	if err != nil {
		t.Fatal(err)
	}
	if rows, _ := raw.Size(); rows != 2 {
		t.Fatalf("%d rows, want 2", rows)
	}
	streamed := 0
	for batch, err := range StreamCSV(strings.NewReader("a\n1\n2\n3\n"), 2) {
		if err != nil {
			t.Fatal(err)
		}
		if got := batch.ColNames(); !reflect.DeepEqual(got, []string{"a"}) {
			t.Fatalf("batch columns %v", got)
		}
		streamed++
	}
	if streamed != 2 {
		t.Fatalf("%d batches, want 2", streamed)
	}
}

func TestToCSVWritesTheHeaderByDefault(t *testing.T) {
	dt := NewDataTable(NewDataList(1, 2).SetName("a"))
	dt.SetRowNames([]string{"r1", "r2"})
	dir := t.TempDir()
	read := func(name string, opts ...CSVWriteOptions) string {
		path := filepath.Join(dir, name)
		if err := dt.ToCSV(path, opts...); err != nil {
			t.Fatal(err)
		}
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	if got := read("default.csv"); got != "a\n1\n2\n" {
		t.Fatalf("default ToCSV wrote %q", got)
	}
	if got := read("bare.csv", CSVWriteOptions{NoHeaderRow: true}); got != "1\n2\n" {
		t.Fatalf("NoHeaderRow wrote %q", got)
	}
	if got := read("named.csv", CSVWriteOptions{HasRowNames: true}); got != ",a\nr1,1\nr2,2\n" {
		t.Fatalf("HasRowNames wrote %q", got)
	}
	if err := dt.ToCSV(filepath.Join(dir, "x.csv"), CSVWriteOptions{}, CSVWriteOptions{}); err == nil {
		t.Fatal("two options structs were accepted")
	}
}

// The old names stay one release as Deprecated wrappers and keep their old
// meaning, positional bools included.
func TestDeprecatedReadWriteNamesKeepTheirMeaning(t *testing.T) {
	path := writeTempCSV(t, "name,age\nAmy,30\n")
	newer, _ := ReadCSVFile(path)
	older, err := ReadCSV_File(path, false, true)
	if err != nil || !reflect.DeepEqual(older.ColNames(), newer.ColNames()) {
		t.Fatalf("ReadCSV_File(path, false, true) = %v, %v", older.ColNames(), err)
	}
	noHeader, _ := ReadCSV_File(path, false, false)
	if rows, _ := noHeader.Size(); rows != 2 {
		t.Fatalf("ReadCSV_File(path, false, false) read %d rows, want 2", rows)
	}
	s, _ := ReadCSV_String("x\n1\n", false, true)
	if !reflect.DeepEqual(s.ColNames(), []string{"x"}) {
		t.Fatalf("ReadCSV_String kept columns %v", s.ColNames())
	}

	dt := NewDataTable(NewDataList(1).SetName("a"))
	if dt.ToJSON_String(true) != dt.ToJSONString(true) || string(dt.ToJSON_Bytes(false)) != string(dt.ToJSONBytes(false)) {
		t.Fatal("the deprecated JSON spellings differ from the new ones")
	}
	jsonPath := filepath.Join(t.TempDir(), "d.json")
	if err := dt.ToJSON(jsonPath, true); err != nil {
		t.Fatal(err)
	}
	a, err1 := ReadJSONFile(jsonPath)
	b, err2 := ReadJSON_File(jsonPath)
	if err1 != nil || err2 != nil || a.ToJSONString(true) != b.ToJSONString(true) {
		t.Fatalf("ReadJSONFile and ReadJSON_File differ: %v %v", err1, err2)
	}
}

func TestExcelWriteOptionsWriteTheHeaderByDefault(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.xlsx")
	if err := NewDataTable(NewDataList(1).SetName("a")).ToExcel(path, ExcelWriteOptions{}); err != nil {
		t.Fatal(err)
	}
	if got := sheetCells(t, path, "Sheet1"); !reflect.DeepEqual(got, [][]string{{"a"}, {"1"}}) {
		t.Fatalf("cells %v, want the header then the value", got)
	}
}
