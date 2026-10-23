package insyra

import (
	"reflect"
	"strings"
	"sync"
	"testing"
)

var _ IDataTable = (*DataTable)(nil)

// ColRange takes both ends: Excel B:D, pandas loc["b":"d"]. Every bound is a
// real column in either spelling, so the one-past-the-end value that
// SliceCols accepts is an error here.
func TestColRangeIncludesBothEnds(t *testing.T) {
	dt := sliceFixture()
	check := func(what string, got *DataTable, cols []string) {
		t.Helper()
		if !reflect.DeepEqual(got.ColNames(), cols) {
			t.Errorf("%s has columns %v, want %v", what, got.ColNames(), cols)
		}
		if got.NumCols() > 0 && got.NumRows() != dt.NumRows() {
			t.Errorf("%s has %d rows, want %d", what, got.NumRows(), dt.NumRows())
		}
	}
	bcd := []string{"b", "c", "d"}
	check(`ColRange("B", "D")`, dt.ColRange("B", "D"), bcd)
	check(`ColRange(1, 3)`, dt.ColRange(1, 3), bcd)
	check(`ColRange("B", -1)`, dt.ColRange("B", -1), bcd)
	check(`ColRange(1, "D")`, dt.ColRange(1, "D"), bcd)
	check(`ColRange("B", "B")`, dt.ColRange("B", "B"), []string{"b"})
	check(`ColRange("C", nil)`, dt.ColRange("C", nil), []string{"c", "d"})
	check(`ColRange(nil, "B")`, dt.ColRange(nil, "B"), []string{"a", "b"})
	check(`ColRange(nil, nil)`, dt.ColRange(nil, nil), []string{"a", "b", "c", "d"})
	check(`ColRange(-2, nil)`, dt.ColRange(-2, nil), []string{"c", "d"})
	check(`ColRange(-3, -2)`, dt.ColRange(-3, -2), []string{"b", "c"})
	if e := dt.Err(); e != nil {
		t.Fatalf("valid ranges recorded %v", e)
	}
	// SliceCols("B", "B") is empty on purpose; ColRange("B", "B") is one column.
	if n := dt.SliceCols("B", "B").NumCols(); n != 0 {
		t.Errorf("SliceCols(B, B) has %d columns, want 0", n)
	}
	got := dt.ColRange("B", "D")
	if !reflect.DeepEqual(got.RowNames(), dt.RowNames()) || got.GetName() != "t" {
		t.Errorf("row names %q, table name %q", got.RowNames(), got.GetName())
	}
	if want := []any{"v", "w", "x", "y", "z"}; !reflect.DeepEqual(got.GetColByNumber(0).Data(), want) {
		t.Errorf("first column = %v, want %v", got.GetColByNumber(0).Data(), want)
	}
}

func TestColRangeRefusesWhatItCannotRead(t *testing.T) {
	for _, c := range []struct {
		from, to any
		want     string
	}{
		{"A", "Z", "column Z does not exist"},
		{"1", nil, "not an Excel-style"},
		{0, 4, "out of range"},
		{"A", 4, "out of range"},
		{-5, nil, "out of range"},
		{1.5, nil, "float64"},
		{nil, true, "bool"},
		{"C", "B", "after"},
		{2, 1, "after"},
		{-1, "B", "after"},
	} {
		dt := sliceFixture()
		got := dt.ColRange(c.from, c.to)
		if got == nil || got.NumCols() != 0 {
			t.Errorf("ColRange(%v, %v) returned %v, want an empty table", c.from, c.to, got)
		}
		e := dt.Err()
		if e == nil || e.FuncName != "ColRange" || !strings.Contains(e.Message, c.want) {
			t.Errorf("ColRange(%v, %v) recorded %v, want an error containing %q", c.from, c.to, e, c.want)
		}
	}
}

func TestColRangeOnEmptyTable(t *testing.T) {
	for _, c := range [][2]any{{nil, nil}, {"A", nil}, {0, 0}} {
		dt := NewDataTable()
		got := dt.ColRange(c[0], c[1])
		if got == nil || got.NumCols() != 0 {
			t.Errorf("ColRange(%v, %v) on an empty table = %v, want an empty table", c[0], c[1], got)
		}
		if e := dt.Err(); e == nil || e.FuncName != "ColRange" {
			t.Errorf("ColRange(%v, %v) on an empty table recorded %v, want an error", c[0], c[1], e)
		}
	}
}

func TestColRangePadsShortColumns(t *testing.T) {
	dt := NewDataTable(NewDataList(1, 2, 3), NewDataList("x"))
	dt.columns[1].data = []any{"x"}
	got := dt.ColRange(0, 1)
	if want := [][]any{{1, "x"}, {2, nil}, {3, nil}}; !reflect.DeepEqual(got.To2DSlice(), want) {
		t.Fatalf("ColRange of a short column = %v, want %v", got.To2DSlice(), want)
	}
	if len(dt.columns[1].data) != 1 {
		t.Errorf("source column grew to %d cells", len(dt.columns[1].data))
	}
}

func TestColRangeOwnsItsData(t *testing.T) {
	dt := sliceFixture()
	got := dt.ColRange("B", "C")
	got.GetColByNumber(0).Append("extra")
	got.UpdateElement(0, "B", "changed")
	if v := dt.GetElement(0, "B"); v != "v" {
		t.Errorf("source cell changed to %v", v)
	}
	if n := dt.GetColByNumber(1).Len(); n != 5 {
		t.Errorf("source column has %d cells, want 5", n)
	}
}

func TestColRangeConcurrent(t *testing.T) {
	dt := sliceFixture()
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				if got := dt.ColRange("B", "D"); got.NumCols() != 3 {
					t.Errorf("ColRange gave %d columns", got.NumCols())
				}
				dt.UpdateElement(0, "A", j)
			}
		}()
	}
	wg.Wait()
}
