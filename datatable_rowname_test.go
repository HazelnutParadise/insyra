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
	if result.GetRowNameByIndex(0) != "Row1" || result.GetRowNameByIndex(1) != "Row2" || result.GetRowNameByIndex(2) != "Row3" {
		t.Errorf("Expected row names Row1, Row2, Row3, got %s, %s, %s", result.GetRowNameByIndex(0), result.GetRowNameByIndex(1), result.GetRowNameByIndex(2))
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
	if result2.GetRowNameByIndex(0) != "R1" || result2.GetRowNameByIndex(1) != "R2" || result2.GetRowNameByIndex(2) != "" || result2.GetRowNameByIndex(3) != "" {
		t.Errorf("Expected row names R1, R2, '', '', got %s, %s, %s, %s", result2.GetRowNameByIndex(0), result2.GetRowNameByIndex(1), result2.GetRowNameByIndex(2), result2.GetRowNameByIndex(3))
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
	if result3.GetRowNameByIndex(0) != "First" || result3.GetRowNameByIndex(1) != "Second" {
		t.Errorf("Expected row names First, Second, got %s, %s", result3.GetRowNameByIndex(0), result3.GetRowNameByIndex(1))
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
	if result4.GetRowNameByIndex(0) != "" || result4.GetRowNameByIndex(1) != "" {
		t.Errorf("Expected row names '', '', got %s, %s", result4.GetRowNameByIndex(0), result4.GetRowNameByIndex(1))
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
	if result5.GetRowNameByIndex(0) != "" || result5.GetRowNameByIndex(1) != "Valid" {
		t.Errorf("Expected row names '', Valid, got %s, %s", result5.GetRowNameByIndex(0), result5.GetRowNameByIndex(1))
	}
}
