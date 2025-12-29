package insyra

import (
	"testing"
)

func TestSetRowNames(t *testing.T) {
	// Create a DataTable with some data
	dt := NewDataTable()
	col1 := NewDataList()
	col1.Append("A1", "A2", "A3")
	col2 := NewDataList()
	col2.Append("B1", "B2", "B3")
	dt.AppendCols(col1, col2)

	// Test case: Set row names
	rowNames := []string{"Row1", "Row2", "Row3"}
	result := dt.SetRowNames(rowNames)
	if result.getMaxColLength() != 3 {
		t.Errorf("Expected 3 rows, got %d", result.getMaxColLength())
	}
	name0, _ := result.GetRowNameByIndex(0)
	name1, _ := result.GetRowNameByIndex(1)
	name2, _ := result.GetRowNameByIndex(2)
	if name0 != "Row1" || name1 != "Row2" || name2 != "Row3" {
		t.Errorf("Expected row names Row1, Row2, Row3, got %s, %s, %s", name0, name1, name2)
	}

	// Test case: Fewer row names than rows
	dt2 := NewDataTable()
	col3 := NewDataList()
	col3.Append("X1", "X2", "X3", "X4")
	col4 := NewDataList()
	col4.Append("Y1", "Y2", "Y3", "Y4")
	dt2.AppendCols(col3, col4)
	rowNames2 := []string{"R1", "R2"}
	result2 := dt2.SetRowNames(rowNames2)
	name20, _ := result2.GetRowNameByIndex(0)
	name21, _ := result2.GetRowNameByIndex(1)
	name22, _ := result2.GetRowNameByIndex(2)
	name23, _ := result2.GetRowNameByIndex(3)
	if name20 != "R1" || name21 != "R2" || name22 != "" || name23 != "" {
		t.Errorf("Expected row names R1, R2, '', '', got %s, %s, %s, %s", name20, name21, name22, name23)
	}

	// Test case: More row names than rows (ignore excess)
	dt3 := NewDataTable()
	col5 := NewDataList()
	col5.Append("P1", "P2")
	col6 := NewDataList()
	col6.Append("Q1", "Q2")
	dt3.AppendCols(col5, col6)
	rowNames3 := []string{"First", "Second", "Third", "Fourth"}
	result3 := dt3.SetRowNames(rowNames3)
	name30, _ := result3.GetRowNameByIndex(0)
	name31, _ := result3.GetRowNameByIndex(1)
	if name30 != "First" || name31 != "Second" {
		t.Errorf("Expected row names First, Second, got %s, %s", name30, name31)
	}

	// Test case: Empty row names
	dt4 := NewDataTable()
	col7 := NewDataList()
	col7.Append("M1", "M2")
	col8 := NewDataList()
	col8.Append("N1", "N2")
	dt4.AppendCols(col7, col8)
	rowNames4 := []string{}
	result4 := dt4.SetRowNames(rowNames4)
	name40, _ := result4.GetRowNameByIndex(0)
	name41, _ := result4.GetRowNameByIndex(1)
	if name40 != "" || name41 != "" {
		t.Errorf("Expected row names '', '', got %s, %s", name40, name41)
	}

	// Test case: Row names with empty strings
	dt5 := NewDataTable()
	col9 := NewDataList()
	col9.Append("S1", "S2")
	col10 := NewDataList()
	col10.Append("T1", "T2")
	dt5.AppendCols(col9, col10)
	rowNames5 := []string{"", "Valid"}
	result5 := dt5.SetRowNames(rowNames5)
	name50, _ := result5.GetRowNameByIndex(0)
	name51, _ := result5.GetRowNameByIndex(1)
	if name50 != "" || name51 != "Valid" {
		t.Errorf("Expected row names '', Valid, got %s, %s", name50, name51)
	}
}

func TestGetRowNameByIndex(t *testing.T) {
	dt := NewDataTable()
	col1 := NewDataList()
	col1.Append("A1", "A2", "A3")
	col2 := NewDataList()
	col2.Append("B1", "B2", "B3")
	dt.AppendCols(col1, col2)

	// Set row names
	dt.SetRowNames([]string{"First", "Second", "Third"})

	// Test getting existing row names
	name0, exists0 := dt.GetRowNameByIndex(0)
	if !exists0 || name0 != "First" {
		t.Errorf("Expected ('First', true), got ('%s', %v)", name0, exists0)
	}

	name1, exists1 := dt.GetRowNameByIndex(1)
	if !exists1 || name1 != "Second" {
		t.Errorf("Expected ('Second', true), got ('%s', %v)", name1, exists1)
	}

	// Test getting row without a name (should return empty string and false)
	dt2 := NewDataTable()
	col3 := NewDataList()
	col3.Append("X1", "X2")
	dt2.AppendCols(col3)

	name, exists := dt2.GetRowNameByIndex(0)
	if exists || name != "" {
		t.Errorf("Expected ('', false), got ('%s', %v)", name, exists)
	}
}

func TestGetRowIndexByName(t *testing.T) {
	dt := NewDataTable()
	col1 := NewDataList()
	col1.Append("A1", "A2", "A3")
	col2 := NewDataList()
	col2.Append("B1", "B2", "B3")
	dt.AppendCols(col1, col2)

	// Set row names
	dt.SetRowNames([]string{"Alpha", "Beta", "Gamma"})

	// Test getting existing row indices
	idx0, exists0 := dt.GetRowIndexByName("Alpha")
	if !exists0 || idx0 != 0 {
		t.Errorf("Expected (0, true), got (%d, %v)", idx0, exists0)
	}

	idx1, exists1 := dt.GetRowIndexByName("Beta")
	if !exists1 || idx1 != 1 {
		t.Errorf("Expected (1, true), got (%d, %v)", idx1, exists1)
	}

	idx2, exists2 := dt.GetRowIndexByName("Gamma")
	if !exists2 || idx2 != 2 {
		t.Errorf("Expected (2, true), got (%d, %v)", idx2, exists2)
	}

	// Test getting non-existent row name (should return -1 and false)
	idx, exists := dt.GetRowIndexByName("NonExistent")
	if exists || idx != -1 {
		t.Errorf("Expected (-1, false), got (%d, %v)", idx, exists)
	}
}
