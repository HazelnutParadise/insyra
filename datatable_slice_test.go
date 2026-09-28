package insyra

import (
	"reflect"
	"strings"
	"testing"
)

func sliceFixture() *DataTable {
	dt := NewDataTable(
		NewDataList(1, 2, 3, 4, 5).SetName("a"),
		NewDataList("v", "w", "x", "y", "z").SetName("b"),
		NewDataList(1.5, 2.5, 3.5, 4.5, 5.5).SetName("c"),
		NewDataList(true, false, true, false, true).SetName("d"),
	)
	dt.SetName("t")
	dt.SetRowNameByIndex(1, "one")
	dt.SetRowNameByIndex(2, "two")
	dt.SetRowNameByIndex(4, "four")
	return dt
}

func TestSliceRows(t *testing.T) {
	dt := sliceFixture()
	got := dt.SliceRows(1, 3)
	if e := dt.Err(); e != nil {
		t.Fatalf("SliceRows(1, 3) recorded %v", e)
	}
	want := [][]any{{2, "w", 2.5, false}, {3, "x", 3.5, true}}
	if !reflect.DeepEqual(got.To2DSlice(), want) {
		t.Fatalf("SliceRows(1, 3) = %v, want %v", got.To2DSlice(), want)
	}
	if !reflect.DeepEqual(got.RowNames(), []string{"one", "two"}) {
		t.Errorf("row names = %q, want [one two]", got.RowNames())
	}
	if !reflect.DeepEqual(got.ColNames(), []string{"a", "b", "c", "d"}) || got.GetName() != "t" {
		t.Errorf("column names %v, table name %q", got.ColNames(), got.GetName())
	}
	if all := dt.SliceRows(0, 5); !reflect.DeepEqual(all.To2DSlice(), dt.To2DSlice()) {
		t.Errorf("SliceRows(0, 5) = %v, want the whole table", all.To2DSlice())
	}
	empty := dt.SliceRows(2, 2)
	if empty.NumRows() != 0 || !reflect.DeepEqual(empty.ColNames(), []string{"a", "b", "c", "d"}) {
		t.Errorf("SliceRows(2, 2) is %d rows with columns %v, want 0 rows and every column", empty.NumRows(), empty.ColNames())
	}
	if e := dt.Err(); e != nil {
		t.Fatalf("in-range slices recorded %v", e)
	}
}

func TestSliceCols(t *testing.T) {
	dt := sliceFixture()
	got := dt.SliceCols(1, 3)
	if e := dt.Err(); e != nil {
		t.Fatalf("SliceCols(1, 3) recorded %v", e)
	}
	if !reflect.DeepEqual(got.ColNames(), []string{"b", "c"}) {
		t.Fatalf("SliceCols(1, 3) has columns %v, want [b c]", got.ColNames())
	}
	if want := []any{"v", "w", "x", "y", "z"}; !reflect.DeepEqual(got.GetColByNumber(0).Data(), want) {
		t.Errorf("first column = %v, want %v", got.GetColByNumber(0).Data(), want)
	}
	if !reflect.DeepEqual(got.RowNames(), dt.RowNames()) || got.GetName() != "t" {
		t.Errorf("row names %q, table name %q", got.RowNames(), got.GetName())
	}
	if all := dt.SliceCols(0, 4); !reflect.DeepEqual(all.To2DSlice(), dt.To2DSlice()) {
		t.Errorf("SliceCols(0, 4) = %v, want the whole table", all.To2DSlice())
	}
	if none := dt.SliceCols(4, 4); none.NumCols() != 0 {
		t.Errorf("SliceCols(4, 4) has %d columns, want 0", none.NumCols())
	}
	if e := dt.Err(); e != nil {
		t.Fatalf("in-range slices recorded %v", e)
	}
}

// A bound outside 0 <= from <= to <= length is an error, as in a Go slice
// expression, reported on the table rather than by panicking.
func TestSliceOutOfRange(t *testing.T) {
	for _, c := range []struct {
		name     string
		from, to int
		rows     bool
	}{
		{"SliceRows", -1, 2, true},
		{"SliceRows", 3, 2, true},
		{"SliceRows", 0, 6, true},
		{"SliceCols", -1, 1, false},
		{"SliceCols", 2, 1, false},
		{"SliceCols", 0, 5, false},
	} {
		dt := sliceFixture()
		var got *DataTable
		if c.rows {
			got = dt.SliceRows(c.from, c.to)
		} else {
			got = dt.SliceCols(c.from, c.to)
		}
		if got == nil {
			t.Fatalf("%s(%d, %d) returned nil", c.name, c.from, c.to)
		}
		if got.NumCols() != 0 || got.NumRows() != 0 {
			t.Errorf("%s(%d, %d) returned %d x %d, want an empty table", c.name, c.from, c.to, got.NumRows(), got.NumCols())
		}
		e := dt.Err()
		if e == nil || e.FuncName != c.name || !strings.Contains(e.Message, "out of range") {
			t.Errorf("%s(%d, %d) recorded %v, want an out-of-range error", c.name, c.from, c.to, e)
		}
	}
}

func TestSliceResultOwnsItsData(t *testing.T) {
	dt := sliceFixture()
	rows := dt.SliceRows(0, 2)
	rows.UpdateElement(0, "A", 100)
	cols := dt.SliceCols(0, 1)
	cols.UpdateElement(1, "A", 200)
	if v := dt.GetElementByNumberIndex(0, 0); v != 1 {
		t.Errorf("editing SliceRows' result changed the source to %v", v)
	}
	if v := dt.GetElementByNumberIndex(1, 0); v != 2 {
		t.Errorf("editing SliceCols' result changed the source to %v", v)
	}
}

// The ten deprecated index filters return what the slice they stand for
// returns, for every threshold inside the table.
func TestDeprecatedIndexFiltersMatchSlices(t *testing.T) {
	dt := sliceFixture()
	same := func(what string, a, b *DataTable) {
		t.Helper()
		if !reflect.DeepEqual(a.To2DSlice(), b.To2DSlice()) || !reflect.DeepEqual(a.ColNames(), b.ColNames()) ||
			!reflect.DeepEqual(a.RowNames(), b.RowNames()) {
			t.Errorf("%s: deprecated gave %v %v %q, slice gave %v %v %q", what,
				a.To2DSlice(), a.ColNames(), a.RowNames(), b.To2DSlice(), b.ColNames(), b.RowNames())
		}
	}
	nCols := dt.NumCols()
	for i := 0; i < nCols; i++ {
		letter, _ := CalcColIndex(i)
		if i+1 < nCols {
			same("ColIndexGreaterThan "+letter, dt.FilterColsByColIndexGreaterThan(letter), dt.SliceCols(i+1, nCols))
		}
		same("ColIndexGreaterThanOrEqualTo "+letter, dt.FilterColsByColIndexGreaterThanOrEqualTo(letter), dt.SliceCols(i, nCols))
		same("ColIndexEqualTo "+letter, dt.FilterColsByColIndexEqualTo(letter), dt.SliceCols(i, i+1))
		if i > 0 {
			same("ColIndexLessThan "+letter, dt.FilterColsByColIndexLessThan(letter), dt.SliceCols(0, i))
		}
		same("ColIndexLessThanOrEqualTo "+letter, dt.FilterColsByColIndexLessThanOrEqualTo(letter), dt.SliceCols(0, i+1))
	}
	nRows := dt.NumRows()
	for k := 0; k < nRows; k++ {
		same("RowIndexGreaterThan", dt.FilterRowsByRowIndexGreaterThan(k), dt.SliceRows(k+1, nRows))
		same("RowIndexGreaterThanOrEqualTo", dt.FilterRowsByRowIndexGreaterThanOrEqualTo(k), dt.SliceRows(k, nRows))
		same("RowIndexEqualTo", dt.FilterRowsByRowIndexEqualTo(k), dt.SliceRows(k, k+1))
		same("RowIndexLessThan", dt.FilterRowsByRowIndexLessThan(k), dt.SliceRows(0, k))
		same("RowIndexLessThanOrEqualTo", dt.FilterRowsByRowIndexLessThanOrEqualTo(k), dt.SliceRows(0, k+1))
	}
	if e := dt.Err(); e != nil {
		t.Fatalf("recorded %v", e)
	}
}

// A letter past the last column meant "every column" to the less-than
// filters, but they sliced past the end and panicked.
func TestDeprecatedColIndexFiltersDoNotPanicPastTheEnd(t *testing.T) {
	dt := sliceFixture()
	for name, got := range map[string]*DataTable{
		"LessThan":          dt.FilterColsByColIndexLessThan("Z"),
		"LessThanOrEqualTo": dt.FilterColsByColIndexLessThanOrEqualTo("Z"),
	} {
		if !reflect.DeepEqual(got.To2DSlice(), dt.To2DSlice()) {
			t.Errorf("FilterColsByColIndex%s(\"Z\") = %v, want every column", name, got.To2DSlice())
		}
	}
}

func TestHeadersAreColNames(t *testing.T) {
	dt := sliceFixture()
	if !reflect.DeepEqual(dt.Headers(), dt.ColNames()) {
		t.Fatalf("Headers() = %v, ColNames() = %v", dt.Headers(), dt.ColNames())
	}
	a, b := sliceFixture(), sliceFixture()
	a.SetHeaders([]string{"x", "y"})
	b.SetColNames([]string{"x", "y"})
	if !reflect.DeepEqual(a.ColNames(), b.ColNames()) {
		t.Fatalf("SetHeaders gave %v, SetColNames gave %v", a.ColNames(), b.ColNames())
	}
}
