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

func TestDataTable_EditColByIndexUsingCCL(t *testing.T) {
	// 創建測試 DataTable
	dt := NewDataTable(
		NewDataList(1, 2, 3, 4).SetName("X"),
		NewDataList(10, 20, 30, 40).SetName("Y"),
		NewDataList(100, 200, 300, 400).SetName("Z"),
	)

	// 使用 CCL 編輯第一欄 (A)
	dt.EditColByIndexUsingCCL("A", "A * 10")

	// 驗證結果
	expected := []any{float64(10), float64(20), float64(30), float64(40)}
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

func TestDataTable_EditColByIndexUsingCCL_WithOtherCols(t *testing.T) {
	// 創建測試 DataTable
	dt := NewDataTable(
		NewDataList(1, 2, 3).SetName("A"),
		NewDataList(10, 20, 30).SetName("B"),
		NewDataList(100, 200, 300).SetName("C"),
	)

	// 使用其他欄位的值編輯第二欄 (B)
	dt.EditColByIndexUsingCCL("B", "A + ['C']")

	// 驗證結果 B = A + C
	expected := []any{float64(101), float64(202), float64(303)}
	colB := dt.GetColByNumber(1)
	for i, v := range colB.Data() {
		if v != expected[i] {
			t.Errorf("Row %d: expected %v, got %v", i, expected[i], v)
		}
	}
}

func TestDataTable_EditColByNameUsingCCL(t *testing.T) {
	// 創建測試 DataTable
	dt := NewDataTable(
		NewDataList(5, 10, 15, 20).SetName("price"),
		NewDataList(2, 3, 4, 5).SetName("quantity"),
	)

	// 使用 CCL 編輯 price 欄
	dt.EditColByNameUsingCCL("price", "['price'] * 2")

	// 驗證結果
	expected := []any{float64(10), float64(20), float64(30), float64(40)}
	colPrice := dt.GetColByName("price")
	if colPrice == nil {
		t.Fatal("Column 'price' not found")
	}
	for i, v := range colPrice.Data() {
		if v != expected[i] {
			t.Errorf("Row %d: expected %v, got %v", i, expected[i], v)
		}
	}
}

func TestDataTable_EditColByNameUsingCCL_WithCondition(t *testing.T) {
	// 創建測試 DataTable
	dt := NewDataTable(
		NewDataList(10, 20, 30, 40).SetName("value"),
		NewDataList(5, 25, 15, 35).SetName("threshold"),
	)

	// 使用條件表達式編輯欄位
	dt.EditColByNameUsingCCL("value", "IF(['value'] > ['threshold'], ['value'] * 2, ['value'])")

	// 驗證結果: value > threshold 時 value * 2, 否則保持原值
	// Row 0: 10 > 5 -> 20
	// Row 1: 20 > 25 -> 20 (不變)
	// Row 2: 30 > 15 -> 60
	// Row 3: 40 > 35 -> 80
	expected := []float64{20, 20, 60, 80}
	colValue := dt.GetColByName("value")
	for i, v := range colValue.Data() {
		var got float64
		switch val := v.(type) {
		case float64:
			got = val
		case int:
			got = float64(val)
		default:
			t.Errorf("Row %d: unexpected type %T", i, v)
			continue
		}
		if got != expected[i] {
			t.Errorf("Row %d: expected %v, got %v", i, expected[i], got)
		}
	}
}

func TestDataTable_EditColByNameUsingCCL_NonExistent(t *testing.T) {
	// 創建測試 DataTable
	dt := NewDataTable(
		NewDataList(1, 2, 3).SetName("A"),
	)

	// 嘗試編輯不存在的欄位（應該記錄警告但不會 panic）
	dt.EditColByNameUsingCCL("NonExistent", "A + 1")

	// 確保 DataTable 保持不變
	if len(dt.columns) != 1 {
		t.Errorf("Expected 1 column, got %d", len(dt.columns))
	}
}

func TestDataTable_EditColByIndexUsingCCL_OutOfRange(t *testing.T) {
	// 創建測試 DataTable
	dt := NewDataTable(
		NewDataList(1, 2, 3).SetName("A"),
	)

	// 嘗試編輯超出範圍的欄位索引（應該記錄警告但不會 panic）
	dt.EditColByIndexUsingCCL("ZZ", "A + 1")

	// 確保 DataTable 保持不變
	if len(dt.columns) != 1 {
		t.Errorf("Expected 1 column, got %d", len(dt.columns))
	}
}

func TestDataTable_EditColByNameUsingCCL_StringConcat(t *testing.T) {
	// 創建測試 DataTable
	dt := NewDataTable(
		NewDataList("Hello", "World", "Test").SetName("greeting"),
		NewDataList("Foo", "Bar", "Baz").SetName("suffix"),
	)

	// 使用字串連接運算符
	dt.EditColByNameUsingCCL("greeting", "['greeting'] & '-' & ['suffix']")

	// 驗證結果
	expected := []any{"Hello-Foo", "World-Bar", "Test-Baz"}
	colGreeting := dt.GetColByName("greeting")
	for i, v := range colGreeting.Data() {
		if v != expected[i] {
			t.Errorf("Row %d: expected %v, got %v", i, expected[i], v)
		}
	}
}

// ==== 表達式模式拒絕賦值語法測試 ====

func TestDataTable_AddColUsingCCL_RejectsAssignment(t *testing.T) {
	dt := NewDataTable(
		NewDataList(1, 2, 3).SetName("A"),
	)

	// AddColUsingCCL 不應該接受賦值語法（在公式中使用 =）
	dt.AddColUsingCCL("NewCol", "B = A + 1")

	// 應該保持原始欄位數量（賦值語法被拒絕，不會新增欄位）
	if len(dt.columns) != 1 {
		t.Errorf("Expected 1 column (assignment syntax should be rejected), got %d", len(dt.columns))
	}
}

func TestDataTable_AddColUsingCCL_RejectsNEW(t *testing.T) {
	dt := NewDataTable(
		NewDataList(1, 2, 3).SetName("A"),
	)

	// AddColUsingCCL 不應該接受 NEW 函數
	dt.AddColUsingCCL("NewCol", "NEW('test')")

	// 應該保持原始欄位數量（NEW 函數被拒絕，不會新增欄位）
	if len(dt.columns) != 1 {
		t.Errorf("Expected 1 column (NEW function should be rejected), got %d", len(dt.columns))
	}
}

func TestDataTable_EditColByIndexUsingCCL_RejectsAssignment(t *testing.T) {
	dt := NewDataTable(
		NewDataList(1, 2, 3).SetName("A"),
	)

	originalData := dt.GetColByNumber(0).Data()

	// EditColByIndexUsingCCL 不應該接受賦值語法
	dt.EditColByIndexUsingCCL("A", "B = A + 1")

	// 原始資料應保持不變
	newData := dt.GetColByNumber(0).Data()
	for i, v := range originalData {
		if newData[i] != v {
			t.Errorf("Row %d: expected %v (unchanged), got %v", i, v, newData[i])
		}
	}
}

func TestDataTable_EditColByNameUsingCCL_RejectsAssignment(t *testing.T) {
	dt := NewDataTable(
		NewDataList(1, 2, 3).SetName("A"),
	)

	originalData := dt.GetColByName("A").Data()

	// EditColByNameUsingCCL 不應該接受賦值語法
	dt.EditColByNameUsingCCL("A", "A = A + 1")

	// 原始資料應保持不變
	newData := dt.GetColByName("A").Data()
	for i, v := range originalData {
		if newData[i] != v {
			t.Errorf("Row %d: expected %v (unchanged), got %v", i, v, newData[i])
		}
	}
}

func TestDataTable_ExecuteCCL_AggregateTotal(t *testing.T) {
	createDT := func() *DataTable {
		dt := NewDataTable()
		dt.AppendCols(
			NewDataList(1.0, 2.0, 3.0).SetName("A"),
			NewDataList(4.0, 5.0, 6.0).SetName("B"),
		)
		return dt
	}

	// SUM(@) should sum all numeric values in the table: 1+2+3+4+5+6 = 21
	t.Run("SUM", func(t *testing.T) {
		dt := createDT()
		dt.ExecuteCCL("NEW('total_sum') = SUM(@)")
		colTotalSum := dt.GetColByName("total_sum")
		if colTotalSum == nil {
			t.Fatal("Column total_sum not found")
		}
		for i := 0; i < 3; i++ {
			if colTotalSum.Get(i) != 21.0 {
				t.Errorf("Expected 21.0 at row %d, got %v", i, colTotalSum.Get(i))
			}
		}
	})

	// COUNT(@) should count all cells: 3 rows * 2 columns = 6
	t.Run("COUNT", func(t *testing.T) {
		dt := createDT()
		dt.ExecuteCCL("NEW('total_count') = COUNT(@)")
		colTotalCount := dt.GetColByName("total_count")
		if colTotalCount == nil {
			t.Fatal("Column total_count not found")
		}
		for i := 0; i < 3; i++ {
			if colTotalCount.Get(i) != 6.0 {
				t.Errorf("Expected 6.0 at row %d, got %v", i, colTotalCount.Get(i))
			}
		}
	})

	// AVG(@) should be 21/6 = 3.5
	t.Run("AVG", func(t *testing.T) {
		dt := createDT()
		dt.ExecuteCCL("NEW('total_avg') = AVG(@)")
		colTotalAvg := dt.GetColByName("total_avg")
		if colTotalAvg == nil {
			t.Fatal("Column total_avg not found")
		}
		for i := 0; i < 3; i++ {
			if colTotalAvg.Get(i) != 3.5 {
				t.Errorf("Expected 3.5 at row %d, got %v", i, colTotalAvg.Get(i))
			}
		}
	})

	// MAX(@) should be 6.0
	t.Run("MAX", func(t *testing.T) {
		dt := createDT()
		dt.ExecuteCCL("NEW('total_max') = MAX(@)")
		colTotalMax := dt.GetColByName("total_max")
		if colTotalMax == nil {
			t.Fatal("Column total_max not found")
		}
		for i := 0; i < 3; i++ {
			if colTotalMax.Get(i) != 6.0 {
				t.Errorf("Expected 6.0 at row %d, got %v", i, colTotalMax.Get(i))
			}
		}
	})

	// MIN(@) should be 1.0
	t.Run("MIN", func(t *testing.T) {
		dt := createDT()
		dt.ExecuteCCL("NEW('total_min') = MIN(@)")
		colTotalMin := dt.GetColByName("total_min")
		if colTotalMin == nil {
			t.Fatal("Column total_min not found")
		}
		for i := 0; i < 3; i++ {
			if colTotalMin.Get(i) != 1.0 {
				t.Errorf("Expected 1.0 at row %d, got %v", i, colTotalMin.Get(i))
			}
		}
	})

	// Test multiple aggregates in one ExecuteCCL call
	t.Run("MultipleAggregates", func(t *testing.T) {
		dt := createDT()
		// In one call, NEW('S') = SUM(@) and NEW('C') = COUNT(@)
		// Both should see only A and B because of snapshotting.
		dt.ExecuteCCL("NEW('S') = SUM(@); NEW('C') = COUNT(@)")

		colS := dt.GetColByName("S")
		colC := dt.GetColByName("C")

		if colS == nil || colC == nil {
			t.Fatal("Columns S or C not found")
		}

		for i := 0; i < 3; i++ {
			if colS.Get(i) != 21.0 {
				t.Errorf("Expected S=21.0 at row %d, got %v", i, colS.Get(i))
			}
			if colC.Get(i) != 6.0 {
				t.Errorf("Expected C=6.0 at row %d, got %v", i, colC.Get(i))
			}
		}
	})
}
