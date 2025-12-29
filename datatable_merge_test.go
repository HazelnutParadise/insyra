package insyra

import (
	"testing"
)

func TestDataTable_Merge_Horizontal(t *testing.T) {
	dt1 := NewDataTable(
		NewDataList(1, 2, 3).SetName("ID"),
		NewDataList("A", "B", "C").SetName("Name"),
	)
	dt2 := NewDataTable(
		NewDataList(2, 3, 4).SetName("ID"),
		NewDataList(20, 30, 40).SetName("Age"),
	)

	// Inner Join
	res, err := dt1.Merge(dt2, MergeDirectionHorizontal, MergeModeInner, "ID")
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}
	if res.GetColByNumber(0).Len() != 2 {
		t.Errorf("Expected 2 rows, got %d", res.GetColByNumber(0).Len())
	}

	// Outer Join
	res, err = dt1.Merge(dt2, MergeDirectionHorizontal, MergeModeOuter, "ID")
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}
	if res.GetColByNumber(0).Len() != 4 {
		t.Errorf("Expected 4 rows, got %d", res.GetColByNumber(0).Len())
	}
}

func TestDataTable_Merge_Vertical_RowNames(t *testing.T) {
	dt1 := NewDataTable(
		NewDataList(1, 2).SetName("ID"),
		NewDataList(10, 20).SetName("Val1"),
	)
	dt1.SetRowNames([]string{"R1", "R2"})

	dt2 := NewDataTable(
		NewDataList(1, 3).SetName("ID"),
		NewDataList(100, 300).SetName("Val2"),
	)
	dt2.SetRowNames([]string{"R1", "R3"})

	// Vertical merge
	res, err := dt1.Merge(dt2, MergeDirectionVertical, MergeModeOuter)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	// Expected rows: 4
	// Row names: R1, R2, R1_1, R3
	expectedNames := []string{"R1", "R2", "R1_1", "R3"}
	for i, expected := range expectedNames {
		if name, ok := res.GetRowNameByIndex(i); !ok || name != expected {
			t.Errorf("At index %d: expected row name %s, got %s", i, expected, name)
		}
	}

	// Check data
	// ID column
	idCol := res.GetColByName("ID")
	if idCol == nil {
		t.Fatal("ID column not found")
	}
	expectedID := []any{1, 2, 1, 3}
	for i, v := range expectedID {
		if idCol.Get(i) != v {
			t.Errorf("ID at %d: expected %v, got %v", i, v, idCol.Get(i))
		}
	}
}

func TestDataTable_Merge_Horizontal_RowNames(t *testing.T) {
	dt1 := NewDataTable(
		NewDataList(1, 2).SetName("V1"),
	)
	dt1.SetRowNameByIndex(0, "R1")
	// Row 1 is nameless

	dt2 := NewDataTable(
		NewDataList(10, 20).SetName("V2"),
	)
	dt2.SetRowNameByIndex(0, "R1")
	dt2.SetRowNameByIndex(1, "R2")

	// Outer Join on row names
	res, err := dt1.Merge(dt2, MergeDirectionHorizontal, MergeModeOuter)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	// Expected rows:
	// 1. R1 (matched)
	// 2. R2 (from dt2)
	// 3. Nameless (from dt1)
	if res.getMaxColLength() != 3 {
		t.Errorf("Expected 3 rows, got %d", res.getMaxColLength())
	}

	// Check row names
	names := make(map[string]bool)
	for i := 0; i < res.getMaxColLength(); i++ {
		if name, ok := res.GetRowNameByIndex(i); ok {
			names[name] = true
		}
	}

	if !names["R1"] {
		t.Errorf("Expected row name R1 not found")
	}
	if !names["R2"] {
		t.Errorf("Expected row name R2 not found")
	}
}
