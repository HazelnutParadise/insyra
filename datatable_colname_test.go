package insyra

import (
	"testing"
)

func TestSetColNames(t *testing.T) {
	// Test case 1: Equal length
	dt1 := NewDataTable()
	dt1.AppendCols(NewDataList().SetName("A"), NewDataList().SetName("B"))
	colNames1 := []string{"X", "Y"}
	result1 := dt1.SetColNames(colNames1)
	if len(result1.columns) != 2 {
		t.Errorf("Expected 2 columns, got %d", len(result1.columns))
	}
	if result1.columns[0].name != "X" || result1.columns[1].name != "Y" {
		t.Errorf("Expected names X, Y, got %s, %s", result1.columns[0].name, result1.columns[1].name)
	}

	// Test case 2: More colNames than columns (add new columns)
	dt2 := NewDataTable()
	dt2.AppendCols(NewDataList().SetName("A"))
	colNames2 := []string{"X", "Y", "Z"}
	result2 := dt2.SetColNames(colNames2)
	if len(result2.columns) != 3 {
		t.Errorf("Expected 3 columns, got %d", len(result2.columns))
	}
	if result2.columns[0].name != "X" || result2.columns[1].name != "Y" || result2.columns[2].name != "Z" {
		t.Errorf("Expected names X, Y, Z, got %s, %s, %s", result2.columns[0].name, result2.columns[1].name, result2.columns[2].name)
	}

	// Test case 3: Fewer colNames than columns (set excess to empty)
	dt3 := NewDataTable()
	dt3.AppendCols(NewDataList().SetName("A"), NewDataList().SetName("B"), NewDataList().SetName("C"))
	colNames3 := []string{"X", "Y"}
	result3 := dt3.SetColNames(colNames3)
	if len(result3.columns) != 3 {
		t.Errorf("Expected 3 columns, got %d", len(result3.columns))
	}
	if result3.columns[0].name != "X" || result3.columns[1].name != "Y" || result3.columns[2].name != "" {
		t.Errorf("Expected names X, Y, '', got %s, %s, %s", result3.columns[0].name, result3.columns[1].name, result3.columns[2].name)
	}

	// Test case 4: Empty colNames
	dt4 := NewDataTable()
	dt4.AppendCols(NewDataList().SetName("A"), NewDataList().SetName("B"))
	colNames4 := []string{}
	result4 := dt4.SetColNames(colNames4)
	if result4.columns[0].name != "" || result4.columns[1].name != "" {
		t.Errorf("Expected names '', '', got %s, %s", result4.columns[0].name, result4.columns[1].name)
	}

	// Test case 5: ColNames with empty strings
	dt5 := NewDataTable()
	dt5.AppendCols(NewDataList().SetName("A"), NewDataList().SetName("B"))
	colNames5 := []string{"", "Y"}
	result5 := dt5.SetColNames(colNames5)
	if result5.columns[0].name != "" || result5.columns[1].name != "Y" {
		t.Errorf("Expected names '', Y, got %s, %s", result5.columns[0].name, result5.columns[1].name)
	}

	// Test case 6: Name conflict resolution
	dt6 := NewDataTable()
	dt6.AppendCols(NewDataList().SetName("A"), NewDataList().SetName("B"))
	colNames6 := []string{"X", "X"}
	result6 := dt6.SetColNames(colNames6)
	if result6.columns[0].name != "X" || result6.columns[1].name != "X_1" {
		t.Errorf("Expected names X, X_1, got %s, %s", result6.columns[0].name, result6.columns[1].name)
	}
}
