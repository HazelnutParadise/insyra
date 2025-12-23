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
	err := dt.ExecuteCCL("NEW('sum_A') = SUM(A)")
	if err != nil {
		t.Fatalf("ExecuteCCL failed: %v", err)
	}
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
	err = dt.ExecuteCCL("NEW('avg_B') = AVG(B)")
	if err != nil {
		t.Fatalf("ExecuteCCL failed: %v", err)
	}
	colAvgB := dt.GetColByName("avg_B")
	for i := 0; i < 3; i++ {
		if colAvgB.Get(i) != 20.0 {
			t.Errorf("Expected 20.0 at row %d, got %v", i, colAvgB.Get(i))
		}
	}

	// Test COUNT
	err = dt.ExecuteCCL("NEW('count_A') = COUNT(A)")
	if err != nil {
		t.Fatalf("ExecuteCCL failed: %v", err)
	}
	colCountA := dt.GetColByName("count_A")
	for i := 0; i < 3; i++ {
		if colCountA.Get(i) != 3.0 {
			t.Errorf("Expected 3.0 at row %d, got %v", i, colCountA.Get(i))
		}
	}

	// Test MAX/MIN
	err = dt.ExecuteCCL("NEW('max_A') = MAX(A); NEW('min_B') = MIN(B)")
	if err != nil {
		t.Fatalf("ExecuteCCL failed: %v", err)
	}
	colMaxA := dt.GetColByName("max_A")
	colMinB := dt.GetColByName("min_B")
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
	err := dt.ExecuteCCL("NEW('row_sum') = SUM(@.0)")
	if err != nil {
		t.Fatalf("ExecuteCCL failed: %v", err)
	}
	colRowSum := dt.GetColByName("row_sum")
	for i := 0; i < 3; i++ {
		if colRowSum.Get(i) != 11.0 {
			t.Errorf("Expected 11.0 at row %d, got %v", i, colRowSum.Get(i))
		}
	}

	// Test SUM(A.0, B.1) - Sum of specific elements (1 + 20 = 21)
	err = dt.ExecuteCCL("NEW('spec_sum') = SUM(A.0, B.1)")
	if err != nil {
		t.Fatalf("ExecuteCCL failed: %v", err)
	}
	colSpecSum := dt.GetColByName("spec_sum")
	for i := 0; i < 3; i++ {
		if colSpecSum.Get(i) != 21.0 {
			t.Errorf("Expected 21.0 at row %d, got %v", i, colSpecSum.Get(i))
		}
	}

	// Test SUM(A + B) - Sum of (A+B) for all rows (11 + 22 + 33 = 66)
	err = dt.ExecuteCCL("NEW('expr_sum') = SUM(A + B)")
	if err != nil {
		t.Fatalf("ExecuteCCL failed: %v", err)
	}
	colExprSum := dt.GetColByName("expr_sum")
	for i := 0; i < 3; i++ {
		if colExprSum.Get(i) != 66.0 {
			t.Errorf("Expected 66.0 at row %d, got %v", i, colExprSum.Get(i))
		}
	}
}
