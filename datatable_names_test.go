package insyra

import (
	"log/slog"
	"reflect"
	"strings"
	"testing"
)

func twoColTable() *DataTable {
	return NewDataTable(NewDataList(1, 2).SetName("a"), NewDataList(3, 4).SetName("b"))
}

func warningsMentioning(h *recordingHandler, fragment string) int {
	n := 0
	for _, r := range h.snapshot() {
		if r.level == slog.LevelWarn && strings.Contains(r.message, fragment) {
			n++
		}
	}
	return n
}

func assertNames(t *testing.T, label string, got, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s: got %q, want %q", label, got, want)
	}
}

func TestSetColNamesKeepsANameTheColumnAlreadyHas(t *testing.T) {
	dt := twoColTable().SetColNames([]string{"a", "b"})
	assertNames(t, "same names", dt.ColNames(), []string{"a", "b"})

	dt = twoColTable().SetColNames([]string{"b", "a"})
	assertNames(t, "swapped names", dt.ColNames(), []string{"b", "a"})

	dt = twoColTable().SetColNames([]string{"x", "x"})
	assertNames(t, "repeated in the call", dt.ColNames(), []string{"x", "x_1"})
	if dt.Err() != nil {
		t.Fatalf("unexpected error: %v", dt.Err())
	}
}

func TestSetColNamesIgnoresExtraNamesWithAWarning(t *testing.T) {
	h, _ := useRecordingLogger(t, LogLevelWarning)
	dt := twoColTable().SetColNames([]string{"x", "y", "z"})
	if dt.NumCols() != 2 {
		t.Fatalf("NumCols = %d, want 2", dt.NumCols())
	}
	assertNames(t, "extra names", dt.ColNames(), []string{"x", "y"})
	if dt.Err() != nil {
		t.Fatalf("Err = %v, want nil", dt.Err())
	}
	if warningsMentioning(h, "ignored") == 0 {
		t.Fatalf("no warning about the ignored name: %+v", h.snapshot())
	}
}

func TestSetColNamesWithFewerNamesClearsTheRest(t *testing.T) {
	dt := twoColTable().SetColNames([]string{"x"})
	assertNames(t, "fewer names", dt.ColNames(), []string{"x", ""})
}

func TestSingleColumnRenamesSkipTheColumnItself(t *testing.T) {
	assertNames(t, "ByIndex", twoColTable().SetColNameByIndex("A", "a").ColNames(), []string{"a", "b"})
	assertNames(t, "ByNumber", twoColTable().SetColNameByNumber(1, "b").ColNames(), []string{"a", "b"})
	assertNames(t, "ChangeColName", twoColTable().ChangeColName("a", "a").ColNames(), []string{"a", "b"})
	assertNames(t, "clash still suffixed", twoColTable().SetColNameByNumber(0, "b").ColNames(), []string{"b_1", "b"})
}

func TestSetRowToColNamesAcceptsSwappedNames(t *testing.T) {
	dt := twoColTable()
	dt.AppendRowsFromDataList(NewDataList("b", "a"))
	dt.SetRowToColNames(2)
	assertNames(t, "row to names", dt.ColNames(), []string{"b", "a"})
}

func TestARenamedDuplicateColumnWarns(t *testing.T) {
	h, _ := useRecordingLogger(t, LogLevelWarning)
	dt := twoColTable().AppendCols(NewDataList(5, 6).SetName("a"))
	assertNames(t, "append", dt.ColNames(), []string{"a", "b", "a_1"})
	if dt.Err() != nil {
		t.Fatalf("Err = %v, want nil", dt.Err())
	}
	if warningsMentioning(h, "a_1") == 0 {
		t.Fatalf("no warning naming a_1: %+v", h.snapshot())
	}
}

func TestSetRowNamesKeepsNamesTheRowsAlreadyHave(t *testing.T) {
	dt := twoColTable().SetRowNames([]string{"r1", "r2"})
	dt.SetRowNames([]string{"r1", "r2"})
	assertNames(t, "same", dt.RowNames(), []string{"r1", "r2"})
	dt.SetRowNames([]string{"r2", "r1"})
	assertNames(t, "swapped", dt.RowNames(), []string{"r2", "r1"})
	dt.SetRowNameByIndex(0, "r2")
	assertNames(t, "ByIndex own name", dt.RowNames(), []string{"r2", "r1"})
}

func TestSetRowNamesWarnsAboutIgnoredAndRenamedNames(t *testing.T) {
	h, _ := useRecordingLogger(t, LogLevelWarning)
	dt := twoColTable().SetRowNames([]string{"r1", "r2", "r3"})
	assertNames(t, "extra", dt.RowNames(), []string{"r1", "r2"})
	if warningsMentioning(h, "ignored") == 0 {
		t.Fatalf("no warning about the ignored row name: %+v", h.snapshot())
	}
	dt.SetRowNames([]string{"r", "r"})
	assertNames(t, "dup", dt.RowNames(), []string{"r", "r_1"})
	if warningsMentioning(h, "r_1") == 0 {
		t.Fatalf("no warning naming r_1: %+v", h.snapshot())
	}
	if dt.Err() != nil {
		t.Fatalf("Err = %v, want nil", dt.Err())
	}
}

func TestSampleKeepsColumnNames(t *testing.T) {
	assertNames(t, "sample", twoColTable().Sample(1, false).ColNames(), []string{"a", "b"})
}
