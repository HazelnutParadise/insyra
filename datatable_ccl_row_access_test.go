package insyra

import (
	"testing"
)

func TestDataTable_ExecuteCCL_RowAccess(t *testing.T) {
	// 創建測試 DataTable
	dt := NewDataTable(
		NewDataList(1, 2, 3, 4).SetName("A"),
		NewDataList(10, 20, 30, 40).SetName("B"),
	)

	// 測試 A.0 (第一列的值)
	dt.ExecuteCCL("B = A.0 + A")

	// 驗證結果: B[i] = A[0] + A[i]
	// B[0] = 1 + 1 = 2
	// B[1] = 1 + 2 = 3
	// B[2] = 1 + 3 = 4
	// B[3] = 1 + 4 = 5
	expected := []any{float64(2), float64(3), float64(4), float64(5)}
	colB := dt.GetColByName("B")
	for i, v := range colB.Data() {
		if v != expected[i] {
			t.Errorf("Row %d: expected %v, got %v", i, expected[i], v)
		}
	}
}

func TestDataTable_ExecuteCCL_RowAccessWithName(t *testing.T) {
	// 創建測試 DataTable
	dt := NewDataTable(
		NewDataList(1, 2, 3, 4).SetName("A"),
		NewDataList(10, 20, 30, 40).SetName("B"),
	)
	dt.SetRowNameByIndex(1, "Jack")
	dt.SetRowNameByIndex(2, "Rose")

	// 測試 [A].'Jack' (第二列的值)
	dt.ExecuteCCL("B = [A].'Jack' + A")

	// 驗證結果: B[i] = A[1] + A[i]
	// B[0] = 2 + 1 = 3
	// B[1] = 2 + 2 = 4
	// B[2] = 2 + 3 = 5
	// B[3] = 2 + 4 = 6
	expected := []any{float64(3), float64(4), float64(5), float64(6)}
	colB := dt.GetColByName("B")
	for i, v := range colB.Data() {
		if v != expected[i] {
			t.Errorf("Row %d: expected %v, got %v", i, expected[i], v)
		}
	}
}

func TestDataTable_ExecuteCCL_RowAccessVariations(t *testing.T) {
	// 創建測試 DataTable
	dt := NewDataTable(
		NewDataList(1.0, 2.0, 3.0, 4.0).SetName("A"),
		NewDataList(10.0, 20.0, 30.0, 40.0).SetName("B"),
	)
	dt.SetRowNameByIndex(0, "First")

	// 1. Identifier: A.0
	dt.ExecuteCCL("B = A.0")
	if dt.GetColByName("B").Data()[0] != 1.0 {
		t.Errorf("A.0 failed, got %v", dt.GetColByName("B").Data()[0])
	}

	// 2. Column Index: [A].0
	dt.ExecuteCCL("B = [A].0")
	if dt.GetColByName("B").Data()[0] != 1.0 {
		t.Errorf("[A].0 failed, got %v", dt.GetColByName("B").Data()[0])
	}

	// 3. Column Name: ['A'].0
	dt.ExecuteCCL("B = ['A'].0")
	if dt.GetColByName("B").Data()[0] != 1.0 {
		t.Errorf("['A'].0 failed, got %v", dt.GetColByName("B").Data()[0])
	}

	// 4. Row Name: A.'First'
	dt.ExecuteCCL("B = A.'First'")
	if dt.GetColByName("B").Data()[0] != 1.0 {
		t.Errorf("A.'First' failed, got %v", dt.GetColByName("B").Data()[0])
	}

	// 5. Column Index + Row Name: [A].'First'
	dt.ExecuteCCL("B = [A].'First'")
	if dt.GetColByName("B").Data()[0] != 1.0 {
		t.Errorf("[A].'First' failed, got %v", dt.GetColByName("B").Data()[0])
	}

	// 6. Column Name + Row Name: ['A'].'First'
	dt.ExecuteCCL("B = ['A'].'First'")
	if dt.GetColByName("B").Data()[0] != 1.0 {
		t.Errorf("['A'].'First' failed, got %v", dt.GetColByName("B").Data()[0])
	}
}

func TestDataTable_ExecuteCCL_AtOperator(t *testing.T) {
	// 創建測試 DataTable
	dt := NewDataTable(
		NewDataList(1.0, 2.0, 3.0, 4.0).SetName("A"),
		NewDataList(10.0, 20.0, 30.0, 40.0).SetName("B"),
	)

	// 測試 @.0 (第一列的所有值)
	dt.ExecuteCCL("NEW('C') = @.0")

	// 驗證結果: 現在 @.0 不會重複 row 數次
	// 由於 DataTable 會自動補齊長度到 maxLength (4)，所以 C 的長度會是 4
	// 但前兩個元素應該是 [1, 10]
	colC := dt.GetColByName("C")
	data := colC.Data()
	if len(data) != 4 {
		t.Errorf("Expected 4 elements in column C (due to padding), got %d", len(data))
	}
	if data[0] != 1.0 || data[1] != 10.0 {
		t.Errorf("Expected data[0]=1, data[1]=10, got %v", data)
	}
}
