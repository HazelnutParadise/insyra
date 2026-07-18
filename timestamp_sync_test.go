package insyra

import "testing"

// lastModifiedTimestamp must be updated synchronously: the new value is
// visible the moment a mutating call returns, with no goroutine-scheduling
// window. These tests reset the timestamp to 0 (in-package access) and assert
// the very next read after a mutation sees a fresh value — under the previous
// `go updateTimestamp()` form they would flake toward failure.
func TestUpdateTimestampSynchronous_DataList(t *testing.T) {
	dl := NewDataList(1, 2, 3)
	dl.lastModifiedTimestamp.Store(0)
	dl.Append(4)
	if dl.GetLastModifiedTimestamp() == 0 {
		t.Fatal("Append returned before lastModifiedTimestamp was updated")
	}
}

func TestUpdateTimestampSynchronous_DataTable(t *testing.T) {
	dt := NewDataTable(NewDataList(1, 2).SetName("c"))
	dt.lastModifiedTimestamp.Store(0)
	dt.AppendRowsByColIndex(map[string]any{"A": 3})
	if dt.GetLastModifiedTimestamp() == 0 {
		t.Fatal("AppendRowsByColIndex returned before lastModifiedTimestamp was updated")
	}
}

func TestUpdateTimestampSynchronous_DataTableColumn(t *testing.T) {
	dt := NewDataTable(NewDataList(2, 5).SetName("c"))
	dt.columns[0].lastModifiedTimestamp.Store(0)
	dt.ReplaceInRow(0, 2, 99)
	if dt.columns[0].lastModifiedTimestamp.Load() == 0 {
		t.Fatal("ReplaceInRow returned before the column's lastModifiedTimestamp was updated")
	}
}
