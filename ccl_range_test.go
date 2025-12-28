package insyra

import (
	"fmt"
	"testing"
)

func TestCCLRange(t *testing.T) {
	dt := NewDataTable(
		NewDataList(10, 20, 30, 40, 50).SetName("A"),
		NewDataList(1, 2, 3, 4, 5).SetName("B"),
	)

	// Test 1: Row Range A.0:2
	// Should return [10, 20, 30]
	dt.AddColUsingCCL("Range1", "A.0:2")
	col := dt.GetColByName("Range1")
	if col == nil {
		t.Fatal("Range1 not created")
	}
	val := col.Data()[0]
	fmt.Printf("Range1: %v (Type: %T)\n", val, val)

	// Verify content
	if s, ok := val.([]any); ok {
		if len(s) != 3 {
			t.Errorf("Range1 length mismatch: got %d, want 3", len(s))
		}
	} else {
		t.Errorf("Range1 type mismatch: got %T, want []any", val)
	}

	// Test 2: Column Range A:B.0
	// Should return [10, 1]
	dt.AddColUsingCCL("Range2", "A:B.0")
	val2 := dt.GetColByName("Range2").Data()[0]
	fmt.Printf("Range2: %v (Type: %T)\n", val2, val2)

	if s, ok := val2.([]any); ok {
		if len(s) != 2 {
			t.Errorf("Range2 length mismatch: got %d, want 2", len(s))
		}
	} else {
		t.Errorf("Range2 type mismatch: got %T, want []any", val2)
	}

	// Test 3: Column Range and Row Range A:B.0:1
	// Should return [[10, 1], [20, 2]]
	dt.AddColUsingCCL("Range3", "A:B.0:1")
	val3 := dt.GetColByName("Range3").Data()[0]
	fmt.Printf("Range3: %v (Type: %T)\n", val3, val3)

	if s, ok := val3.([]any); ok {
		if len(s) != 2 {
			t.Errorf("Range3 length mismatch: got %d, want 2", len(s))
		}
		// Check inner types
		if len(s) > 0 {
			if _, ok := s[0].([]any); !ok {
				t.Errorf("Range3 inner type mismatch: got %T, want []any", s[0])
			}
		}
	} else {
		t.Errorf("Range3 type mismatch: got %T, want []any", val3)
	}
}
