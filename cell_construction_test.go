package insyra

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

// NewDataList flattens slices on purpose, so a list reads like a pandas
// Series. Cell is how a caller opts one argument out of that, inline among
// ordinary values, which Append cannot do because it is a different call.

func TestCellKeepsOneValueInOneCell(t *testing.T) {
	dl := NewDataList(Cell([]int{1, 2}), 3, "a")
	if dl.Len() != 3 {
		t.Fatalf("got %d cells, want 3", dl.Len())
	}
	got, ok := dl.Get(0).([]int)
	if !ok {
		t.Fatalf("the cell holds %T, want []int — the marker must not be stored", dl.Get(0))
	}
	if !reflect.DeepEqual(got, []int{1, 2}) {
		t.Errorf("the cell holds %v", got)
	}
	if dl.Get(1) != 3 || dl.Get(2) != "a" {
		t.Errorf("the other values changed: %v, %v", dl.Get(1), dl.Get(2))
	}
}

func TestAnUnmarkedSliceStillFlattens(t *testing.T) {
	if n := NewDataList([]int{1, 2}).Len(); n != 2 {
		t.Errorf("got %d cells, want 2 — flattening is deliberate", n)
	}
	if n := NewDataList([]byte("ab")).Len(); n != 2 {
		t.Errorf("a bare []byte gave %d cells, want 2", n)
	}
}

// Append never flattens, so Cell is a no-op there. It still has to be
// accepted: someone will write it for consistency, and storing the marker
// would put a value nothing understands into the table.
func TestEveryEntryPointUnwrapsTheMark(t *testing.T) {
	v := []int{7, 8}

	t.Run("Append", func(t *testing.T) {
		dl := NewDataList()
		dl.Append(Cell(v))
		assertHolds(t, dl.Get(0), v)
	})
	t.Run("Update", func(t *testing.T) {
		dl := NewDataList(1)
		dl.Update(0, Cell(v))
		assertHolds(t, dl.Get(0), v)
	})
	t.Run("InsertAt", func(t *testing.T) {
		dl := NewDataList(1)
		dl.InsertAt(0, Cell(v))
		assertHolds(t, dl.Get(0), v)
	})
	t.Run("ReplaceFirst", func(t *testing.T) {
		dl := NewDataList(1, 2)
		dl.ReplaceFirst(1, Cell(v))
		assertHolds(t, dl.Get(0), v)
	})
	t.Run("ReplaceAll", func(t *testing.T) {
		dl := NewDataList(1, 1)
		dl.ReplaceAll(1, Cell(v))
		assertHolds(t, dl.Get(0), v)
		assertHolds(t, dl.Get(1), v)
	})
	t.Run("UpdateElement", func(t *testing.T) {
		dt := NewDataTable(NewDataList(1).SetName("c"))
		dt.UpdateElement(0, "A", Cell(v))
		assertHolds(t, dt.GetElementByNumberIndex(0, 0), v)
	})
	t.Run("AppendRowsByColIndex", func(t *testing.T) {
		dt := NewDataTable(NewDataList(1).SetName("c"))
		dt.AppendRowsByColIndex(map[string]any{"A": Cell(v)})
		assertHolds(t, dt.GetElementByNumberIndex(1, 0), v)
	})
	t.Run("AppendRowsByColName", func(t *testing.T) {
		dt := NewDataTable(NewDataList(1).SetName("c"))
		dt.AppendRowsByColName(map[string]any{"c": Cell(v)})
		assertHolds(t, dt.GetElementByNumberIndex(1, 0), v)
	})
}

func TestSearchingWithTheMarkAgrees(t *testing.T) {
	v := []int{7, 8}
	dl := NewDataList(Cell(v), Cell(v), 1)
	if got, want := dl.Count(Cell(v)), dl.Count(v); got != want {
		t.Errorf("Count(Cell(v)) = %d, Count(v) = %d — they must agree", got, want)
	}
	if got := dl.Count(v); got != 2 {
		t.Errorf("Count = %d, want 2", got)
	}
}

func TestTheMarkNeverReachesTheOutput(t *testing.T) {
	dl := NewDataList(Cell([]int{1, 2}))
	dt := NewDataTable(dl.SetName("c"))
	var buf bytes.Buffer
	dl.ShowTo(&buf)
	for _, out := range []string{dt.ToJSON_String(true), buf.String()} {
		if strings.Contains(out, "cellMarker") || strings.Contains(out, "insyra.cell") {
			t.Errorf("the marker leaked into output: %s", out)
		}
	}
}

func assertHolds(t *testing.T, got any, want []int) {
	t.Helper()
	v, ok := got.([]int)
	if !ok {
		t.Fatalf("cell holds %T, want []int — the marker was stored", got)
	}
	if !reflect.DeepEqual(v, want) {
		t.Errorf("cell holds %v, want %v", v, want)
	}
}
