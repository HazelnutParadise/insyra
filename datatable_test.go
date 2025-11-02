package insyra

import (
	"fmt"
	"testing"
	"time"
)

func IDataTableTest(dt IDataTable) bool {
	return true
}

func TestIDataTable(t *testing.T) {
	defer func() {
		r := recover()
		if r != nil {
			t.Errorf("IDataTableTest() panicked with: %v", r)
		}
	}()
	dt := NewDataTable()
	if !IDataTableTest(dt) {
		t.Errorf("IDataTableTest() failed")
	}
}

func TestNewDataTable(t *testing.T) {
	dt := NewDataTable()
	if dt == nil {
		t.Errorf("NewDataTable() returned nil")
	}
}

func TestDataTable_AppendCols(t *testing.T) {
	dt := NewDataTable()
	dl := NewDataList(1, 2, 3)
	dt.AppendCols(dl)
	if len(dt.columns) != 1 {
		t.Errorf("AppendCols() did not add the column correctly")
	}
}

func TestDataTable_AddColUsingCCL(t *testing.T) {
	dt := NewDataTable()
	dlA := NewDataList(1, 2, 3)
	dlB := NewDataList(2, 3, 4)
	dlC := NewDataList(3, 4, 5)
	dt.AppendCols(dlA, dlB, dlC)
	dt.AddColUsingCCL("new_col", "A + B + C")
	if len(dt.columns) != 4 {
		t.Errorf("AddColUsingCCL() did not add the column correctly")
	}
	if dt.GetCol("D").Data()[0] != 6.0 {
		t.Errorf("AddColUsingCCL() did not compute the column correctly")
	}
}

func TestDataTable_AppendRowsFromDataList(t *testing.T) {
	dt := NewDataTable()
	dl := NewDataList(1, 2, 3)
	dt.AppendRowsFromDataList(dl)
	if r, c := dt.Size(); r != 1 || c != 3 {
		t.Errorf("AppendRowsFromDataList() did not add the row correctly")
	}
}

func TestDataTable_AppendRowsByIndex(t *testing.T) {
	dt := NewDataTable()
	dt.AppendRowsByColIndex(map[string]any{"a": 1, "B": 2, "C": 3})
	dt.Show()
	if r, c := dt.Size(); r != 1 || c != 3 {
		t.Errorf("AppendRowsByIndex() did not add the row correctly")
	}
}

func TestDataTable_AppendRowsByName(t *testing.T) {
	dt := NewDataTable()
	dt.AppendRowsByColName(map[string]any{"first": 1, "second": 2, "third": 3})
	dt.Show()
	if r, c := dt.Size(); r != 1 || c != 3 {
		t.Errorf("AppendRowsByName() did not add the row correctly")
	}
}

func TestDataTable_GetElement(t *testing.T) {
	dt := NewDataTable()
	dt.AppendRowsByColIndex(map[string]any{"A": 1, "B": 2, "C": 3})
	dt.Show()
	if dt.GetElement(0, "A") != 1 {
		t.Errorf("GetElement() did not return the correct element")
	}
}

func TestDataTable_GetElementByNumberIndex(t *testing.T) {
	dt := NewDataTable()
	dt.AppendRowsByColIndex(map[string]any{"A": 1, "B": 2, "C": 3})
	dt.Show()
	if dt.GetElementByNumberIndex(0, 0) != 1 {
		t.Errorf("GetElementByNumberIndex() did not return the correct element")
	}
}

func TestDataTable_GetCol(t *testing.T) {
	dt := NewDataTable()
	dt.AppendRowsByColIndex(map[string]any{"A": 1, "B": 2, "C": 3})
	dt.Show()
	if dt.GetCol("A").Data()[0] != 1 {
		t.Errorf("GetCol() did not return the correct column")
	}
}

func TestDataTable_GetColByNumber(t *testing.T) {
	dt := NewDataTable()
	dt.AppendRowsByColIndex(map[string]any{"A": 1, "B": 2, "C": 3})
	dt.Show()
	if dt.GetColByNumber(0).Data()[0] != 1 {
		t.Errorf("GetColByNumber() did not return the correct column")
	}
}

func TestDataTable_GetRow(t *testing.T) {
	dt := NewDataTable()
	dl := NewDataList(1, 2, 3)
	dt.AppendCols(dl, dl, dl)
	dt.Show()
	if dt.GetRow(2).Data()[0] != 3 {
		t.Errorf("GetRow() did not return the correct row")
	}
}

func TestDataTable_UpdateElement(t *testing.T) {
	dt := NewDataTable()
	dt.AppendRowsByColIndex(map[string]any{"A": 1, "B": 2, "C": 3})
	dt.Show()
	dt.UpdateElement(0, "A", 10)
	dt.Show()
	if dt.GetElement(0, "A") != 10 {
		t.Errorf("UpdateElement() did not update the correct element")
	}
}

func TestDataTable_UpdateCol(t *testing.T) {
	dt := NewDataTable()
	dt.AppendRowsByColIndex(map[string]any{"A": 1, "B": 2, "C": 3})
	dt.Show()
	dt.UpdateCol("A", NewDataList(10, 20, 30))
	dt.Show()
	if dt.GetCol("A").Data()[0] != 10 {
		t.Errorf("UpdateCol() did not update the correct column")
	}
}

func TestDataTable_UpdateColByNumber(t *testing.T) {
	dt := NewDataTable()
	dt.AppendRowsByColIndex(map[string]any{"A": 1, "B": 2, "C": 3})
	dt.Show()
	dt.UpdateColByNumber(0, NewDataList(10, 20, 30))
	dt.Show()
	if dt.GetColByNumber(0).Data()[0] != 10 {
		t.Errorf("UpdateColByNumber() did not update the correct column")
	}
}

func TestDataTable_UpdateRow(t *testing.T) {
	dt := NewDataTable()
	dt.AppendRowsByColIndex(map[string]any{"A": 1, "B": 2, "C": 3})
	dt.Show()
	dt.UpdateRow(0, NewDataList(10, 20, 30))
	dt.Show()
	if dt.GetRow(0).Data()[2] != 30 {
		t.Errorf("UpdateRow() did not update the correct row")
	}
}

func TestDataTable_Counter(t *testing.T) {
	dt := NewDataTable()
	dt.AppendRowsByColIndex(map[string]any{"A": 1, "B": 2, "C": 1})
	dt.Show()
	counter := dt.Counter()
	fmt.Println(counter)
	if counter[1] != 2 {
		t.Errorf("Counter() did not return the correct counter")
	}
}

func TestDataTable_GetLastModifiedTimestamp(t *testing.T) {
	dt := NewDataTable()
	dl := NewDataList(1, 2, 3)
	dt.AppendRowsFromDataList(dl)
	if dt.GetLastModifiedTimestamp() != time.Now().Unix() {
		t.Errorf("GetLastModifiedTimestamp() did not return the correct timestamp")
	}
}

func TestDataTable_GetColNameByIndex(t *testing.T) {
	dt := NewDataTable()
	dl1 := NewDataList(1, 2, 3).SetName("ColA")
	dl2 := NewDataList(4, 5, 6)
	dl3 := NewDataList(7, 8, 9)
	dt.AppendCols(dl1, dl2, dl3)
	dt.SetColNameByIndex("B", "ColB")
	dt.SetColNameByIndex("C", "ColC")

	if dt.GetColNameByIndex("A") != "ColA" {
		t.Errorf("GetColNameByIndex(\"A\") = %s; want ColA", dt.GetColNameByIndex("A"))
	}
	if dt.GetColNameByIndex("B") != "ColB" {
		t.Errorf("GetColNameByIndex(\"B\") = %s; want ColB", dt.GetColNameByIndex("B"))
	}
	if dt.GetColNameByIndex("C") != "ColC" {
		t.Errorf("GetColNameByIndex(\"C\") = %s; want ColC", dt.GetColNameByIndex("C"))
	}
	// Test invalid index
	if dt.GetColNameByIndex("Z") != "" {
		t.Errorf("GetColNameByIndex(\"Z\") = %s; want empty string", dt.GetColNameByIndex("Z"))
	}
}

func TestDataTable_GetColIndexByName(t *testing.T) {
	dt := NewDataTable()
	dl1 := NewDataList(1, 2, 3)
	dl2 := NewDataList(4, 5, 6)
	dl3 := NewDataList(7, 8, 9)
	dt.AppendCols(dl1, dl2, dl3)
	dt.SetColNameByIndex("A", "ColA")
	dt.SetColNameByIndex("B", "ColB")
	dt.SetColNameByIndex("C", "ColC")

	if dt.GetColIndexByName("ColA") != "A" {
		t.Errorf("GetColIndexByName(\"ColA\") = %s; want A", dt.GetColIndexByName("ColA"))
	}
	if dt.GetColIndexByName("ColB") != "B" {
		t.Errorf("GetColIndexByName(\"ColB\") = %s; want B", dt.GetColIndexByName("ColB"))
	}
	if dt.GetColIndexByName("ColC") != "C" {
		t.Errorf("GetColIndexByName(\"ColC\") = %s; want C", dt.GetColIndexByName("ColC"))
	}
	// Test invalid name
	if dt.GetColIndexByName("NonExistent") != "" {
		t.Errorf("GetColIndexByName(\"NonExistent\") = %s; want empty string", dt.GetColIndexByName("NonExistent"))
	}
}

func TestDataTable_GetColIndexByNumber(t *testing.T) {
	dt := NewDataTable()
	dl1 := NewDataList(1, 2, 3)
	dl2 := NewDataList(4, 5, 6)
	dl3 := NewDataList(7, 8, 9)
	dt.AppendCols(dl1, dl2, dl3)

	if dt.GetColIndexByNumber(0) != "A" {
		t.Errorf("GetColIndexByNumber(0) = %s; want A", dt.GetColIndexByNumber(0))
	}
	if dt.GetColIndexByNumber(1) != "B" {
		t.Errorf("GetColIndexByNumber(1) = %s; want B", dt.GetColIndexByNumber(1))
	}
	if dt.GetColIndexByNumber(2) != "C" {
		t.Errorf("GetColIndexByNumber(2) = %s; want C", dt.GetColIndexByNumber(2))
	}
	// Test negative index (from end)
	if dt.GetColIndexByNumber(-1) != "C" {
		t.Errorf("GetColIndexByNumber(-1) = %s; want C", dt.GetColIndexByNumber(-1))
	}
	if dt.GetColIndexByNumber(-2) != "B" {
		t.Errorf("GetColIndexByNumber(-2) = %s; want B", dt.GetColIndexByNumber(-2))
	}
	// Test invalid number
	if dt.GetColIndexByNumber(10) != "" {
		t.Errorf("GetColIndexByNumber(10) = %s; want empty string", dt.GetColIndexByNumber(10))
	}
}

func TestDataTable_GetColNumberByName(t *testing.T) {
	dt := NewDataTable()
	dl1 := NewDataList(1, 2, 3)
	dl2 := NewDataList(4, 5, 6)
	dl3 := NewDataList(7, 8, 9)
	dt.AppendCols(dl1, dl2, dl3)
	dt.SetColNameByIndex("A", "ColA")
	dt.SetColNameByIndex("B", "ColB")
	dt.SetColNameByIndex("C", "ColC")

	if dt.GetColNumberByName("ColA") != 0 {
		t.Errorf("GetColNumberByName(\"ColA\") = %d; want 0", dt.GetColNumberByName("ColA"))
	}
	if dt.GetColNumberByName("ColB") != 1 {
		t.Errorf("GetColNumberByName(\"ColB\") = %d; want 1", dt.GetColNumberByName("ColB"))
	}
	if dt.GetColNumberByName("ColC") != 2 {
		t.Errorf("GetColNumberByName(\"ColC\") = %d; want 2", dt.GetColNumberByName("ColC"))
	}
	// Test invalid name
	if dt.GetColNumberByName("NonExistent") != -1 {
		t.Errorf("GetColNumberByName(\"NonExistent\") = %d; want -1", dt.GetColNumberByName("NonExistent"))
	}
}

func TestDataTable_Clone(t *testing.T) {
	// Create original DataTable
	dt := NewDataTable()
	dl1 := NewDataList(1, 2, 3).SetName("Col1")
	dl2 := NewDataList("a", "b", "c").SetName("Col2")
	dt.AppendCols(dl1, dl2)
	dt.SetName("OriginalTable")

	// Clone the DataTable
	clonedDT := dt.Clone()
	dt.AtomicDo(func(dt *DataTable) {
		dt.Show()
	})

	// Check if cloned DataTable is not nil
	if clonedDT == nil {
		t.Errorf("Clone() returned nil")
	}

	// Check if name is copied
	if clonedDT.GetName() != dt.GetName() {
		t.Errorf("Clone() did not copy name correctly: got %s, want %s", clonedDT.GetName(), dt.GetName())
	}

	// Check if columns are cloned (deep copy)
	if len(clonedDT.columns) != len(dt.columns) {
		t.Errorf("Clone() did not copy columns correctly: got %d columns, want %d", len(clonedDT.columns), len(dt.columns))
	}

	// Check if data is independent (modify original and check clone)
	dt.columns[0].data[0] = 999
	if clonedDT.columns[0].data[0] == 999 {
		t.Errorf("Clone() did not create deep copy: clone was affected by original modification")
	}

	// Check if columnIndex is copied
	if len(clonedDT.columnIndex) != len(dt.columnIndex) {
		t.Errorf("Clone() did not copy columnIndex correctly")
	}

	// Check if rowNames is copied
	if len(clonedDT.rowNames) != len(dt.rowNames) {
		t.Errorf("Clone() did not copy rowNames correctly")
	}
}

func TestDataTable_DropRowsContain(t *testing.T) {
	dt := NewDataTable()
	dl1 := NewDataList(1, 2, 3, 4)
	dl2 := NewDataList("a", "b", "c", "d")
	dl3 := NewDataList(1.1, 2.2, 3.3, 4.4)
	dt.AppendCols(dl1, dl2, dl3)

	// Original size should be 4 rows
	numRows, _ := dt.Size()
	if numRows != 4 {
		t.Errorf("Expected 4 rows, got %d", numRows)
	}

	// Drop rows containing 2
	dt.DropRowsContain(2)

	// After dropping, should have 3 rows
	numRows, _ = dt.Size()
	if numRows != 3 {
		t.Errorf("Expected 3 rows after dropping rows containing 2, got %d", numRows)
	}

	// Check that the row with 2 is gone
	found := false
	for i := 0; i < numRows; i++ {
		if dt.GetElement(i, "A") == 2 {
			found = true
			break
		}
	}
	if found {
		t.Errorf("Row containing 2 was not dropped")
	}

	// Test dropping multiple values
	dt2 := NewDataTable()
	dl1_2 := NewDataList(1, 2, 3, 4, 5)
	dl2_2 := NewDataList("x", "y", "z", "w", "v")
	dt2.AppendCols(dl1_2, dl2_2)

	dt2.DropRowsContain(1, 5)

	numRows2, _ := dt2.Size()
	if numRows2 != 3 {
		t.Errorf("Expected 3 rows after dropping rows containing 1 or 5, got %d", numRows2)
	}
}

func TestDataTable_DropColsContain(t *testing.T) {
	dt := NewDataTable()
	dl1 := NewDataList(1, 2, 3)
	dl2 := NewDataList("a", "b", "c")
	dl3 := NewDataList(1.1, 2.2, 3.3)
	dt.AppendCols(dl1, dl2, dl3)

	// Original size should be 3 columns
	_, numCols := dt.Size()
	if numCols != 3 {
		t.Errorf("Expected 3 columns, got %d", numCols)
	}

	// Drop columns containing "a"
	dt.DropColsContain("a")

	// After dropping, should have 2 columns
	_, numCols = dt.Size()
	if numCols != 2 {
		t.Errorf("Expected 2 columns after dropping columns containing 'a', got %d", numCols)
	}

	// Check that the column with "a" is gone
	colNames := dt.ColNames()
	found := false
	for _, name := range colNames {
		if name == "B" { // Assuming columns are A, B, C and B contained "a"
			found = true
			break
		}
	}
	if found {
		t.Errorf("Column containing 'a' was not dropped")
	}

	// Test dropping multiple values
	dt2 := NewDataTable()
	dl1_2 := NewDataList(1, 2, 3)
	dl2_2 := NewDataList("x", "y", "z")
	dl3_2 := NewDataList(1.1, 2.2, 3.3)
	dl4_2 := NewDataList(10, 20, 30)
	dt2.AppendCols(dl1_2, dl2_2, dl3_2, dl4_2)

	dt2.DropColsContain(1, 10)

	_, numCols2 := dt2.Size()
	if numCols2 != 2 {
		t.Errorf("Expected 2 columns after dropping columns containing 1 or 10, got %d", numCols2)
	}
}

func TestDataTable_To2DSlice(t *testing.T) {
	dt := NewDataTable()
	dl1 := NewDataList(1, 2, 3)
	dl2 := NewDataList(4, 5)
	dl3 := NewDataList(6, 7, 8, 9)
	dt.AppendCols(dl1, dl2, dl3)

	slice := dt.To2DSlice()

	// Check dimensions
	if len(slice) != 4 { // max length is 4
		t.Errorf("Expected 4 rows, got %d", len(slice))
	}
	if len(slice[0]) != 3 { // 3 columns
		t.Errorf("Expected 3 columns, got %d", len(slice[0]))
	}

	// Check values
	if slice[0][0] != 1 || slice[0][1] != 4 || slice[0][2] != 6 {
		t.Errorf("Row 0 values incorrect: %v", slice[0])
	}
	if slice[1][0] != 2 || slice[1][1] != 5 || slice[1][2] != 7 {
		t.Errorf("Row 1 values incorrect: %v", slice[1])
	}
	if slice[2][0] != 3 || slice[2][1] != nil || slice[2][2] != 8 {
		t.Errorf("Row 2 values incorrect: %v", slice[2])
	}
	if slice[3][0] != nil || slice[3][1] != nil || slice[3][2] != 9 {
		t.Errorf("Row 3 values incorrect: %v", slice[3])
	}
}
