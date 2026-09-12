package insyra

import (
	"reflect"
	"testing"
)

// T-15 of #228: NewDataTable(NewDataList(...)) leaves every column unnamed, and
// mergeVertical refused two such tables as having "duplicate column names" —
// the empty name counted as a duplicate of itself. So the shape a caller gets
// from the plainest constructor could not be merged with another of its kind.
func TestDataTable_MergeVertical_UnnamedColumns(t *testing.T) {
	dt1 := NewDataTable(
		NewDataList(1, 2),
		NewDataList("a", "b"),
	)
	dt2 := NewDataTable(
		NewDataList(3, 4),
		NewDataList("c", "d"),
	)

	res, err := dt1.Merge(dt2, MergeDirectionVertical, MergeModeOuter)
	if err != nil {
		t.Fatalf("merging two unnamed tables: %v", err)
	}

	rows, cols := res.Size()
	if rows != 4 || cols != 2 {
		t.Fatalf("size: got %dx%d, want 4x2", rows, cols)
	}
	// Unnamed columns line up by position, so column 0 is the numbers and
	// column 1 the letters — not everything piled into the first column.
	if got, want := res.GetColByNumber(0).Data(), []any{1, 2, 3, 4}; !reflect.DeepEqual(got, want) {
		t.Errorf("column 0: got %v, want %v", got, want)
	}
	if got, want := res.GetColByNumber(1).Data(), []any{"a", "b", "c", "d"}; !reflect.DeepEqual(got, want) {
		t.Errorf("column 1: got %v, want %v", got, want)
	}
}

// Named columns still line up by name whatever order they are in.
func TestDataTable_MergeVertical_NamedColumnsStillMatchByName(t *testing.T) {
	dt1 := NewDataTable(
		NewDataList(1, 2).SetName("n"),
		NewDataList("a", "b").SetName("s"),
	)
	dt2 := NewDataTable(
		NewDataList("c", "d").SetName("s"),
		NewDataList(3, 4).SetName("n"),
	)

	res, err := dt1.Merge(dt2, MergeDirectionVertical, MergeModeOuter)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	if got, want := res.GetColByName("n").Data(), []any{1, 2, 3, 4}; !reflect.DeepEqual(got, want) {
		t.Errorf("n: got %v, want %v", got, want)
	}
	if got, want := res.GetColByName("s").Data(), []any{"a", "b", "c", "d"}; !reflect.DeepEqual(got, want) {
		t.Errorf("s: got %v, want %v", got, want)
	}
}

// A table that mixes named and unnamed columns: the named ones match by name,
// the unnamed ones by their position among the unnamed.
func TestDataTable_MergeVertical_MixedNaming(t *testing.T) {
	dt1 := NewDataTable(
		NewDataList(1, 2).SetName("n"),
		NewDataList("a", "b"),
	)
	dt2 := NewDataTable(
		NewDataList(3, 4).SetName("n"),
		NewDataList("c", "d"),
	)

	res, err := dt1.Merge(dt2, MergeDirectionVertical, MergeModeOuter)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	if got, want := res.GetColByName("n").Data(), []any{1, 2, 3, 4}; !reflect.DeepEqual(got, want) {
		t.Errorf("n: got %v, want %v", got, want)
	}
	unnamed := res.GetColByNumber(1)
	if got, want := unnamed.Data(), []any{"a", "b", "c", "d"}; !reflect.DeepEqual(got, want) {
		t.Errorf("the unnamed column: got %v, want %v", got, want)
	}
}

// A table cannot actually hold two columns with the same name: both the
// constructor and SetColNameByNumber rename the second to "n_1". So the
// duplicate check in mergeVertical only ever fired on the empty name, which was
// the bug. This pins the renaming, so that if it ever stops the merge does not
// start silently dropping a column.
func TestDataTable_DuplicateColumnNamesAreRenamed(t *testing.T) {
	dt := NewDataTable(
		NewDataList(1).SetName("n"),
		NewDataList(2).SetName("n"),
	)
	if got, want := dt.ColNames(), []string{"n", "n_1"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("column names: got %v, want %v", got, want)
	}

	// And two such tables merge, because the names are distinct by then.
	res, err := dt.Merge(dt, MergeDirectionVertical, MergeModeOuter)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	if got, want := res.GetColByName("n").Data(), []any{1, 1}; !reflect.DeepEqual(got, want) {
		t.Errorf("n: got %v, want %v", got, want)
	}
	if got, want := res.GetColByName("n_1").Data(), []any{2, 2}; !reflect.DeepEqual(got, want) {
		t.Errorf("n_1: got %v, want %v", got, want)
	}
}
