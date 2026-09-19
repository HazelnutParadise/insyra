package isr

import (
	"reflect"
	"testing"
)

// DT.From's and DT.Of's doc comments both list map[int]any as a supported
// input, and it never worked: the key was stringified to "0" where
// AppendRowsByColIndex wants an Excel-style index, so every key was rejected
// and the result was an empty table carrying "Invalid column index '0'".
// The sibling Row path gets this right through numberToColIndex.
func TestDT_From_MapIntKey(t *testing.T) {
	quietTest(t)

	table := DT.From(map[int]any{0: 1, 1: "x", 2: true})
	if err := table.Err(); err != nil {
		t.Fatalf("building from a map[int]any: %v", err)
	}

	rows, cols := table.Size()
	if rows != 1 || cols != 3 {
		t.Fatalf("size: got %dx%d, want 1x3", rows, cols)
	}
	// Key 0 is column A, key 1 is column B, key 2 is column C.
	for i, want := range []any{1, "x", true} {
		got := table.GetColByNumber(i).Get(0)
		if got != want {
			t.Errorf("column %d: got %v, want %v", i, got, want)
		}
	}
}

// The Row path already did this, and the two must agree.
func TestDT_From_MapIntKeyMatchesRow(t *testing.T) {
	quietTest(t)

	fromMap := DT.From(map[int]any{0: 1, 1: 2})
	fromRow := DT.From(Row{0: 1, 1: 2})

	mapRows, mapCols := fromMap.Size()
	rowRows, rowCols := fromRow.Size()
	if mapRows != rowRows || mapCols != rowCols {
		t.Fatalf("map gave %dx%d, Row gave %dx%d", mapRows, mapCols, rowRows, rowCols)
	}
	for i := 0; i < mapCols; i++ {
		a := fromMap.GetColByNumber(i).Data()
		b := fromRow.GetColByNumber(i).Data()
		if !reflect.DeepEqual(a, b) {
			t.Errorf("column %d: map gave %v, Row gave %v", i, a, b)
		}
	}
}

// A negative key has no column to land in. numberToColIndex returns "" for one,
// which AppendRowsByColIndex refuses, so the failure is recorded rather than
// silently dropping the value.
func TestDT_From_MapIntKeyNegative(t *testing.T) {
	quietTest(t)

	table := DT.From(map[int]any{-1: 1})
	if table.Err() == nil {
		t.Error("a negative key recorded no error")
	}
}
