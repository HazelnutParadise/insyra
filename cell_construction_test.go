package insyra

import (
	"bytes"
	"math"
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
	t.Run("ReplaceLast", func(t *testing.T) {
		dl := NewDataList(1, 2)
		dl.ReplaceLast(1, Cell(v))
		assertHolds(t, dl.Get(0), v)
	})

	// Every replace-with and fill entry point stores a caller's value in a
	// cell, so each has to unwrap it. The DataTable methods reach the DataList
	// helpers directly, which is why the helpers unwrap and not only the
	// public DataList methods.
	t.Run("ReplaceNaNsWith", func(t *testing.T) {
		dl := NewDataList(math.NaN())
		dl.ReplaceNaNsWith(Cell(v))
		assertHolds(t, dl.Get(0), v)
	})
	t.Run("ReplaceNilsWith", func(t *testing.T) {
		dl := NewDataList(nil)
		dl.ReplaceNilsWith(Cell(v))
		assertHolds(t, dl.Get(0), v)
	})
	t.Run("ReplaceNaNsAndNilsWith", func(t *testing.T) {
		dl := NewDataList(nil)
		dl.ReplaceNaNsAndNilsWith(Cell(v))
		assertHolds(t, dl.Get(0), v)
	})
	t.Run("Shift fill", func(t *testing.T) {
		out := NewDataList(1, 2).Shift(1, Cell(v))
		assertHolds(t, out.Get(0), v)
	})

	nanTable := func() *DataTable { return NewDataTable(NewDataList(math.NaN()).SetName("c")) }
	nilTable := func() *DataTable { return NewDataTable(NewDataList(nil).SetName("c")) }
	oneTable := func() *DataTable { return NewDataTable(NewDataList(1).SetName("c")) }
	for _, c := range []struct {
		name  string
		table func() *DataTable
		call  func(dt *DataTable)
	}{
		{"DataTable.Replace", oneTable, func(dt *DataTable) { dt.Replace(1, Cell(v)) }},
		{"DataTable.ReplaceNaNsWith", nanTable, func(dt *DataTable) { dt.ReplaceNaNsWith(Cell(v)) }},
		{"DataTable.ReplaceNilsWith", nilTable, func(dt *DataTable) { dt.ReplaceNilsWith(Cell(v)) }},
		{"DataTable.ReplaceNaNsAndNilsWith", nilTable, func(dt *DataTable) { dt.ReplaceNaNsAndNilsWith(Cell(v)) }},
		{"DataTable.ReplaceInRow", oneTable, func(dt *DataTable) { dt.ReplaceInRow(0, 1, Cell(v)) }},
		{"DataTable.ReplaceNaNsInRow", nanTable, func(dt *DataTable) { dt.ReplaceNaNsInRow(0, Cell(v)) }},
		{"DataTable.ReplaceNilsInRow", nilTable, func(dt *DataTable) { dt.ReplaceNilsInRow(0, Cell(v)) }},
		{"DataTable.ReplaceNaNsAndNilsInRow", nilTable, func(dt *DataTable) { dt.ReplaceNaNsAndNilsInRow(0, Cell(v)) }},
		{"DataTable.ReplaceInCol", oneTable, func(dt *DataTable) { dt.ReplaceInCol("A", 1, Cell(v)) }},
		{"DataTable.ReplaceInColByName", oneTable, func(dt *DataTable) { dt.ReplaceInColByName("c", 1, Cell(v)) }},
		{"DataTable.ReplaceNaNsInCol", nanTable, func(dt *DataTable) { dt.ReplaceNaNsInCol("A", Cell(v)) }},
		{"DataTable.ReplaceNaNsInColByName", nanTable, func(dt *DataTable) { dt.ReplaceNaNsInColByName("c", Cell(v)) }},
		{"DataTable.ReplaceNilsInCol", nilTable, func(dt *DataTable) { dt.ReplaceNilsInCol("A", Cell(v)) }},
		{"DataTable.ReplaceNilsInColByName", nilTable, func(dt *DataTable) { dt.ReplaceNilsInColByName("c", Cell(v)) }},
		{"DataTable.ReplaceNaNsAndNilsInCol", nilTable, func(dt *DataTable) { dt.ReplaceNaNsAndNilsInCol("A", Cell(v)) }},
		{"DataTable.ReplaceNaNsAndNilsInColByName", nilTable, func(dt *DataTable) { dt.ReplaceNaNsAndNilsInColByName("c", Cell(v)) }},
	} {
		t.Run(c.name, func(t *testing.T) {
			dt := c.table()
			c.call(dt)
			assertHolds(t, dt.GetElementByNumberIndex(0, 0), v)
		})
	}
}

// The mark is removed once, whatever the entry point, so a doubly marked value
// is stored the same way by Append and by a replace.
func TestADoubleMarkIsUnwrappedOnceEverywhere(t *testing.T) {
	v := []int{7, 8}
	appended := NewDataList().Append(Cell(Cell(v))).Get(0)
	replaced := NewDataList(1).ReplaceAll(1, Cell(Cell(v))).Get(0)
	if !reflect.DeepEqual(appended, replaced) {
		t.Errorf("Append stored %#v, ReplaceAll stored %#v", appended, replaced)
	}
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
	for _, out := range []string{dt.ToJSONString(true), buf.String()} {
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
