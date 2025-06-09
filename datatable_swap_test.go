package insyra

import (
	"reflect"
	"testing"
)

func TestDataTable_SwapColsByName(t *testing.T) {
	// Test case 1: Basic swap
	dt1 := NewDataTable(
		NewDataList(1, 2, 3).SetName("Col1"),
		NewDataList("a", "b", "c").SetName("Col2"),
		NewDataList(true, false, true).SetName("Col3"),
	)
	dt1.SwapColsByName("Col1", "Col3")
	expectedCols1 := []*DataList{
		NewDataList(true, false, true).SetName("Col3"),
		NewDataList("a", "b", "c").SetName("Col2"),
		NewDataList(1, 2, 3).SetName("Col1"),
	}
	if !reflect.DeepEqual(dt1.columns[0].name, expectedCols1[0].name) || !reflect.DeepEqual(dt1.columns[0].data, expectedCols1[0].data) {
		t.Errorf("TestSwapColsByName Case 1 Failed: Expected col 0 to be %v, got %v", expectedCols1[0], dt1.columns[0])
	}
	if !reflect.DeepEqual(dt1.columns[2].name, expectedCols1[2].name) || !reflect.DeepEqual(dt1.columns[2].data, expectedCols1[2].data) {
		t.Errorf("TestSwapColsByName Case 1 Failed: Expected col 2 to be %v, got %v", expectedCols1[2], dt1.columns[2])
	}

	// Test case 2: Swap with non-existent column
	dt2 := NewDataTable(
		NewDataList(1).SetName("ColA"),
		NewDataList(2).SetName("ColB"),
	)
	dt2.SwapColsByName("ColA", "ColC") // ColC does not exist
	if dt2.columns[0].name != "ColA" { // Should not change
		t.Errorf("TestSwapColsByName Case 2 Failed: Expected ColA to remain, got %s", dt2.columns[0].name)
	}

	// Test case 3: Swap same column
	dt3 := NewDataTable(
		NewDataList(10).SetName("X"),
		NewDataList(20).SetName("Y"),
	)
	dt3.SwapColsByName("X", "X")
	if dt3.columns[0].name != "X" || !reflect.DeepEqual(dt3.columns[0].data, []any{10}) {
		t.Errorf("TestSwapColsByName Case 3 Failed: Expected column X to remain unchanged.")
	}
}

func TestDataTable_SwapColsByIndex(t *testing.T) {
	// Test case 1: Basic swap
	dt1 := NewDataTable(
		NewDataList([]any{1, 2}).SetName("Col1"),
		NewDataList([]any{"a", "b"}).SetName("Col2"),
		NewDataList([]any{true, false}).SetName("Col3"),
	)
	dt1.SwapColsByIndex("A", "C")
	expectedColA_Name1 := "Col3"
	expectedColA_Data1 := []any{true, false}
	expectedColC_Name1 := "Col1"
	expectedColC_Data1 := []any{1, 2}

	if dt1.columns[dt1.columnIndex["A"]].name != expectedColA_Name1 || !reflect.DeepEqual(dt1.columns[dt1.columnIndex["A"]].data, expectedColA_Data1) {
		t.Errorf("TestSwapColsByIndex Case 1 Failed: Expected col A to be %s %v, got %s %v", expectedColA_Name1, expectedColA_Data1, dt1.columns[dt1.columnIndex["A"]].name, dt1.columns[dt1.columnIndex["A"]].data)
	}
	if dt1.columns[dt1.columnIndex["C"]].name != expectedColC_Name1 || !reflect.DeepEqual(dt1.columns[dt1.columnIndex["C"]].data, expectedColC_Data1) {
		t.Errorf("TestSwapColsByIndex Case 1 Failed: Expected col C to be %s %v, got %s %v", expectedColC_Name1, expectedColC_Data1, dt1.columns[dt1.columnIndex["C"]].name, dt1.columns[dt1.columnIndex["C"]].data)
	}

	// Test case 2: Swap with non-existent index
	dt2 := NewDataTable(
		NewDataList(1).SetName("ColX"),
	)
	dt2.SwapColsByIndex("A", "B")                         // Index B does not exist for a single column table
	if dt2.columns[dt2.columnIndex["A"]].name != "ColX" { // Should not change
		t.Errorf("TestSwapColsByIndex Case 2 Failed: Expected ColX to remain, got %s", dt2.columns[dt2.columnIndex["A"]].name)
	}

	// Test case 3: Swap same index
	dt3 := NewDataTable(
		NewDataList([]any{100}).SetName("P"),
		NewDataList([]any{200}).SetName("Q"),
	)
	dt3.SwapColsByIndex("A", "A")
	if dt3.columns[dt3.columnIndex["A"]].name != "P" || !reflect.DeepEqual(dt3.columns[dt3.columnIndex["A"]].data, []any{100}) {
		t.Errorf("TestSwapColsByIndex Case 3 Failed: Expected column P at index A to remain unchanged.")
	}
}

func TestDataTable_SwapColsByNumber(t *testing.T) {
	// Test case 1: Basic swap
	dt1 := NewDataTable(
		NewDataList(10, 20).SetName("Num1"),
		NewDataList(30, 40).SetName("Num2"),
		NewDataList(50, 60).SetName("Num3"),
	)
	dt1.SwapColsByNumber(0, 2)
	expectedCol0_Name1 := "Num3"
	expectedCol0_Data1 := []any{50, 60}
	expectedCol2_Name1 := "Num1"
	expectedCol2_Data1 := []any{10, 20}

	if dt1.columns[0].name != expectedCol0_Name1 || !reflect.DeepEqual(dt1.columns[0].data, expectedCol0_Data1) {
		t.Errorf("TestSwapColsByNumber Case 1 Failed: Expected col 0 to be %s %v, got %s %v", expectedCol0_Name1, expectedCol0_Data1, dt1.columns[0].name, dt1.columns[0].data)
	}
	if dt1.columns[2].name != expectedCol2_Name1 || !reflect.DeepEqual(dt1.columns[2].data, expectedCol2_Data1) {
		t.Errorf("TestSwapColsByNumber Case 1 Failed: Expected col 2 to be %s %v, got %s %v", expectedCol2_Name1, expectedCol2_Data1, dt1.columns[2].name, dt1.columns[2].data)
	}

	// Test case 2: Index out of range
	dt2 := NewDataTable(
		NewDataList(1).SetName("Val1"),
	)
	dt2.SwapColsByNumber(0, 1)         // Index 1 is out of range
	if dt2.columns[0].name != "Val1" { // Should not change
		t.Errorf("TestSwapColsByNumber Case 2 Failed: Expected Val1 to remain, got %s", dt2.columns[0].name)
	}

	// Test case 3: Swap same number index
	dt3 := NewDataTable(
		NewDataList(11).SetName("First"),
		NewDataList(22).SetName("Second"),
	)
	dt3.SwapColsByNumber(0, 0)
	if dt3.columns[0].name != "First" || !reflect.DeepEqual(dt3.columns[0].data, []any{11}) {
		t.Errorf("TestSwapColsByNumber Case 3 Failed: Expected column First at index 0 to remain unchanged.")
	}

	// Test case 4: Negative index
	dt4 := NewDataTable(
		NewDataList(1).SetName("NegTest"),
		NewDataList(2).SetName("Another"),
	)
	dt4.SwapColsByNumber(-1, 0)
	if dt4.columns[1].name != "NegTest" {
		t.Errorf("TestSwapColsByNumber Case 4 Failed: Expected NegTest to remain with negative index input.")
	}
}

func TestDataTable_SwapRowsByIndex(t *testing.T) {
	// Test case 1: Basic swap
	dt1 := NewDataTable(
		NewDataList(1, "a", true).SetName("Col1"),
		NewDataList(2, "b", false).SetName("Col2"),
	)
	dt1.SetRowNameByIndex(0, "Row1")
	dt1.SetRowNameByIndex(1, "Row2")
	dt1.SetRowNameByIndex(2, "Row3")
	dt1.SwapRowsByIndex(0, 2)
	expectedRow0Col1_1 := true // Data from original Row3, Col1 (index 2)
	expectedRow2Col1_1 := 1    // Data from original Row1, Col1 (index 0)

	if dt1.columns[0].data[0] != expectedRow0Col1_1 {
		t.Errorf("TestSwapRowsByIndex Case 1 Failed: Expected row 0 col 0 to be %v, got %v", expectedRow0Col1_1, dt1.columns[0].data[0])
	}
	if dt1.columns[0].data[2] != expectedRow2Col1_1 {
		t.Errorf("TestSwapRowsByIndex Case 1 Failed: Expected row 2 col 0 to be %v, got %v", expectedRow2Col1_1, dt1.columns[0].data[2])
	}

	// Test case 2: Index out of range
	dt2 := NewDataTable(
		NewDataList(1, 2).SetName("ColA"),
	)
	dt2.SetRowNameByIndex(0, "R1")
	dt2.SetRowNameByIndex(1, "R2")
	dt2.SwapRowsByIndex(0, 2)        // Index 2 is out of range
	if dt2.columns[0].data[0] != 1 { // Should not change
		t.Errorf("TestSwapRowsByIndex Case 2 Failed: Expected row 0 to remain 1, got %v", dt2.columns[0].data[0])
	}

	// Test case 3: Swap same index
	dt3 := NewDataTable(
		NewDataList(10, 20).SetName("ColX"),
	)
	dt3.SetRowNameByIndex(0, "T1")
	dt3.SetRowNameByIndex(1, "T2")
	dt3.SwapRowsByIndex(0, 0)
	if dt3.columns[0].data[0] != 10 {
		t.Errorf("TestSwapRowsByIndex Case 3 Failed: Expected row 0 to remain 10, got %v", dt3.columns[0].data[0])
	}

	// Test case 4: Negative index
	dt4 := NewDataTable(
		NewDataList(100, 200, 300).SetName("ColY"),
	)
	dt4.SetRowNameByIndex(0, "N1")
	dt4.SetRowNameByIndex(1, "N2")
	dt4.SetRowNameByIndex(2, "N3")
	dt4.SwapRowsByIndex(-1, 0) // Swap last row with first row
	expectedRow0Col0_4 := 300
	expectedRowLastCol0_4 := 100
	if dt4.columns[0].data[0] != expectedRow0Col0_4 {
		t.Errorf("TestSwapRowsByIndex Case 4 Failed: Expected row 0 to be %v, got %v", expectedRow0Col0_4, dt4.columns[0].data[0])
	}
	if dt4.columns[0].data[2] != expectedRowLastCol0_4 {
		t.Errorf("TestSwapRowsByIndex Case 4 Failed: Expected row 2 to be %v, got %v", expectedRowLastCol0_4, dt4.columns[0].data[2])
	}
}

func TestDataTable_SwapRowsByName(t *testing.T) {
	// Test case 1: Basic swap
	dt1 := NewDataTable(
		NewDataList(1, "a").SetName("Col1"),
		NewDataList(2, "b").SetName("Col2"),
	)
	dt1.SetRowNameByIndex(0, "RowA")
	dt1.SetRowNameByIndex(1, "RowB")
	dt1.SwapRowsByName("RowA", "RowB")
	expectedRowAIndex_1 := 1 // After swap, RowA should be at index 1
	expectedRowBIndex_1 := 0 // After swap, RowB should be at index 0

	if dt1.rowNames["RowA"] != expectedRowAIndex_1 {
		t.Errorf("TestSwapRowsByName Case 1 Failed: Expected RowA to be at index %d, got %d", expectedRowAIndex_1, dt1.rowNames["RowA"])
	}
	if dt1.rowNames["RowB"] != expectedRowBIndex_1 {
		t.Errorf("TestSwapRowsByName Case 1 Failed: Expected RowB to be at index %d, got %d", expectedRowBIndex_1, dt1.rowNames["RowB"])
	}
	if dt1.columns[0].data[0] != "a" { // Data of original RowB, Col1 (index 1)
		t.Errorf("TestSwapRowsByName Case 1 Failed: Expected data at [0,0] to be \"a\", got %v", dt1.columns[0].data[0])
	}

	// Test case 2: Swap with non-existent row name
	dt2 := NewDataTable(
		NewDataList(10).SetName("ColX"),
	)
	dt2.SetRowNameByIndex(0, "RAlpha")
	dt2.SwapRowsByName("RAlpha", "RBeta") // RBeta does not exist
	if dt2.rowNames["RAlpha"] != 0 {      // Should not change
		t.Errorf("TestSwapRowsByName Case 2 Failed: Expected RAlpha to remain at index 0, got %d", dt2.rowNames["RAlpha"])
	}

	// Test case 3: Swap same row name
	dt3 := NewDataTable(
		NewDataList(100, 200).SetName("ColY"),
	)
	dt3.SetRowNameByIndex(0, "TFirst")
	dt3.SetRowNameByIndex(1, "TSecond")
	dt3.SwapRowsByName("TFirst", "TFirst")
	if dt3.rowNames["TFirst"] != 0 || dt3.columns[0].data[0] != 100 {
		t.Errorf("TestSwapRowsByName Case 3 Failed: Expected TFirst to remain unchanged.")
	}
}
