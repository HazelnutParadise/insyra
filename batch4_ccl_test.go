package insyra

import (
	"reflect"
	"testing"
)

// CCL-1 (range part): an Excel index past the last column is an error, not
// a silent column of nil.
func TestCCLOutOfRangeColumnIsError(t *testing.T) {
	dt := NewDataTable(NewDataList(1, 2, 3).SetName("price"))
	dt.AddColUsingCCL("r", "E + 1")
	if dt.Err() == nil {
		t.Fatal("expected Err() for a reference to column E on a one-column table")
	}
	if dt.NumCols() != 1 {
		t.Fatalf("failed expression must not add a column, got %d columns", dt.NumCols())
	}
}

// CCL-2: `@` as a value hands each row its own slice.
func TestCCLCurrentRowIsCopied(t *testing.T) {
	dt := NewDataTable(NewDataList(10, 20, 30), NewDataList(1, 2, 3))
	dt.AddColUsingCCL("r", "@")
	got := dt.GetColByName("r").Data()
	want := []any{[]any{10, 1}, []any{20, 2}, []any{30, 3}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("@ column = %v, want %v", got, want)
	}
	dt2 := NewDataTable(NewDataList(10, 20, 30), NewDataList(1, 2, 3))
	dt2.ExecuteCCL("NEW('r') = @")
	if got := dt2.GetColByName("r").Data(); !reflect.DeepEqual(got, want) {
		t.Fatalf("ExecuteCCL @ column = %v, want %v", got, want)
	}
}
