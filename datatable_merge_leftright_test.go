package insyra

import (
	"reflect"
	"testing"
)

// The left and right join examples in Docs/DataTable.md had no test behind them.
// These pin the exact rows and their order, not just the row count, because a
// join that keeps the right number of rows can still attach the wrong ones.
func TestDataTable_Merge_Horizontal_LeftAndRight(t *testing.T) {
	newPair := func() (*DataTable, *DataTable) {
		dt1 := NewDataTable(
			NewDataList("A", "B", "C").SetName("ID"),
			NewDataList(1, 2, 3).SetName("Val1"),
		)
		dt2 := NewDataTable(
			NewDataList("B", "C", "D").SetName("ID"),
			NewDataList(10, 20, 30).SetName("Val2"),
		)
		return dt1, dt2
	}

	t.Run("left keeps every row of the first table", func(t *testing.T) {
		dt1, dt2 := newPair()
		res, err := dt1.Merge(dt2, MergeDirectionHorizontal, MergeModeLeft, "ID")
		if err != nil {
			t.Fatalf("Merge failed: %v", err)
		}
		wantID := []any{"A", "B", "C"}
		wantVal1 := []any{1, 2, 3}
		wantVal2 := []any{nil, 10, 20}
		if got := res.GetColByName("ID").Data(); !reflect.DeepEqual(got, wantID) {
			t.Errorf("ID: got %v, want %v", got, wantID)
		}
		if got := res.GetColByName("Val1").Data(); !reflect.DeepEqual(got, wantVal1) {
			t.Errorf("Val1: got %v, want %v", got, wantVal1)
		}
		if got := res.GetColByName("Val2").Data(); !reflect.DeepEqual(got, wantVal2) {
			t.Errorf("Val2: got %v, want %v", got, wantVal2)
		}
	})

	t.Run("right keeps every row of the second table", func(t *testing.T) {
		dt1, dt2 := newPair()
		res, err := dt1.Merge(dt2, MergeDirectionHorizontal, MergeModeRight, "ID")
		if err != nil {
			t.Fatalf("Merge failed: %v", err)
		}
		wantID := []any{"B", "C", "D"}
		wantVal1 := []any{2, 3, nil}
		wantVal2 := []any{10, 20, 30}
		if got := res.GetColByName("ID").Data(); !reflect.DeepEqual(got, wantID) {
			t.Errorf("ID: got %v, want %v", got, wantID)
		}
		if got := res.GetColByName("Val1").Data(); !reflect.DeepEqual(got, wantVal1) {
			t.Errorf("Val1: got %v, want %v", got, wantVal1)
		}
		if got := res.GetColByName("Val2").Data(); !reflect.DeepEqual(got, wantVal2) {
			t.Errorf("Val2: got %v, want %v", got, wantVal2)
		}
	})
}

// Joining on row names takes a different path through mergeHorizontal: there is
// no key column to carry the key value, and unnamed rows are handled separately.
func TestDataTable_Merge_Horizontal_LeftAndRight_ByRowName(t *testing.T) {
	newPair := func() (*DataTable, *DataTable) {
		dt1 := NewDataTable(NewDataList(1, 2, 3).SetName("Val1"))
		dt1.SetRowNames([]string{"A", "B", "C"})
		dt2 := NewDataTable(NewDataList(10, 20, 30).SetName("Val2"))
		dt2.SetRowNames([]string{"B", "C", "D"})
		return dt1, dt2
	}

	t.Run("left", func(t *testing.T) {
		dt1, dt2 := newPair()
		res, err := dt1.Merge(dt2, MergeDirectionHorizontal, MergeModeLeft)
		if err != nil {
			t.Fatalf("Merge failed: %v", err)
		}
		wantNames := []string{"A", "B", "C"}
		for i, want := range wantNames {
			if got, ok := res.GetRowNameByIndex(i); !ok || got != want {
				t.Errorf("row %d: got name %q, want %q", i, got, want)
			}
		}
		if got, want := res.GetColByName("Val1").Data(), []any{1, 2, 3}; !reflect.DeepEqual(got, want) {
			t.Errorf("Val1: got %v, want %v", got, want)
		}
		if got, want := res.GetColByName("Val2").Data(), []any{nil, 10, 20}; !reflect.DeepEqual(got, want) {
			t.Errorf("Val2: got %v, want %v", got, want)
		}
	})

	t.Run("right", func(t *testing.T) {
		dt1, dt2 := newPair()
		res, err := dt1.Merge(dt2, MergeDirectionHorizontal, MergeModeRight)
		if err != nil {
			t.Fatalf("Merge failed: %v", err)
		}
		wantNames := []string{"B", "C", "D"}
		for i, want := range wantNames {
			if got, ok := res.GetRowNameByIndex(i); !ok || got != want {
				t.Errorf("row %d: got name %q, want %q", i, got, want)
			}
		}
		if got, want := res.GetColByName("Val1").Data(), []any{2, 3, nil}; !reflect.DeepEqual(got, want) {
			t.Errorf("Val1: got %v, want %v", got, want)
		}
		if got, want := res.GetColByName("Val2").Data(), []any{10, 20, 30}; !reflect.DeepEqual(got, want) {
			t.Errorf("Val2: got %v, want %v", got, want)
		}
	})
}
