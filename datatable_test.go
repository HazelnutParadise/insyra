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
