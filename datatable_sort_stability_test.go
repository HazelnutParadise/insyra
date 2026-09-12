package insyra

import (
	"reflect"
	"testing"
)

// Docs/DataTable.md promises SortBy "uses stable sort to maintain relative order
// of equal elements". Nothing tested it, and an unstable sort passes every
// existing sort test, because they all have distinct keys or a tie-break column.
func TestDataTable_SortBy_StableOnTies(t *testing.T) {
	// Every row of Key repeats, and Seq records where the row started. A stable
	// sort leaves Seq ascending inside each group of equal keys.
	dt := NewDataTable(
		NewDataList(2, 1, 2, 1, 2, 1, 2, 1).SetName("Key"),
		NewDataList(0, 1, 2, 3, 4, 5, 6, 7).SetName("Seq"),
	)

	dt.SortBy(DataTableSortConfig{ColumnName: "Key"})

	wantKey := []any{1, 1, 1, 1, 2, 2, 2, 2}
	wantSeq := []any{1, 3, 5, 7, 0, 2, 4, 6}
	if got := dt.GetColByName("Key").Data(); !reflect.DeepEqual(got, wantKey) {
		t.Errorf("Key: got %v, want %v", got, wantKey)
	}
	if got := dt.GetColByName("Seq").Data(); !reflect.DeepEqual(got, wantSeq) {
		t.Errorf("Seq: got %v, want %v — ties did not keep their original order", got, wantSeq)
	}
}

// The same promise for a descending sort: reversing the order of the groups
// must not reverse the rows inside a group.
func TestDataTable_SortBy_StableOnTies_Descending(t *testing.T) {
	dt := NewDataTable(
		NewDataList(1, 2, 1, 2, 1, 2).SetName("Key"),
		NewDataList(0, 1, 2, 3, 4, 5).SetName("Seq"),
	)

	dt.SortBy(DataTableSortConfig{ColumnName: "Key", Descending: true})

	wantKey := []any{2, 2, 2, 1, 1, 1}
	wantSeq := []any{1, 3, 5, 0, 2, 4}
	if got := dt.GetColByName("Key").Data(); !reflect.DeepEqual(got, wantKey) {
		t.Errorf("Key: got %v, want %v", got, wantKey)
	}
	if got := dt.GetColByName("Seq").Data(); !reflect.DeepEqual(got, wantSeq) {
		t.Errorf("Seq: got %v, want %v — ties did not keep their original order", got, wantSeq)
	}
}

// Multi-level: rows equal on every configured column keep their original order.
func TestDataTable_SortBy_StableOnTies_MultiLevel(t *testing.T) {
	dt := NewDataTable(
		NewDataList("x", "y", "x", "y", "x", "y").SetName("A"),
		NewDataList(1, 1, 1, 1, 1, 1).SetName("B"),
		NewDataList(0, 1, 2, 3, 4, 5).SetName("Seq"),
	)

	dt.SortBy(
		DataTableSortConfig{ColumnName: "A"},
		DataTableSortConfig{ColumnName: "B", Descending: true},
	)

	wantA := []any{"x", "x", "x", "y", "y", "y"}
	wantSeq := []any{0, 2, 4, 1, 3, 5}
	if got := dt.GetColByName("A").Data(); !reflect.DeepEqual(got, wantA) {
		t.Errorf("A: got %v, want %v", got, wantA)
	}
	if got := dt.GetColByName("Seq").Data(); !reflect.DeepEqual(got, wantSeq) {
		t.Errorf("Seq: got %v, want %v — ties did not keep their original order", got, wantSeq)
	}
}

// Row names travel with their rows through a sort with ties.
func TestDataTable_SortBy_StableOnTies_RowNames(t *testing.T) {
	dt := NewDataTable(
		NewDataList(2, 1, 2, 1).SetName("Key"),
		NewDataList(0, 1, 2, 3).SetName("Seq"),
	)
	dt.SetRowNames([]string{"r0", "r1", "r2", "r3"})

	dt.SortBy(DataTableSortConfig{ColumnName: "Key"})

	wantNames := []string{"r1", "r3", "r0", "r2"}
	for i, want := range wantNames {
		if got, ok := dt.GetRowNameByIndex(i); !ok || got != want {
			t.Errorf("row %d: got name %q, want %q", i, got, want)
		}
	}
}
