package insyra

import (
	"reflect"
	"testing"
)

// A predicate that drops a column cannot shift the columns the filter reads:
// the filter works from the columns as the call found them.
func TestFilterRowsWhereSurvivesAPredicateDroppingAColumn(t *testing.T) {
	dt := NewDataTable(NewDataList(1).SetName("a"), NewDataList(2).SetName("b"))
	got := dt.FilterRowsWhere(func(row *DataList) bool {
		dt.DropColsByIndex("A")
		return true
	})
	if want := [][]any{{1, 2}}; !reflect.DeepEqual(got.To2DSlice(), want) {
		t.Fatalf("FilterRowsWhere = %v, want %v", got.To2DSlice(), want)
	}
}

func TestFilterSurvivesAPredicateDroppingAColumn(t *testing.T) {
	dt := NewDataTable(NewDataList(1).SetName("a"), NewDataList(2).SetName("b"))
	dropped := false
	got := dt.Filter(func(int, string, any) bool {
		if !dropped {
			dropped = true
			dt.DropColsByIndex("A")
		}
		return true
	})
	if want := [][]any{{1, 2}}; !reflect.DeepEqual(got.To2DSlice(), want) {
		t.Fatalf("Filter = %v, want %v", got.To2DSlice(), want)
	}
}

// A table's columns may differ in length; SliceCols keeps every row and pads
// a short column with nil.
func TestSliceColsKeepsEveryRowOfARaggedTable(t *testing.T) {
	dt := NewDataTable(NewDataList(1, 2, 3), NewDataList(4, 5, 6))
	dt.UpdateColByNumber(1, NewDataList(9))
	if dt.NumRows() != 3 {
		t.Fatalf("setup: NumRows = %d, want 3", dt.NumRows())
	}
	got := dt.SliceCols(1, nil)
	if got.NumRows() != 3 {
		t.Fatalf("SliceCols(1, nil) has %d rows, want 3", got.NumRows())
	}
	if want := []any{9, nil, nil}; !reflect.DeepEqual(got.GetColByNumber(0).Data(), want) {
		t.Errorf("column = %v, want %v", got.GetColByNumber(0).Data(), want)
	}
}

// The largest letter ColIndexToNumber reads is the largest int; adding one to it
// must not wrap around into a negative slice bound.
func TestFilterColsByColIndexLessThanOrEqualToTheLargestLetter(t *testing.T) {
	letters, ok := ColNumberToIndex(int(^uint(0) >> 1))
	if !ok {
		t.Skip("ColNumberToIndex cannot spell the largest int")
	}
	dt := NewDataTable(NewDataList(1), NewDataList(2))
	got := dt.FilterColsByColIndexLessThanOrEqualTo(letters) //nolint:staticcheck
	if got.NumCols() != 2 {
		t.Fatalf("NumCols = %d, want 2", got.NumCols())
	}
}
