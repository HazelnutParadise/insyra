package insyra

import (
	"reflect"
	"strings"
	"testing"
)

func rowsWhereFixture() *DataTable {
	dt := NewDataTable(
		NewDataList(1, 5, 3, 8).SetName("price"),
		NewDataList(2, 4, 3, 1).SetName("cost"),
	)
	dt.SetName("orders")
	dt.SetRowNameByIndex(0, "r0")
	dt.SetRowNameByIndex(1, "r1")
	dt.SetRowNameByIndex(3, "r3")
	return dt
}

// A condition across columns, the case Filter and FilterRows cannot express
// because they judge one cell at a time.
func TestFilterRowsWhereComparesAcrossColumns(t *testing.T) {
	dt := rowsWhereFixture()
	got := dt.FilterRowsWhere(func(row *DataList) bool {
		return ToFloat64(row.Get(0)) > ToFloat64(row.Get(1))
	})
	if e := got.Err(); e != nil {
		t.Fatalf("FilterRowsWhere recorded %v", e)
	}
	if want := [][]any{{5, 4}, {8, 1}}; !reflect.DeepEqual(got.To2DSlice(), want) {
		t.Fatalf("FilterRowsWhere kept %v, want %v", got.To2DSlice(), want)
	}
	if names := got.ColNames(); !reflect.DeepEqual(names, []string{"price", "cost"}) {
		t.Errorf("column names = %v", names)
	}
	if got.GetName() != "orders" {
		t.Errorf("table name = %q, want orders", got.GetName())
	}
	if n, _ := got.GetRowNameByIndex(0); n != "r1" {
		t.Errorf("row 0 is named %q, want r1", n)
	}
	if n, _ := got.GetRowNameByIndex(1); n != "r3" {
		t.Errorf("row 1 is named %q, want r3", n)
	}
}

// The predicate sees the whole row once, in column order, named after the
// row, and a slice cell stays one cell.
func TestFilterRowsWhereHandsOverTheRow(t *testing.T) {
	blob := []byte{0, 255}
	dt := NewDataTable(
		NewDataList("a", "b").SetName("key"),
		NewDataList(Cell(blob), nil).SetName("blob"),
	)
	dt.SetRowNameByIndex(1, "second")
	var seen [][]any
	var names []string
	dt.FilterRowsWhere(func(row *DataList) bool {
		seen = append(seen, row.Data())
		names = append(names, row.GetName())
		return true
	})
	want := [][]any{{"a", blob}, {"b", nil}}
	if !reflect.DeepEqual(seen, want) {
		t.Fatalf("predicate saw %#v, want %#v", seen, want)
	}
	if !reflect.DeepEqual(names, []string{"", "second"}) {
		t.Fatalf("predicate saw row names %q", names)
	}
}

// Changing the row inside the predicate changes neither the source nor the
// result: the result holds the table's own cells.
func TestFilterRowsWhereRowIsACopy(t *testing.T) {
	dt := rowsWhereFixture()
	got := dt.FilterRowsWhere(func(row *DataList) bool {
		row.Update(0, 999)
		return true
	})
	if !reflect.DeepEqual(got.To2DSlice(), dt.To2DSlice()) {
		t.Fatalf("result %v differs from source %v", got.To2DSlice(), dt.To2DSlice())
	}
	if v := dt.GetElementByNumberIndex(0, 0); v != 1 {
		t.Fatalf("source cell changed to %v", v)
	}
	got.UpdateElement(0, "A", -1)
	if v := dt.GetElementByNumberIndex(0, 0); v != 1 {
		t.Fatalf("editing the result changed the source to %v", v)
	}
}

func TestFilterRowsWhereNoMatchKeepsColumns(t *testing.T) {
	got := rowsWhereFixture().FilterRowsWhere(func(*DataList) bool { return false })
	if got.NumRows() != 0 || got.NumCols() != 2 {
		t.Fatalf("no match gave %d rows x %d cols, want 0 x 2", got.NumRows(), got.NumCols())
	}
	if names := got.ColNames(); !reflect.DeepEqual(names, []string{"price", "cost"}) {
		t.Errorf("column names = %v", names)
	}
}

func TestFilterRowsWhereNilPredicate(t *testing.T) {
	dt := rowsWhereFixture()
	got := dt.FilterRowsWhere(nil)
	if got == nil {
		t.Fatal("FilterRowsWhere(nil) returned nil")
	}
	if e := dt.Err(); e == nil || !strings.Contains(e.Message, "nil") {
		t.Fatalf("FilterRowsWhere(nil) recorded %v, want an error about the nil function", e)
	}
}

// Filter keeps a row when any one cell passes. This is the documented
// meaning, pinned so it cannot drift towards a row predicate.
func TestFilterKeepsARowWhenAnyCellPasses(t *testing.T) {
	dt := rowsWhereFixture()
	got := dt.Filter(func(_ int, _ string, v any) bool { return v == 3 || v == 4 })
	if want := [][]any{{5, 4}, {3, 3}}; !reflect.DeepEqual(got.To2DSlice(), want) {
		t.Fatalf("Filter kept %v, want %v", got.To2DSlice(), want)
	}
	// The column argument is the Excel-style letter, not the column's name.
	var cols []string
	dt.Filter(func(row int, col string, _ any) bool {
		if row == 0 {
			cols = append(cols, col)
		}
		return false
	})
	if !reflect.DeepEqual(cols, []string{"A", "B"}) {
		t.Fatalf("Filter passed columns %v, want [A B]", cols)
	}
}

// FilterByCustomElement is Filter with the row and column arguments dropped.
// Checked on the fixture, on a table with nil and slice cells, and on an
// empty table, for a predicate that keeps some, all and no rows.
func TestFilterByCustomElementEqualsFilter(t *testing.T) {
	withNil := NewDataTable(
		NewDataList(nil, 2, "x").SetName("a"),
		NewDataList(Cell([]byte{1}), nil, 3.5).SetName("b"),
	)
	withNil.SetRowNameByIndex(2, "last")
	tables := map[string]*DataTable{
		"fixture": rowsWhereFixture(),
		"nil":     withNil,
		"empty":   NewDataTable(),
	}
	predicates := map[string]func(any) bool{
		"some": func(v any) bool { f, ok := ToFloat64Safe(v); return ok && f > 2 },
		"all":  func(any) bool { return true },
		"none": func(any) bool { return false },
		"nil":  func(v any) bool { return v == nil },
	}
	for tn, dt := range tables {
		for pn, f := range predicates {
			a := dt.FilterByCustomElement(f)
			b := dt.Filter(func(_ int, _ string, v any) bool { return f(v) })
			if !reflect.DeepEqual(a.To2DSlice(), b.To2DSlice()) ||
				!reflect.DeepEqual(a.ColNames(), b.ColNames()) ||
				!reflect.DeepEqual(a.RowNames(), b.RowNames()) ||
				a.GetName() != b.GetName() {
				t.Errorf("%s/%s: FilterByCustomElement gave %v %v %v, Filter gave %v %v %v", tn, pn,
					a.To2DSlice(), a.ColNames(), a.RowNames(), b.To2DSlice(), b.ColNames(), b.RowNames())
			}
		}
	}
}

// keep may call the table's own methods. A column it adds is not in the
// result and does not crash the call.
func TestFilterRowsWhereSurvivesAColumnAddedByKeep(t *testing.T) {
	dt := rowsWhereFixture()
	added := false
	got := dt.FilterRowsWhere(func(*DataList) bool {
		if !added {
			added = true
			dt.AppendCols(NewDataList(0, 0, 0, 0).SetName("extra"))
		}
		return true
	})
	if got.NumCols() != 2 || got.NumRows() != 4 {
		t.Fatalf("result is %d x %d, want 4 x 2", got.NumRows(), got.NumCols())
	}
}

// Filter and FilterRows call their function inside the table's lock too, and
// crashed with an index out of range when it added a column.
func TestFilterAndFilterRowsSurviveAColumnAddedByTheirFunction(t *testing.T) {
	for name, run := range map[string]func(dt *DataTable, add func()) *DataTable{
		"Filter": func(dt *DataTable, add func()) *DataTable {
			return dt.Filter(func(int, string, any) bool { add(); return true })
		},
		"FilterRows": func(dt *DataTable, add func()) *DataTable {
			return dt.FilterRows(func(string, string, any) bool { add(); return true })
		},
	} {
		dt := rowsWhereFixture()
		added := false
		got := run(dt, func() {
			if !added {
				added = true
				dt.AppendCols(NewDataList(0, 0, 0, 0).SetName("extra"))
			}
		})
		if got.NumCols() != 2 || got.NumRows() != 4 {
			t.Errorf("%s: result is %d x %d, want 4 x 2", name, got.NumRows(), got.NumCols())
		}
	}
}
