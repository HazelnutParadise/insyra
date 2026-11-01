package insyra

import (
	"reflect"
	"strings"
	"testing"
	"time"
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
	expected := []any{int64(101), int64(202), int64(303), int64(404)}
	colB := dt.GetColByName("B")
	if colB == nil {
		t.Fatal("Column B not found")
		return
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
	expected := []any{int64(2), int64(4), int64(6), int64(8)}
	colA := dt.GetColByNumber(0)
	if colA == nil {
		t.Fatal("Column A (index 0) not found")
		return
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
	expected := []any{int64(11), int64(22), int64(33), int64(44)}
	colSum := dt.GetColByName("Sum")
	if colSum == nil {
		t.Fatal("Column Sum not found")
		return
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
		NewDataList(1, 2, 3, 4).SetName("Ai"),
		NewDataList(10, 20, 30, 40).SetName("B"),
		NewDataList(100, 200, 300, 400).SetName("C"),
	)

	// 執行多行 CCL
	dt.ExecuteCCL(`
		A = ['Ai'] * 10
		NEW('D') = A + B + C
	`)

	// 驗證 A 列被修改
	expectedA := []any{int64(10), int64(20), int64(30), int64(40)}
	colA := dt.GetColByName("Ai")
	for i, v := range colA.Data() {
		if v != expectedA[i] {
			t.Errorf("Column A, Row %d: expected %v, got %v", i, expectedA[i], v)
		}
	}

	// 驗證 D 列被創建 (D = A + B + C, A已經是修改後的值)
	expectedD := []any{int64(120), int64(240), int64(360), int64(480)}
	colD := dt.GetColByName("D")
	if colD == nil {
		t.Fatal("Column D not found")
		return
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
	expectedA := []any{int64(2), int64(3), int64(4)}
	colA := dt.GetColByNumber(0)
	for i, v := range colA.Data() {
		if v != expectedA[i] {
			t.Errorf("Column A, Row %d: expected %v, got %v", i, expectedA[i], v)
		}
	}

	// 驗證 Z 列被創建 (Z = A * B, A已經是修改後的值, B是第二列)
	expectedZ := []any{int64(10), int64(18), int64(28)}
	colZ := dt.GetColByName("Z")
	if colZ == nil {
		t.Fatal("Column Z not found")
		return
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
	expected := []any{int64(2), int64(6), int64(12)}
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
	expected := []any{int64(10), int64(20), int64(30), int64(40)}
	colA := dt.GetColByNumber(0)
	if colA == nil {
		t.Fatal("Column A (index 0) not found")
		return
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
	expected := []any{int64(101), int64(202), int64(303)}
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
	expected := []any{int64(10), int64(20), int64(30), int64(40)}
	colPrice := dt.GetColByName("price")
	if colPrice == nil {
		t.Fatal("Column 'price' not found")
		return
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
		case int64:
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
			return
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
			return
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
			return
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
			return
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
			return
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
		// Statements are executed sequentially; the second statement sees the S column added by the first.
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
			// Because S has been added, COUNT(@) counts 3 rows * 3 cols = 9
			if colC.Get(i) != 9.0 {
				t.Errorf("Expected C=9.0 at row %d, got %v", i, colC.Get(i))
			}
		}
	})
}

func TestDataTable_ExecuteCCL_Transposition(t *testing.T) {
	// Create a table with 3 rows and 4 columns
	// Row 0: 1, 2, 3, 4
	// Row 1: 5, 6, 7, 8
	// Row 2: 9, 10, 11, 12
	dt := NewDataTable(
		NewDataList(1, 5, 9).SetName("A"),
		NewDataList(2, 6, 10).SetName("B"),
		NewDataList(3, 7, 11).SetName("C"),
		NewDataList(4, 8, 12).SetName("D"),
	)

	// Transpose Row 0 into new column 'n'
	// Row 0 is [1, 2, 3, 4]
	// Column 'n' should be [1, 2, 3, 4]
	dt.ExecuteCCL("NEW('n') = @.0")

	// Check column 'n'
	colN := dt.GetColByName("n")
	if colN == nil {
		t.Fatal("Column 'n' not found")
		return
	}

	data := colN.Data()
	if len(data) != 4 {
		t.Errorf("Expected column 'n' length 4, got %d", len(data))
	}

	expected := []any{1, 2, 3, 4}
	for i, v := range expected {
		if i < len(data) {
			if data[i] != v {
				t.Errorf("Row %d: expected %v, got %v", i, v, data[i])
			}
		}
	}

	// Check if other columns are padded
	colA := dt.GetColByName("A")
	if len(colA.Data()) != 4 {
		t.Errorf("Expected column 'A' length 4 (padded), got %d", len(colA.Data()))
	}
	if colA.Data()[3] != nil {
		t.Errorf("Expected column 'A' row 3 to be nil, got %v", colA.Data()[3])
	}
}

// Regression test for issue #184: malformed CCL (especially input containing
// characters the tokenizer has no token for, like '$') used to make
// AddColUsingCCL/ExecuteCCL spin forever instead of returning. The call must
// return promptly, leave the table unchanged, and surface the failure via Err().
func TestDataTable_CCL_MalformedFormulaReturnsAndSetsErr(t *testing.T) {
	cases := []struct {
		name string
		run  func(dt *DataTable)
	}{
		{"AddColUsingCCL unknown char", func(dt *DataTable) { dt.AddColUsingCCL("X", "this is junk @#$") }},
		{"AddColUsingCCL trailing tokens", func(dt *DataTable) { dt.AddColUsingCCL("X", "5 * 3 garbage") }},
		{"ExecuteCCL unknown char", func(dt *DataTable) { dt.ExecuteCCL("NEW('X') = A + $") }},
		{"EditColByNameUsingCCL unknown char", func(dt *DataTable) { dt.EditColByNameUsingCCL("A", "~~~") }},
		// Over-deep nesting used to fatally overflow the stack (unrecoverable,
		// kills the process) instead of erroring. See change bound-ccl-compile-recursion.
		// (Long operator chains are no longer depth-capped — see
		// TestDataTable_CCL_LongChainFormula.)
		{"AddColUsingCCL over-deep nesting", func(dt *DataTable) {
			dt.AddColUsingCCL("X", strings.Repeat("(", 10001)+"1"+strings.Repeat(")", 10001))
		}},
		{"ExecuteCCL over-deep nesting", func(dt *DataTable) {
			dt.ExecuteCCL("NEW('X') = " + strings.Repeat("(", 10001) + "1" + strings.Repeat(")", 10001))
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dt := NewDataTable(
				NewDataList(1.0, 2.0, 3.0).SetName("A"),
				NewDataList(10.0, 20.0, 30.0).SetName("B"),
			)

			done := make(chan struct{})
			go func() {
				tc.run(dt)
				close(done)
			}()
			select {
			case <-done:
			case <-time.After(5 * time.Second):
				t.Fatal("CCL call did not return within 5s — possible tokenizer/parser non-termination (issue #184 regression)")
			}

			if dt.Err() == nil {
				t.Error("expected dt.Err() to be set after malformed CCL, got nil")
			}
			numRow, numCol := dt.Size()
			if numCol != 2 || numRow != 3 {
				t.Errorf("expected table to stay 3x2 after failed CCL, got %dx%d", numRow, numCol)
			}
			if a := dt.GetColByName("A").Data(); a[0] != 1.0 || a[1] != 2.0 || a[2] != 3.0 {
				t.Errorf("expected column A unchanged, got %v", a)
			}
		})
	}
}

// TestDataTable_CCL_LongChainFormula pins flatten-ccl-operator-chains end to
// end: long left-associative chains (previously rejected as too complex, and
// before that a fatal stack overflow) now produce correct column values.
func TestDataTable_CCL_LongChainFormula(t *testing.T) {
	const terms = 20000
	dt := NewDataTable(
		NewDataList(1.0, 2.0, 3.0).SetName("A"),
	)

	dt.ClearErr().AddColUsingCCL("sum", "A"+strings.Repeat("+1", terms))
	if err := dt.Err(); err != nil {
		t.Fatalf("AddColUsingCCL long chain unexpectedly failed: %v", err)
	}
	col := dt.GetColByName("sum")
	if col == nil {
		t.Fatal("column 'sum' not found")
		return
	}
	want := []float64{1.0 + terms, 2.0 + terms, 3.0 + terms}
	for i, v := range col.Data() {
		if v != want[i] {
			t.Errorf("row %d: got %v, want %v", i, v, want[i])
		}
	}

	dt2 := NewDataTable(NewDataList(10.0).SetName("A"))
	dt2.ClearErr().ExecuteCCL("NEW('X') = 1" + strings.Repeat("+1", terms))
	if err := dt2.Err(); err != nil {
		t.Fatalf("ExecuteCCL long chain unexpectedly failed: %v", err)
	}
	colX := dt2.GetColByName("X")
	if colX == nil {
		t.Fatal("column 'X' not found")
		return
	}
	if got := colX.Data()[0]; got != int64(terms+1) {
		t.Errorf("ExecuteCCL long chain: got %v (%T), want int64 %v", got, got, terms+1)
	}
}

// b8Table is a table whose B column contains a zero, so A / B is the natural
// way to ask whether an operand was evaluated.
func b8Table() *DataTable {
	return NewDataTable(
		NewDataList(10, 20, 30).SetName("price"),
		NewDataList(1, 0, 3).SetName("qty"),
	)
}

// CCL-12, seen from the public API: the guard must run before what it guards.
func TestCCLShortCircuitThroughAddCol(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)

	for _, tc := range []struct {
		expr string
		want []any
	}{
		{"B != 0 && A / B > 1", []any{true, false, true}},
		{"CASE(B != 0, A / B, nil)", []any{10.0, nil, 10.0}},
	} {
		dt := b8Table()
		dt.AddColUsingCCL("r", tc.expr)
		if err := dt.PopErr(); err != nil {
			t.Errorf("%s: %v", tc.expr, err)
			continue
		}
		got := dt.GetColByName("r").Data()
		for i := range tc.want {
			if got[i] != tc.want[i] {
				t.Errorf("%s = %v, want %v", tc.expr, got, tc.want)
				break
			}
		}
	}
}

// CCL-23 / CCL-39: a range and '@' stand for a row, so a column built from one
// holds that row's values. `A:B` used to fill every cell with an internal
// ccl.ColumnRange struct, and `LAG(@, 1)` with the whole flattened table.
func TestCCLRowShapedValues(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)

	dt := b8Table()
	dt.AddColUsingCCL("r", "A:B")
	if err := dt.PopErr(); err != nil {
		t.Fatalf("A:B: %v", err)
	}
	got := dt.GetColByName("r").Data()
	want := [][]any{{10, 1}, {20, 0}, {30, 3}}
	for i := range want {
		if !reflect.DeepEqual(got[i], []any(want[i])) {
			t.Fatalf("A:B = %v, want %v", got, want)
		}
	}

	dt = b8Table()
	dt.AddColUsingCCL("prev", "LAG(@, 1)")
	if err := dt.PopErr(); err != nil {
		t.Fatalf("LAG(@, 1): %v", err)
	}
	prev := dt.GetColByName("prev").Data()
	if prev[0] != nil {
		t.Errorf("LAG(@, 1): first row = %v, want nil", prev[0])
	}
	if !reflect.DeepEqual(prev[1], []any{10, 1}) {
		t.Errorf("LAG(@, 1): second row = %v, want [10 1]", prev[1])
	}
}

// The readings that consume a range keep the values they always had. This is a
// regression guard: making a bare `A:B` mean the current row briefly made
// `SUM(A.(0:1))` count the same slice once per row.
func TestCCLRangeConsumersUnchanged(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)

	dt := NewDataTable(
		NewDataList(1, 2, 3, 4, 5, 6).SetName("a"),
		NewDataList(10, 20, 30, 40, 50, 60).SetName("b"),
		NewDataList(100, 200, 300, 400, 500, 600).SetName("c"),
	)
	for _, tc := range []struct {
		expr string
		want any
	}{
		{"SUM(A:C)", int64(2331)},
		{"SUM((A:C).(2:5))", int64(1998)},
		{"SUM(A.(0:1))", int64(3)},
		{"COUNT((A:C).(2:5))", 12.0},
	} {
		table := dt.Clone()
		table.AddColUsingCCL("r", tc.expr)
		if err := table.PopErr(); err != nil {
			t.Errorf("%s: %v", tc.expr, err)
			continue
		}
		for _, v := range table.GetColByName("r").Data() {
			if v != tc.want {
				t.Errorf("%s = %v, want %v in every row", tc.expr, v, tc.want)
				break
			}
		}
	}
}

// A row range names rows but not of what, and a sequence function that does
// arithmetic cannot take a row.
func TestCCLRowShapedValuesThatStayErrors(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)

	for _, expr := range []string{"1:2", "CUMSUM(@)", "ROLLING_SUM(A:B, 2)"} {
		dt := b8Table()
		dt.AddColUsingCCL("r", expr)
		if err := dt.PopErr(); err == nil {
			t.Errorf("%s was accepted; column = %v", expr, colData(dt, "r"))
		}
	}
}

// CCL-11: Go's "<nil>" placeholder must not become data.
func TestCCLNilConcatThroughAddCol(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)

	dt := NewDataTable(NewDataList("a", nil, "c").SetName("s"))
	dt.AddColUsingCCL("r", "A & '!'")
	if err := dt.PopErr(); err != nil {
		t.Fatal(err)
	}
	got := dt.GetColByName("r").Data()
	want := []any{"a!", "!", "c!"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

// CCL-13 / CCL-25: a typo must not compile into a different question.
func TestCCLMalformedExpressionsAreRejected(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)

	for _, expr := range []string{
		"SUM(A B)",
		"IF(A > 15, 1, 0,)",
		"A.(1.7)",
		"ROLLING_MEAN(A, 2.9)",
		"AND()",
		"AND('abc', true)",
		"'hello' > 5",
	} {
		dt := b8Table()
		dt.AddColUsingCCL("r", expr)
		if err := dt.PopErr(); err == nil {
			t.Errorf("%s was accepted; column = %v", expr, colData(dt, "r"))
		}
	}
}

// colData returns a column's values, or nil when the column is absent, so a
// failure message never panics on its way out.
func colData(dt *DataTable, name string) []any {
	if col := dt.GetColByName(name); col != nil {
		return col.Data()
	}
	return nil
}

// CCL-14: & now binds looser than +, so a sum can be concatenated.
func TestCCLConcatPrecedenceThroughAddCol(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)

	dt := b8Table()
	dt.AddColUsingCCL("r", "'total: ' & A + B")
	if err := dt.PopErr(); err != nil {
		t.Fatal(err)
	}
	got := dt.GetColByName("r").Data()
	want := []any{"total: 11", "total: 20", "total: 33"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

// CCL-9: a filter on text columns returned nothing, because both directions
// of a string comparison were false.
func TestCCLStringOrderingThroughFilter(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)

	dt := NewDataTable(NewDataList("banana", "apple", "cherry").SetName("fruit"))
	dt.AddColUsingCCL("before_c", "A < 'c'")
	if err := dt.PopErr(); err != nil {
		t.Fatal(err)
	}
	got := dt.GetColByName("before_c").Data()
	want := []any{true, true, false}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

// The rejection message should say what to do, not name a Go type.
func TestCCLRowRangeErrorSaysWhatToDo(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)

	dt := b8Table()
	dt.AddColUsingCCL("r", "1:2")
	err := dt.PopErr()
	if err == nil {
		t.Fatal("a bare row range was accepted")
		return
	}
	if !strings.Contains(err.Error(), "attach it to a column") {
		t.Errorf("error should say how to write it instead: %v", err)
	}
}
