package insyra

import (
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/HazelnutParadise/insyra/internal/ccl"
)

func init() {
	SetDefaultConfig()
	Config.SetLogLevel(LogLevelFatal)
}

func noPanic(t *testing.T, name string, f func()) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("%s panicked: %v", name, r)
		}
	}()
	f()
}

func TestBadIndexSetsErrNotPanic(t *testing.T) {
	dt := NewDataTable(NewDataList(1, 2))
	noPanic(t, "GetElementByNumberIndex", func() {
		if v := dt.GetElementByNumberIndex(0, 5); v != nil {
			t.Fatalf("expected nil, got %v", v)
		}
	})
	if dt.Err() == nil {
		t.Fatal("expected Err")
	}
	dt2 := NewDataTable(NewDataList(1, 2))
	noPanic(t, "SetRowToColNames", func() { dt2.SetRowToColNames(99) })
	if dt2.Err() == nil || dt2.NumRows() != 2 {
		t.Fatalf("expected Err and unchanged table, rows=%d err=%v", dt2.NumRows(), dt2.Err())
	}
	dt3 := NewDataTable(NewDataList(1, 2))
	noPanic(t, "SetColToRowNames", func() { dt3.SetColToRowNames("ZZ") })
	if dt3.Err() == nil || dt3.NumCols() != 1 {
		t.Fatalf("expected Err and unchanged table, cols=%d err=%v", dt3.NumCols(), dt3.Err())
	}
}

func TestDataReturnsCopies(t *testing.T) {
	dt := NewDataTable(NewDataList(1, 2).SetName("a"))
	m := dt.Data()
	m["a"][0] = 99
	if got := dt.GetElement(0, "A"); got != 1 {
		t.Fatalf("Data aliased internal storage: %v", got)
	}
	m2 := dt.ToMap()
	m2["a"][1] = 99
	if got := dt.GetElement(1, "A"); got != 2 {
		t.Fatalf("ToMap aliased internal storage: %v", got)
	}
}

func TestFilterResultsDoNotAlias(t *testing.T) {
	src := NewDataTable(NewDataList(1, 2).SetName("a"), NewDataList(3, 4).SetName("b"))
	src.SetRowNames([]string{"r0", "r1"})
	filters := map[string]func() *DataTable{
		"ByColNameEqualTo":                  func() *DataTable { return src.FilterColsByColNameEqualTo("a") },
		"ByColNameContains":                 func() *DataTable { return src.FilterColsByColNameContains("a") },
		"ByColIndexEqualTo":                 func() *DataTable { return src.FilterColsByColIndexEqualTo("A") },
		"ByColIndexGreaterThanOrEqualTo":    func() *DataTable { return src.FilterColsByColIndexGreaterThanOrEqualTo("A") },
		"ByColIndexLessThanOrEqualTo":       func() *DataTable { return src.FilterColsByColIndexLessThanOrEqualTo("A") },
		"FilterCols":                        func() *DataTable { return src.FilterCols(func(int, string, any) bool { return true }) },
	}
	for name, f := range filters {
		out := f()
		out.UpdateElement(0, "A", 99)
		out.SetRowNameByIndex(0, "changed")
		if got := src.GetElement(0, "A"); got != 1 {
			t.Fatalf("%s: source cell changed to %v", name, got)
		}
		if names := src.RowNames(); names[0] != "r0" {
			t.Fatalf("%s: source row names changed to %v", name, names)
		}
	}
}

func TestNotFoundFilterResultIsUsable(t *testing.T) {
	src := NewDataTable(NewDataList(1, 2).SetName("a"))
	noPanic(t, "empty filter result", func() {
		e := src.FilterColsByColNameEqualTo("zzz")
		e.GetRowIndexByName("x")
		e.SwapRowsByName("x", "y")
		e.FilterRowsByRowNameEqualTo("x")
	})
}

func TestFilterRowsJagged(t *testing.T) {
	j := NewDataTable()
	j.AppendCols(NewDataList(1, 2, 3))
	j.columns = append(j.columns, NewDataList(1))
	var out *DataTable
	noPanic(t, "FilterRows jagged", func() {
		out = j.FilterRows(func(string, string, any) bool { return true })
	})
	if out.NumRows() != 3 || out.GetElement(2, "B") != nil {
		t.Fatalf("expected 3 rows with nil padding, got rows=%d B2=%v", out.NumRows(), out.GetElement(2, "B"))
	}
	noPanic(t, "FilterCols jagged", func() { j.FilterCols(func(int, string, any) bool { return true }) })
}

func TestDropRowsByIndexNormalises(t *testing.T) {
	dt := NewDataTable(NewDataList(0, 1, 2, 3))
	dt.DropRowsByIndex(-1, 0)
	if !reflect.DeepEqual(dt.GetCol("A").Data(), []any{1, 2}) {
		t.Fatalf("DropRowsByIndex(-1,0): %v", dt.GetCol("A").Data())
	}
	dt2 := NewDataTable(NewDataList(0, 1, 2, 3))
	dt2.DropRowsByIndex(1, 1)
	if !reflect.DeepEqual(dt2.GetCol("A").Data(), []any{0, 2, 3}) {
		t.Fatalf("DropRowsByIndex(1,1): %v", dt2.GetCol("A").Data())
	}
}

func TestTransposeKeepsAllRowNames(t *testing.T) {
	dt := NewDataTable(NewDataList(1, 2, 3).SetName("x"), NewDataList(4, 5, 6).SetName("y"))
	dt.SetRowNames([]string{"r0", "r1", "r2"})
	dt.Transpose()
	if !reflect.DeepEqual(dt.ColNames(), []string{"r0", "r1", "r2"}) {
		t.Fatalf("col names after transpose: %v", dt.ColNames())
	}
	if !reflect.DeepEqual(dt.RowNames(), []string{"x", "y"}) {
		t.Fatalf("row names after transpose: %v", dt.RowNames())
	}
}

func TestChangeRowNameDoesNotStealName(t *testing.T) {
	dt := NewDataTable(NewDataList(1, 2))
	dt.SetRowNames([]string{"a", "b"})
	dt.ChangeRowName("b", "a")
	if names := dt.RowNames(); names[0] != "a" || names[1] != "a_1" {
		t.Fatalf("got %v", names)
	}
}

func TestAppendRowsByColIndexGrowsToTarget(t *testing.T) {
	dt := NewDataTable(NewDataList(1), NewDataList(2))
	dt.AppendRowsByColIndex(map[string]any{"Z": 42})
	if dt.NumCols() != 26 {
		t.Fatalf("expected 26 cols, got %d", dt.NumCols())
	}
	if got := dt.GetElement(-1, "Z"); got != 42 {
		t.Fatalf("expected 42 in Z, got %v", got)
	}
	if got := dt.GetElement(-1, "A"); got != nil {
		t.Fatalf("expected nil padding in A, got %v", got)
	}
}

func TestNumericMembershipIncludesInt64(t *testing.T) {
	dt := NewDataTable(NewDataList(int64(1), int64(2)).SetName("n"), NewDataList("s", "t").SetName("s"))
	dt.DropColsContainNumber()
	if !reflect.DeepEqual(dt.ColNames(), []string{"s"}) {
		t.Fatalf("expected only s, got %v", dt.ColNames())
	}
	dt2 := NewDataTable(NewDataList(int64(1), "x"), NewDataList("a", "b"))
	dt2.DropRowsContainNumber()
	if dt2.NumRows() != 1 {
		t.Fatalf("expected 1 row, got %d", dt2.NumRows())
	}
}

func TestMeanDenominatorCountsNumericOnly(t *testing.T) {
	dt := NewDataTable(NewDataList(2, "x"), NewDataList(4, nil))
	if got := dt.Mean().(float64); got != 3 {
		t.Fatalf("expected 3, got %v", got)
	}
	empty := NewDataTable(NewDataList("a"))
	if got := empty.Mean().(float64); !math.IsNaN(got) {
		t.Fatalf("expected NaN, got %v", got)
	}
}

func TestToCSVTimeRoundTrip(t *testing.T) {
	when := time.Date(2024, 1, 2, 3, 4, 5, 600, time.UTC)
	dt := NewDataTable(NewDataList(when, when.Add(time.Hour)).SetName("t"))
	p := filepath.Join(t.TempDir(), "t.csv")
	if err := dt.ToCSV(p, false, true, false); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(p)
	if !strings.Contains(string(raw), "2024-01-02T03:04:05") {
		t.Fatalf("expected RFC3339 in file, got %q", raw)
	}
	back, err := ReadCSV_FileWithOptions(p, CSVReadOptions{FirstRowToColNames: true, RawStrings: true})
	if err != nil {
		t.Fatal(err)
	}
	col := back.GetColByName("t").ParseDates()
	for i, v := range col.Data() {
		got, ok := v.(time.Time)
		if !ok || !got.Equal(when.Add(time.Duration(i)*time.Hour)) {
			t.Fatalf("row %d: got %v", i, v)
		}
	}
}

func TestCCLPanicReturnsReceiver(t *testing.T) {
	ccl.RegisterFunction("ZZPANIC", func(args ...any) (any, error) { panic("boom") })
	dt := NewDataTable(NewDataList(1, 2).SetName("a"))
	var out *DataTable
	noPanic(t, "AddColUsingCCL", func() { out = dt.AddColUsingCCL("b", "ZZPANIC(A)") })
	if out == nil {
		t.Fatal("AddColUsingCCL returned nil after recovered panic")
	}
	if out.Err() == nil {
		t.Fatal("expected Err")
	}
	noPanic(t, "ExecuteCCL", func() { out = dt.ExecuteCCL("NEW('c') = ZZPANIC(A)") })
	if out == nil {
		t.Fatal("ExecuteCCL returned nil after recovered panic")
	}
	noPanic(t, "EditColByIndexUsingCCL", func() { out = dt.EditColByIndexUsingCCL("A", "ZZPANIC(A)") })
	if out == nil {
		t.Fatal("EditColByIndexUsingCCL returned nil")
	}
	noPanic(t, "EditColByNameUsingCCL", func() { out = dt.EditColByNameUsingCCL("a", "ZZPANIC(A)") })
	if out == nil {
		t.Fatal("EditColByNameUsingCCL returned nil")
	}
}
