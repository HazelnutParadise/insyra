package insyra

import (
	"testing"
)

func TestDataTable_ExecuteCCL_AggregateFunctions(t *testing.T) {
	dt := NewDataTable()
	dt.AppendCols(
		NewDataList(1, 2, 3),
		NewDataList(10, 20, 30),
	)
	dt.SetColNames([]string{"A", "B"})

	// Test SUM
	dt.ExecuteCCL("NEW('sum_A') = SUM(A)")
	colSumA := dt.GetColByName("sum_A")
	if colSumA == nil {
		t.Fatal("Column sum_A not found")
	}
	for i := 0; i < 3; i++ {
		if colSumA.Get(i) != 6.0 {
			t.Errorf("Expected 6.0 at row %d, got %v", i, colSumA.Get(i))
		}
	}

	// Test AVG
	dt.ExecuteCCL("NEW('avg_B') = AVG(B)")
	colAvgB := dt.GetColByName("avg_B")
	if colAvgB == nil {
		t.Fatal("Column avg_B not found")
	}
	for i := 0; i < 3; i++ {
		if colAvgB.Get(i) != 20.0 {
			t.Errorf("Expected 20.0 at row %d, got %v", i, colAvgB.Get(i))
		}
	}

	// Test COUNT
	dt.ExecuteCCL("NEW('count_A') = COUNT(A)")
	colCountA := dt.GetColByName("count_A")
	if colCountA == nil {
		t.Fatal("Column count_A not found")
	}
	for i := 0; i < 3; i++ {
		if colCountA.Get(i) != 3.0 {
			t.Errorf("Expected 3.0 at row %d, got %v", i, colCountA.Get(i))
		}
	}

	// Test MAX/MIN
	dt.ExecuteCCL("NEW('max_A') = MAX(A); NEW('min_B') = MIN(B)")
	colMaxA := dt.GetColByName("max_A")
	colMinB := dt.GetColByName("min_B")
	if colMaxA == nil || colMinB == nil {
		t.Fatal("Column max_A or min_B not found")
	}
	for i := 0; i < 3; i++ {
		if colMaxA.Get(i) != 3.0 {
			t.Errorf("Expected 3.0 at row %d, got %v", i, colMaxA.Get(i))
		}
		if colMinB.Get(i) != 10.0 {
			t.Errorf("Expected 10.0 at row %d, got %v", i, colMinB.Get(i))
		}
	}
}

func TestDataTable_ExecuteCCL_AggregateFunctions_Advanced(t *testing.T) {
	dt := NewDataTable()
	dt.AppendCols(
		NewDataList(1, 2, 3),
		NewDataList(10, 20, 30),
	)
	dt.SetColNames([]string{"A", "B"})

	// Test SUM(@.0) - Sum of the first row (1 + 10 = 11)
	dt.ExecuteCCL("NEW('row_sum') = SUM(@.0)")
	colRowSum := dt.GetColByName("row_sum")
	if colRowSum == nil {
		t.Fatal("Column row_sum not found")
	}
	for i := 0; i < 3; i++ {
		if colRowSum.Get(i) != 11.0 {
			t.Errorf("Expected 11.0 at row %d, got %v", i, colRowSum.Get(i))
		}
	}

	// Test SUM(A.0, B.1) - Sum of specific elements (1 + 20 = 21)
	dt.ExecuteCCL("NEW('spec_sum') = SUM(A.0, B.1)")
	colSpecSum := dt.GetColByName("spec_sum")
	if colSpecSum == nil {
		t.Fatal("Column spec_sum not found")
	}
	for i := 0; i < 3; i++ {
		if colSpecSum.Get(i) != 21.0 {
			t.Errorf("Expected 21.0 at row %d, got %v", i, colSpecSum.Get(i))
		}
	}

	// Test SUM(A + B) - Sum of (A+B) for all rows (11 + 22 + 33 = 66)
	dt.ExecuteCCL("NEW('expr_sum') = SUM(A + B)")
	colExprSum := dt.GetColByName("expr_sum")
	if colExprSum == nil {
		t.Fatal("Column expr_sum not found")
	}
	for i := 0; i < 3; i++ {
		if colExprSum.Get(i) != 66.0 {
			t.Errorf("Expected 66.0 at row %d, got %v", i, colExprSum.Get(i))
		}
	}

	// Test SUM(@.#) - Row-wise sum using the new # operator
	// Use a fresh DataTable to avoid interference from previous tests
	dt2 := NewDataTable()
	dt2.AppendCols(
		NewDataList(1, 2, 3),
		NewDataList(10, 20, 30),
	)
	dt2.SetColNames([]string{"A", "B"})
	dt2.ExecuteCCL("NEW('row_sums') = SUM(@.#)")
	colRowSums := dt2.GetColByName("row_sums")
	if colRowSums == nil {
		t.Fatal("Column row_sums not found")
	}
	expectedSums := []float64{11.0, 22.0, 33.0}
	for i := 0; i < 3; i++ {
		if colRowSums.Get(i) != expectedSums[i] {
			t.Errorf("Expected %v at row %d, got %v", expectedSums[i], i, colRowSums.Get(i))
		}
	}
}
