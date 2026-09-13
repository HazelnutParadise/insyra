package insyra

import (
	"reflect"
	"testing"
)

func TestDataTable_SortBy(t *testing.T) {
	dt := NewDataTable()
	colA := NewDataList(3, 1, 2, 1)
	colB := NewDataList("a", "b", "c", "d")
	dt.AppendCols(colA, colB)
	dt.SetColNameByNumber(0, "A")
	dt.SetColNameByNumber(1, "B").Show()

	// 多層排序：先按 A 升序，再按 B 降序
	dt.SortBy(DataTableSortConfig{ColumnName: "A", Descending: false}, DataTableSortConfig{ColumnName: "B", Descending: true}).Show()

	// 預期結果：A 升序，對於相同 A 按 B 降序
	// 原始：A:3,1,2,1 B:a,c,b,d
	// 排序後：A:1,1,2,3 B:d,c,b,a (第一個1的d > c，第二個1的c)
	aData := dt.GetColByName("A").Data()
	bData := dt.GetColByName("B").Data()
	expectedA := []any{1, 1, 2, 3}
	expectedB := []any{"d", "b", "c", "a"}

	if !reflect.DeepEqual(aData, expectedA) {
		t.Errorf("SortBy A failed: got %v, expected %v", aData, expectedA)
	}
	if !reflect.DeepEqual(bData, expectedB) {
		t.Errorf("SortBy B failed: got %v, expected %v", bData, expectedB)
	}
}

func TestDataTable_SortBy_SingleColumn(t *testing.T) {
	dt := NewDataTable()
	colA := NewDataList(3, 1, 2)
	dt.AppendCols(colA)
	dt.SetColNameByNumber(0, "A")

	// 單列降序
	dt.SortBy(DataTableSortConfig{ColumnName: "A", Descending: true})

	aData := dt.GetColByName("A").Data()
	expectedA := []any{3, 2, 1}

	if !reflect.DeepEqual(aData, expectedA) {
		t.Errorf("SortBy single column failed: got %v, expected %v", aData, expectedA)
	}
}

func TestDataTable_SortBy_ByIndex(t *testing.T) {
	dt := NewDataTable()
	colA := NewDataList(3, 1, 2)
	colB := NewDataList("a", "b", "c")
	dt.AppendCols(colA, colB)

	// 按列索引 0 升序
	dt.SortBy(DataTableSortConfig{ColumnNumber: 0, Descending: false})

	aData := dt.GetColByNumber(0).Data()
	bData := dt.GetColByNumber(1).Data()
	expectedA := []any{1, 2, 3}
	expectedB := []any{"b", "c", "a"}

	if !reflect.DeepEqual(aData, expectedA) {
		t.Errorf("SortBy by index A failed: got %v, expected %v", aData, expectedA)
	}
	if !reflect.DeepEqual(bData, expectedB) {
		t.Errorf("SortBy by index B failed: got %v, expected %v", bData, expectedB)
	}
}
