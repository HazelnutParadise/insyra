package insyra

import (
	"testing"
	"time"
)

func IDataTableTest(dt IDataTable) {
	return
}

func TestIDataTable(t *testing.T) {
	defer func() {
		r := recover()
		if r != nil {
			t.Errorf("IDataTableTest() panicked with: %v", r)
		}
	}()
	dt := NewDataTable()
	IDataTableTest(dt)
}

func TestNewDataTable(t *testing.T) {
	dt := NewDataTable()
	if dt == nil {
		t.Errorf("NewDataTable() returned nil")
	}
}

func TestDataTable_AppendColumns(t *testing.T) {
	dt := NewDataTable()
	dl := NewDataList(1, 2, 3)
	dt.AppendColumns(dl)
	if len(dt.columns) != 1 {
		t.Errorf("AppendColumns() did not add the column correctly")
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
	dt.AppendRowsByIndex(map[string]interface{}{"A": 1, "B": 2, "C": 3})
	dt.Show()
	if r, c := dt.Size(); r != 1 || c != 3 {
		t.Errorf("AppendRowsByIndex() did not add the row correctly")
	}
}

func TestDataTable_AppendRowsByName(t *testing.T) {
	dt := NewDataTable()
	dt.AppendRowsByName(map[string]interface{}{"first": 1, "second": 2, "third": 3})
	dt.Show()
	if r, c := dt.Size(); r != 1 || c != 3 {
		t.Errorf("AppendRowsByName() did not add the row correctly")
	}
}

func TestDataTable_GetElement(t *testing.T) {
	dt := NewDataTable()
	dt.AppendRowsByName(map[string]interface{}{"first": 1, "second": 2, "third": 3})
	dt.Show()
	if dt.GetElement(0, "A") != 1 {
		t.Errorf("GetElement() did not return the correct element")
	}
}

func TestDataTable_GetElementByNumberIndex(t *testing.T) {
	dt := NewDataTable()
	dt.AppendRowsByName(map[string]interface{}{"first": 1, "second": 2, "third": 3})
	dt.Show()
	if dt.GetElementByNumberIndex(0, 0) != 1 {
		t.Errorf("GetElementByNumberIndex() did not return the correct element")
	}
}

func TestDataTable_GetColumn(t *testing.T) {
	dt := NewDataTable()
	dt.AppendRowsByIndex(map[string]interface{}{"A": 1, "B": 2, "C": 3})
	dt.Show()
	if dt.GetColumn("A").Data()[0] != 1 {
		t.Errorf("GetColumn() did not return the correct column")
	}
}

func TestDataTable_GetColumnByNumber(t *testing.T) {
	dt := NewDataTable()
	dt.AppendRowsByIndex(map[string]interface{}{"A": 1, "B": 2, "C": 3})
	dt.Show()
	if dt.GetColumnByNumber(0).Data()[0] != 1 {
		t.Errorf("GetColumnByNumber() did not return the correct column")
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
