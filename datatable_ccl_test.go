package insyra

import (
	"testing"
)

func TestDataTable_ExecuteCCL_Assignment(t *testing.T) {
	// 創建測試 DataTable
	dt := NewDataTable(
		NewDataList(1, 2, 3, 4).SetName("A"),
		NewDataList(10, 20, 30, 40).SetName("B"),
		NewDataList(100, 200, 300, 400).SetName("C"),
	)

	// 使用賦值語法修改 B 列: B = A + C
	dt.ExecuteCCL("B = A + C")

	// 驗證結果
	expected := []any{float64(101), float64(202), float64(303), float64(404)}
	colB := dt.GetColByName("B")
	if colB == nil {
		t.Fatal("Column B not found")
	}
	for i, v := range colB.Data() {
		if v != expected[i] {
			t.Errorf("Row %d: expected %v, got %v", i, expected[i], v)
		}
	}
}

func TestDataTable_ExecuteCCL_AssignmentWithColIndex(t *testing.T) {
	// 創建測試 DataTable
	dt := NewDataTable(
		NewDataList(1, 2, 3, 4).SetName("X"),
		NewDataList(10, 20, 30, 40).SetName("Y"),
	)

	// 使用索引形式的賦值語法修改 A 列: A = A * 2
	dt.ExecuteCCL("A = A * 2")

	// 驗證結果
	expected := []any{float64(2), float64(4), float64(6), float64(8)}
	colA := dt.GetColByNumber(0)
	if colA == nil {
		t.Fatal("Column A (index 0) not found")
	}
	for i, v := range colA.Data() {
		if v != expected[i] {
			t.Errorf("Row %d: expected %v, got %v", i, expected[i], v)
		}
	}
}

func TestDataTable_ExecuteCCL_NewColumn(t *testing.T) {
	// 創建測試 DataTable
	dt := NewDataTable(
		NewDataList(1, 2, 3, 4).SetName("A"),
		NewDataList(10, 20, 30, 40).SetName("B"),
	)

	// 使用 NEW 函數創建新列
	dt.ExecuteCCL("NEW('Sum') = A + B")

	// 驗證結果
	expected := []any{float64(11), float64(22), float64(33), float64(44)}
	colSum := dt.GetColByName("Sum")
	if colSum == nil {
		t.Fatal("Column Sum not found")
	}
	for i, v := range colSum.Data() {
		if v != expected[i] {
			t.Errorf("Row %d: expected %v, got %v", i, expected[i], v)
		}
	}
}

func TestDataTable_ExecuteCCL_MultilineStatements(t *testing.T) {
	// 創建測試 DataTable
	dt := NewDataTable(
		NewDataList(1, 2, 3, 4).SetName("A"),
		NewDataList(10, 20, 30, 40).SetName("B"),
		NewDataList(100, 200, 300, 400).SetName("C"),
	)

	// 執行多行 CCL
	dt.ExecuteCCL(`
		A = A * 10
		NEW('D') = A + B + C
	`)

	// 驗證 A 列被修改
	expectedA := []any{float64(10), float64(20), float64(30), float64(40)}
	colA := dt.GetColByName("A")
	for i, v := range colA.Data() {
		if v != expectedA[i] {
			t.Errorf("Column A, Row %d: expected %v, got %v", i, expectedA[i], v)
		}
	}

	// 驗證 D 列被創建 (D = A + B + C, A已經是修改後的值)
	expectedD := []any{float64(120), float64(240), float64(360), float64(480)}
	colD := dt.GetColByName("D")
	if colD == nil {
		t.Fatal("Column D not found")
	}
	for i, v := range colD.Data() {
		if v != expectedD[i] {
			t.Errorf("Column D, Row %d: expected %v, got %v", i, expectedD[i], v)
		}
	}
}

func TestDataTable_ExecuteCCL_MultilineWithSemicolon(t *testing.T) {
	// 創建測試 DataTable
	dt := NewDataTable(
		NewDataList(1, 2, 3).SetName("X"),
		NewDataList(5, 6, 7).SetName("Y"),
	)

	// 執行多行 CCL（使用分號分隔）
	dt.ExecuteCCL("A = A + 1; NEW('Z') = A * B")

	// 驗證 A 列（第一列，索引為 A）被修改
	expectedA := []any{float64(2), float64(3), float64(4)}
	colA := dt.GetColByNumber(0)
	for i, v := range colA.Data() {
		if v != expectedA[i] {
			t.Errorf("Column A, Row %d: expected %v, got %v", i, expectedA[i], v)
		}
	}

	// 驗證 Z 列被創建 (Z = A * B, A已經是修改後的值, B是第二列)
	expectedZ := []any{float64(10), float64(18), float64(28)}
	colZ := dt.GetColByName("Z")
	if colZ == nil {
		t.Fatal("Column Z not found")
	}
	for i, v := range colZ.Data() {
		if v != expectedZ[i] {
			t.Errorf("Column Z, Row %d: expected %v, got %v", i, expectedZ[i], v)
		}
	}
}

func TestDataTable_ExecuteCCL_AssignmentWithColName(t *testing.T) {
	// 創建測試 DataTable
	dt := NewDataTable(
		NewDataList(1, 2, 3).SetName("price"),
		NewDataList(2, 3, 4).SetName("quantity"),
	)

	// 使用 ['colName'] 語法進行賦值
	dt.ExecuteCCL("['price'] = ['price'] * ['quantity']")

	// 驗證結果
	expected := []any{float64(2), float64(6), float64(12)}
	colPrice := dt.GetColByName("price")
	for i, v := range colPrice.Data() {
		if v != expected[i] {
			t.Errorf("Row %d: expected %v, got %v", i, expected[i], v)
		}
	}
}

func TestDataTable_ExecuteCCL_NonExistentColumnError(t *testing.T) {
	// 創建測試 DataTable
	dt := NewDataTable(
		NewDataList(1, 2, 3).SetName("A"),
	)

	// 嘗試對不存在的列進行賦值應該報錯（但不會 panic）
	// ExecuteCCL 會記錄警告但不會崩潰
	dt.ExecuteCCL("NonExistent = A + 1")

	// 確保 DataTable 保持不變
	if len(dt.columns) != 1 {
		t.Errorf("Expected 1 column, got %d", len(dt.columns))
	}
}
