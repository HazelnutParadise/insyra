package insyra

import (
	"math"
	"reflect"
	"testing"
)

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
	quietLogs(t)
	dt := NewDataTable(NewDataList(1, 2))
	noPanic(t, "GetElementByNumberIndex", func() {
		if v := dt.GetElementByNumberIndex(0, 5); v != nil {
			t.Fatalf("expected nil, got %v", v)
		}
	})
	if dt.Err() == nil {
		t.Fatal("expected Err")
	}
	if v := dt.GetElementByNumberIndex(1, -1); v != 2 {
		t.Fatalf("negative column index: expected 2, got %v", v)
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

func TestNotFoundFilterResultIsUsable(t *testing.T) {
	quietLogs(t)
	src := NewDataTable(NewDataList(1, 2).SetName("a"))
	noPanic(t, "empty filter result", func() {
		e := src.FilterColsByColNameEqualTo("zzz")
		e.GetRowIndexByName("x")
		e.SwapRowsByName("x", "y")
		e.FilterRowsByRowNameEqualTo("x")
	})
}

func TestFilterRowsJagged(t *testing.T) {
	quietLogs(t)
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

func TestAppendRowsByColIndexGrowsToTarget(t *testing.T) {
	quietLogs(t)
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
	bad := NewDataTable(NewDataList(1))
	bad.AppendRowsByColIndex(map[string]any{"1": 42})
	if bad.Err() == nil || bad.NumCols() != 1 {
		t.Fatalf("invalid key: expected Err and no new column, cols=%d err=%v", bad.NumCols(), bad.Err())
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
