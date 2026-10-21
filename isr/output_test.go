package isr_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/isr"
)

func outputSample() *isr.DataTable {
	return isr.UseDT(insyra.NewDataTable(
		insyra.NewDataList("Alice", "Bob").SetName("name"),
		insyra.NewDataList(30, 25).SetName("age"),
	))
}

// The exported types are what a caller's own function can name (#237).
func takesTable(t *isr.DataTable) int { return t.NumRows() }
func takesList(l *isr.DataList) int   { return l.Len() }

func TestExportedTypesCanBeNamed(t *testing.T) {
	if got := takesTable(outputSample()); got != 2 {
		t.Fatalf("rows = %d, want 2", got)
	}
	if got := takesList(isr.DL.From(1, 2, 3)); got != 3 {
		t.Fatalf("len = %d, want 3", got)
	}
}

func TestToCSVFileRoundTrips(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.csv")
	if err := outputSample().To(isr.CSV{FilePath: path}); err != nil {
		t.Fatalf("To: %v", err)
	}
	back := isr.DT.From(isr.CSV{FilePath: path})
	if back.Err() != nil {
		t.Fatalf("read back: %v", back.Err())
	}
	if got := back.ColNames(); strings.Join(got, ",") != "name,age" {
		t.Fatalf("columns = %v", got)
	}
	if back.NumRows() != 2 {
		t.Fatalf("rows = %d, want 2", back.NumRows())
	}
}

func TestToCSVOutputOptsApply(t *testing.T) {
	var c isr.CSV
	c.OutputOpts = isr.CSV_outOpts{NoHeaderRow: true}
	if err := outputSample().To(&c); err != nil {
		t.Fatalf("To: %v", err)
	}
	if strings.Contains(c.String, "name") {
		t.Fatalf("NoHeaderRow ignored, got %q", c.String)
	}
	if !strings.HasPrefix(c.String, "Alice,30") {
		t.Fatalf("got %q", c.String)
	}

	var bom isr.CSV
	bom.OutputOpts.IncludeBOM = true
	if err := outputSample().To(&bom); err != nil {
		t.Fatalf("To: %v", err)
	}
	if !strings.HasPrefix(bom.String, "\uFEFF") {
		t.Fatalf("IncludeBOM ignored, got %q", bom.String)
	}
}

func TestToCSVInMemoryReadsBackThroughFrom(t *testing.T) {
	var c isr.CSV
	if err := outputSample().To(&c); err != nil {
		t.Fatalf("To: %v", err)
	}
	back := isr.DT.From(isr.CSV{String: c.String})
	if back.Err() != nil || back.NumRows() != 2 {
		t.Fatalf("read back: rows=%d err=%v", back.NumRows(), back.Err())
	}
}

func TestToJSONUsesNamesByDefaultAndIndicesOnRequest(t *testing.T) {
	var j isr.JSON
	if err := outputSample().To(&j); err != nil {
		t.Fatalf("To: %v", err)
	}
	if !strings.Contains(string(j.Bytes), `"name"`) {
		t.Fatalf("default should write names, got %s", j.Bytes)
	}

	var byIndex isr.JSON
	byIndex.OutputOpts.UseColIndex = true
	if err := outputSample().To(&byIndex); err != nil {
		t.Fatalf("To: %v", err)
	}
	if strings.Contains(string(byIndex.Bytes), `"name"`) || !strings.Contains(string(byIndex.Bytes), `"A"`) {
		t.Fatalf("UseColIndex ignored, got %s", byIndex.Bytes)
	}

	path := filepath.Join(t.TempDir(), "out.json")
	if err := outputSample().To(isr.JSON{FilePath: path}); err != nil {
		t.Fatalf("To file: %v", err)
	}
	back := isr.DT.From(isr.JSON{FilePath: path})
	if back.Err() != nil || back.NumRows() != 2 {
		t.Fatalf("read back: rows=%d err=%v", back.NumRows(), back.Err())
	}
}

func TestToExcelWritesTheNamedSheet(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.xlsx")
	if err := outputSample().To(isr.Excel{FilePath: path, SheetName: "Result"}); err != nil {
		t.Fatalf("To: %v", err)
	}
	back := isr.DT.From(isr.Excel{FilePath: path, SheetName: "Result"})
	if back.Err() != nil || back.NumRows() != 2 {
		t.Fatalf("read back: rows=%d err=%v", back.NumRows(), back.Err())
	}

	// The sheet now exists; the zero policy refuses, Replace overwrites.
	if err := outputSample().To(isr.Excel{FilePath: path, SheetName: "Result"}); err == nil {
		t.Fatal("writing an existing sheet with the zero policy should fail")
	}
	replace := isr.Excel{FilePath: path, SheetName: "Result"}
	replace.OutputOpts.IfSheetExists = insyra.SheetExistsReplace
	if err := outputSample().To(replace); err != nil {
		t.Fatalf("replace: %v", err)
	}
}

func TestToRefusesWhatItCannotWrite(t *testing.T) {
	cases := map[string]any{
		"int":                 42,
		"nil":                 nil,
		"CSV value, no path":  isr.CSV{},
		"JSON value, no path": isr.JSON{},
		"Excel, no path":      &isr.Excel{},
	}
	for name, target := range cases {
		if err := outputSample().To(target); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
	var nilTable *isr.DataTable
	if err := nilTable.To(&isr.CSV{}); err == nil {
		t.Error("nil table: want an error")
	}
}

func TestToReportsAnUnwritablePath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing", "out.csv")
	if err := outputSample().To(isr.CSV{FilePath: path}); err == nil {
		t.Fatal("want an error for a directory that does not exist")
	}
	if _, err := os.Stat(path); err == nil {
		t.Fatal("no file should have been written")
	}
}

// From names the type it was given and the key it could not use (#238).
func TestFromRowErrorNamesTheKey(t *testing.T) {
	got := isr.DT.From(isr.Row{"A": 1, 3.5: 2}).Err()
	if got == nil {
		t.Fatal("want an error")
		return
	}
	msg := got.Error()
	if !strings.Contains(msg, "isr.Row") || !strings.Contains(msg, "3.5") {
		t.Fatalf("message should name isr.Row and the key 3.5, got %q", msg)
	}
}
