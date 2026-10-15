package insyra

import (
	"math"
	"testing"
)

func equalityTable() *DataTable {
	dt := NewDataTable(
		NewDataList(1, 2).SetName("num"),
		NewDataList("a", "b").SetName("txt"),
	)
	dt.SetName("t")
	dt.SetRowNames([]string{"r1", "r2"})
	return dt
}

func TestDataTableIsEqualToIgnoresNames(t *testing.T) {
	a := equalityTable()
	b := NewDataTable(
		NewDataList(1, 2).SetName("x"),
		NewDataList("a", "b").SetName("y"),
	)
	b.SetName("other")
	b.SetRowNames([]string{"p", "q"})
	if !a.IsEqualTo(b) {
		t.Fatal("same cells under different names should be equal")
	}
	if a.IsTheSameAs(b) {
		t.Fatal("different names should not be the same")
	}
}

func TestDataTableIsEqualToComparesCells(t *testing.T) {
	a := equalityTable()

	typed := equalityTable()
	typed.UpdateElement(0, "A", int64(1))
	if a.IsEqualTo(typed) {
		t.Error("int(1) and int64(1) should differ")
	}

	changed := equalityTable()
	changed.UpdateElement(1, "B", "c")
	if a.IsEqualTo(changed) {
		t.Error("a changed cell should differ")
	}

	nanA := NewDataTable(NewDataList(1.0, math.NaN()))
	nanB := NewDataTable(NewDataList(1.0, math.NaN()))
	if !nanA.IsEqualTo(nanB) {
		t.Error("two NaN cells should be equal")
	}
}

func TestDataTableIsEqualToComparesShape(t *testing.T) {
	a := equalityTable()

	moreRows := equalityTable()
	moreRows.AppendRowsByColIndex(map[string]any{"A": 3, "B": "c"})
	if a.IsEqualTo(moreRows) || moreRows.IsEqualTo(a) {
		t.Error("an extra row should differ")
	}

	moreCols := equalityTable()
	moreCols.AppendCols(NewDataList(true, false).SetName("flag"))
	if a.IsEqualTo(moreCols) || moreCols.IsEqualTo(a) {
		t.Error("an extra column should differ")
	}
}

func TestDataTableShortColumnReadsAsNil(t *testing.T) {
	short := &DataTable{columns: []*DataList{NewDataList(1, 2), NewDataList(3)}}
	padded := &DataTable{columns: []*DataList{NewDataList(1, 2), NewDataList(3, nil)}}
	if !short.IsEqualTo(padded) {
		t.Fatal("a cell past the end of a short column should count as nil")
	}
}

func TestDataTableIsTheSameAsComparesEveryName(t *testing.T) {
	a := equalityTable()
	if !a.IsTheSameAs(a.Clone()) {
		t.Fatal("a table and its clone should be the same")
	}

	rowName := equalityTable()
	rowName.SetRowNameByIndex(1, "other")
	if !a.IsEqualTo(rowName) || a.IsTheSameAs(rowName) {
		t.Error("one row name differing should be equal but not the same")
	}

	unnamedRow := NewDataTable(
		NewDataList(1, 2).SetName("num"),
		NewDataList("a", "b").SetName("txt"),
	)
	unnamedRow.SetName("t")
	unnamedRow.SetRowNameByIndex(0, "r1")
	if a.IsTheSameAs(unnamedRow) || unnamedRow.IsTheSameAs(a) {
		t.Error("a row without a name should differ from a named row")
	}

	colName := equalityTable()
	colName.SetColNameByIndex("B", "other")
	if !a.IsEqualTo(colName) || a.IsTheSameAs(colName) {
		t.Error("one column name differing should be equal but not the same")
	}

	tableName := equalityTable()
	tableName.SetName("other")
	if !a.IsEqualTo(tableName) || a.IsTheSameAs(tableName) {
		t.Error("a different table name should be equal but not the same")
	}
}

func TestDataTableEqualityIdentityAndNil(t *testing.T) {
	a := equalityTable()
	if !a.IsEqualTo(a) || !a.IsTheSameAs(a) {
		t.Error("a table should equal itself")
	}
	if a.IsEqualTo(nil) || a.IsTheSameAs(nil) {
		t.Error("nil should not be equal")
	}
	if a.Err() != nil {
		t.Errorf("a nil argument should not record an error, got %v", a.Err())
	}
}
