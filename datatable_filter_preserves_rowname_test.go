package insyra

import (
	"testing"
)

func TestFilterPreservesRowNames(t *testing.T) {
	dt := NewDataTable()
	col1 := NewDataList()
	col1.Append(1, 2, 3)
	col1.name = "ID"
	dt.AppendCols(col1)
	dt.SetRowNames([]string{"Row0", "Row1", "Row2"})

	// Filter rows where ID > 1 (keeps Row1, Row2)
	filtered := dt.Filter(func(rowIndex int, columnIndex string, value any) bool {
		return value.(int) > 1
	})

	if filtered.getMaxColLength() != 2 {
		t.Fatalf("Expected 2 rows, got %d", filtered.getMaxColLength())
	}

	name0, ok0 := filtered.GetRowNameByIndex(0)
	name1, ok1 := filtered.GetRowNameByIndex(1)

	if !ok0 || name0 != "Row1" {
		t.Errorf("Expected row 0 to be Row1, got %s (ok: %v)", name0, ok0)
	}
	if !ok1 || name1 != "Row2" {
		t.Errorf("Expected row 1 to be Row2, got %s (ok: %v)", name1, ok1)
	}
}

func TestFilterRowsByRowNameContainsPreservesRowNames(t *testing.T) {
	dt := NewDataTable()
	col1 := NewDataList()
	col1.Append(1, 2, 3)
	dt.AppendCols(col1)
	dt.SetRowNames([]string{"Apple", "Banana", "Grape"})

	// Filter rows containing "p" (Apple, Grape)
	filtered := dt.FilterRowsByRowNameContains("p")

	for i := 0; i < filtered.getMaxColLength(); i++ {
		name, _ := filtered.GetRowNameByIndex(i)
		t.Logf("Row %d name: %s", i, name)
	}

	if filtered.getMaxColLength() != 2 {
		t.Fatalf("Expected 2 rows, got %d", filtered.getMaxColLength())
	}

	name0, ok0 := filtered.GetRowNameByIndex(0)
	name1, ok1 := filtered.GetRowNameByIndex(1)

	if !ok0 || name0 != "Apple" {
		t.Errorf("Expected row 0 to be Apple, got %s (ok: %v)", name0, ok0)
	}
	if !ok1 || name1 != "Grape" {
		t.Errorf("Expected row 1 to be Grape, got %s (ok: %v)", name1, ok1)
	}
}
