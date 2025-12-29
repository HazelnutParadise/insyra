package insyra

import (
	"testing"
)

// TestCCL_NumericStringComparison 測試數字和字串的比較
func TestCCL_NumericStringComparison(t *testing.T) {
	tests := []struct {
		name     string
		colData  []any
		cclExpr  string
		expected []any
	}{
		{
			name:     "String '123' equals number 123",
			colData:  []any{"123", 123, "123", 123},
			cclExpr:  "A == 123",
			expected: []any{true, true, true, true},
		},
		{
			name:     "String '45.5' greater than 40",
			colData:  []any{"45.5", "30", "50.5", "40"},
			cclExpr:  "A > 40",
			expected: []any{true, false, true, false},
		},
		{
			name:     "String arithmetic with numbers",
			colData:  []any{"100", "50", "25", "10"},
			cclExpr:  "A + 50",
			expected: []any{float64(150), float64(100), float64(75), float64(60)},
		},
		{
			name:     "Mixed string and number multiplication",
			colData:  []any{"10", 20, "30", 40},
			cclExpr:  "A * 2",
			expected: []any{float64(20), float64(40), float64(60), float64(80)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dt := NewDataTable(NewDataList(tt.colData...).SetName("A"))
			dt.AddColUsingCCL("result", tt.cclExpr)

			result := dt.GetColByName("result")
			if result == nil {
				t.Fatal("Result column not found")
			}

			for i, expected := range tt.expected {
				got := result.Data()[i]
				if got != expected {
					t.Errorf("Row %d: expected %v (type %T), got %v (type %T)", i, expected, expected, got, got)
				}
			}
		})
	}
}

// TestCCL_NilHandling 測試 nil 值的處理
func TestCCL_NilHandling(t *testing.T) {
	tests := []struct {
		name     string
		colA     []any
		colB     []any
		cclExpr  string
		expected []any
	}{
		{
			name:     "nil equals nil (comparing two nil columns)",
			colA:     []any{nil, nil, 1, nil},
			colB:     []any{nil, nil, nil, 2},
			cclExpr:  "A == B",
			expected: []any{true, true, false, false},
		},
		{
			name:     "nil not equals number",
			colA:     []any{nil, 0, 1, nil},
			colB:     []any{0, 0, 0, 0},
			cclExpr:  "A != B",
			expected: []any{true, false, true, true},
		},
		{
			name:     "nil in comparison (greater than)",
			colA:     []any{nil, 10, 5, nil},
			colB:     []any{5, 5, 5, 5},
			cclExpr:  "A > B",
			expected: []any{false, true, false, false},
		},
		{
			name:     "nil in comparison (less than)",
			colA:     []any{nil, 3, 10, nil},
			colB:     []any{5, 5, 5, 5},
			cclExpr:  "A < B",
			expected: []any{false, true, false, false},
		},
		{
			name:     "nil in addition (treated as 0)",
			colA:     []any{nil, 10, 20, nil},
			colB:     []any{10, 10, 10, 10},
			cclExpr:  "A + B",
			expected: []any{float64(10), float64(20), float64(30), float64(10)},
		},
		{
			name:     "nil in subtraction (treated as 0)",
			colA:     []any{nil, 50, 30, nil},
			colB:     []any{5, 5, 5, 5},
			cclExpr:  "A - B",
			expected: []any{float64(-5), float64(45), float64(25), float64(-5)},
		},
		{
			name:     "nil in multiplication (treated as 0)",
			colA:     []any{nil, 5, 10, nil},
			colB:     []any{3, 3, 3, 3},
			cclExpr:  "A * B",
			expected: []any{float64(0), float64(15), float64(30), float64(0)},
		},
		{
			name:     "Number minus nil (nil treated as 0)",
			colA:     []any{10, 20, 30, 40},
			colB:     []any{nil, nil, nil, nil},
			cclExpr:  "A - B",
			expected: []any{float64(10), float64(20), float64(30), float64(40)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dt := NewDataTable(
				NewDataList(tt.colA...).SetName("A"),
				NewDataList(tt.colB...).SetName("B"),
			)
			dt.AddColUsingCCL("result", tt.cclExpr)

			result := dt.GetColByName("result")
			if result == nil {
				t.Fatal("Result column not found")
			}

			for i, expected := range tt.expected {
				got := result.Data()[i]
				if got != expected {
					t.Errorf("Row %d: expected %v (type %T), got %v (type %T)", i, expected, expected, got, got)
				}
			}
		})
	}
}

// TestCCL_StringConcatenation 測試字串連接
func TestCCL_StringConcatenation(t *testing.T) {
	tests := []struct {
		name     string
		colA     []any
		colB     []any
		cclExpr  string
		expected []any
	}{
		{
			name:     "String concatenation with &",
			colA:     []any{"Hello", "Good", "Hi", "Hey"},
			colB:     []any{" World", " Morning", " There", " You"},
			cclExpr:  "A & B",
			expected: []any{"Hello World", "Good Morning", "Hi There", "Hey You"},
		},
		{
			name:     "Number to string concatenation",
			colA:     []any{"Price: ", "Value: ", "Total: ", "Count: "},
			colB:     []any{123, 45.67, 999, 42},
			cclExpr:  "A & B",
			expected: []any{"Price: 123", "Value: 45.67", "Total: 999", "Count: 42"},
		},
		{
			name:     "nil in string concatenation",
			colA:     []any{"Value: ", "Data: ", "Result: ", "Output: "},
			colB:     []any{nil, "test", nil, 123},
			cclExpr:  "A & B",
			expected: []any{"Value: <nil>", "Data: test", "Result: <nil>", "Output: 123"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dt := NewDataTable(
				NewDataList(tt.colA...).SetName("A"),
				NewDataList(tt.colB...).SetName("B"),
			)
			dt.AddColUsingCCL("result", tt.cclExpr)

			result := dt.GetColByName("result")
			if result == nil {
				t.Fatal("Result column not found")
			}

			for i, expected := range tt.expected {
				got := result.Data()[i]
				if got != expected {
					t.Errorf("Row %d: expected %v (type %T), got %v (type %T)", i, expected, expected, got, got)
				}
			}
		})
	}
}

// TestCCL_BooleanOperations 測試布林運算
func TestCCL_BooleanOperations(t *testing.T) {
	tests := []struct {
		name     string
		colA     []any
		colB     []any
		cclExpr  string
		expected []any
	}{
		{
			name:     "Logical AND",
			colA:     []any{10, 20, 5, 15},
			colB:     []any{15, 15, 15, 25},
			cclExpr:  "(A > 10) && (B < 20)",
			expected: []any{false, true, false, false},
		},
		{
			name:     "Logical OR",
			colA:     []any{10, 20, 5, 15},
			colB:     []any{15, 15, 15, 25},
			cclExpr:  "(A > 15) || (B > 20)",
			expected: []any{false, true, false, true},
		},
		{
			name:     "Complex logical expression",
			colA:     []any{10, 20, 30, 40},
			colB:     []any{5, 15, 25, 35},
			cclExpr:  "(A > 15 && B > 10) || (A < 15 && B < 10)",
			expected: []any{false, true, true, true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dt := NewDataTable(
				NewDataList(tt.colA...).SetName("A"),
				NewDataList(tt.colB...).SetName("B"),
			)
			dt.AddColUsingCCL("result", tt.cclExpr)

			result := dt.GetColByName("result")
			if result == nil {
				t.Fatal("Result column not found")
			}

			for i, expected := range tt.expected {
				got := result.Data()[i]
				if got != expected {
					t.Errorf("Row %d: expected %v (type %T), got %v (type %T)", i, expected, expected, got, got)
				}
			}
		})
	}
}

// TestCCL_ChainedComparison 測試連續比較
func TestCCL_ChainedComparison(t *testing.T) {
	tests := []struct {
		name     string
		colA     []any
		cclExpr  string
		expected []any
	}{
		{
			name:     "Simple range check",
			colA:     []any{5, 10, 15, 20, 25},
			cclExpr:  "10 < A < 20",
			expected: []any{false, false, true, false, false},
		},
		{
			name:     "Inclusive range check",
			colA:     []any{0, 50, 100, 150, 200},
			cclExpr:  "0 <= A <= 100",
			expected: []any{true, true, true, false, false},
		},
		{
			name:     "String numeric chained comparison",
			colA:     []any{"5", "15", "25", "35", "45"},
			cclExpr:  "10 < A < 30",
			expected: []any{false, true, true, false, false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dt := NewDataTable(NewDataList(tt.colA...).SetName("A"))
			dt.AddColUsingCCL("result", tt.cclExpr)

			result := dt.GetColByName("result")
			if result == nil {
				t.Fatal("Result column not found")
			}

			for i, expected := range tt.expected {
				got := result.Data()[i]
				if got != expected {
					t.Errorf("Row %d: expected %v (type %T), got %v (type %T)", i, expected, expected, got, got)
				}
			}
		})
	}
}

// TestCCL_EdgeCases 測試邊界情況
func TestCCL_EdgeCases(t *testing.T) {
	t.Run("nil compared to empty string", func(t *testing.T) {
		dt := NewDataTable(NewDataList(nil, "", "test", nil).SetName("A"))
		dt.AddColUsingCCL("result", "A == ''")

		result := dt.GetColByName("result")
		expected := []any{false, true, false, false}

		for i, exp := range expected {
			got := result.Data()[i]
			if got != exp {
				t.Errorf("Row %d: expected %v, got %v", i, exp, got)
			}
		}
	})

	t.Run("Mixed types in IF condition", func(t *testing.T) {
		dt := NewDataTable(
			NewDataList("100", 50, "200", 150).SetName("A"),
		)
		dt.AddColUsingCCL("result", "IF(A > 100, 'High', 'Low')")

		result := dt.GetColByName("result")
		expected := []any{"Low", "Low", "High", "High"}

		for i, exp := range expected {
			got := result.Data()[i]
			if got != exp {
				t.Errorf("Row %d: expected %v, got %v", i, exp, got)
			}
		}
	})

	t.Run("nil and 'nil' string are different", func(t *testing.T) {
		dt := NewDataTable(
			NewDataList(nil, "nil", nil, "nil").SetName("A"),
			NewDataList(nil, nil, nil, nil).SetName("B"),
		)
		dt.AddColUsingCCL("result", "A == B")

		result := dt.GetColByName("result")
		expected := []any{true, false, true, false}

		for i, exp := range expected {
			got := result.Data()[i]
			if got != exp {
				t.Errorf("Row %d: expected %v, got %v", i, exp, got)
			}
		}
	})
}
