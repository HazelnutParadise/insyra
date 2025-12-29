package insyra

import (
	"fmt"
	"testing"
)

func toFloat(v any) float64 {
	switch i := v.(type) {
	case int:
		return float64(i)
	case float64:
		return i
	default:
		return 0
	}
}

func TestCCLRangeWithNames(t *testing.T) {
	dt := NewDataTable(
		NewDataList(10, 20, 30, 40, 50).SetName("ColA"),
		NewDataList(1, 2, 3, 4, 5).SetName("ColB"),
		NewDataList(100, 200, 300, 400, 500).SetName("ColC"),
	)

	// Set row names
	dt.SetRowNameByIndex(0, "Row1")
	dt.SetRowNameByIndex(2, "Row3")
	dt.SetRowNameByIndex(4, "Row5")

	// Test 1: Column Range with Names ['ColA']:['ColB']
	// Should return columns ColA and ColB for row 0 (Row1) -> [10, 1]
	dt.AddColUsingCCL("RangeColNames", "['ColA']:['ColB'].0")
	val1 := dt.GetColByName("RangeColNames").Data()[0]

	if s, ok := val1.([]any); ok {
		if len(s) != 2 {
			t.Errorf("RangeColNames length mismatch: got %d, want 2", len(s))
		}
		if toFloat(s[0]) != 10.0 || toFloat(s[1]) != 1.0 {
			t.Errorf("RangeColNames value mismatch: got %v", s)
		}
	} else {
		t.Errorf("RangeColNames type mismatch: got %T", val1)
	}

	// Test 2: Row Range with Names "Row1":"Row3"
	// Should return rows 0, 1, 2 for ColA -> [10, 20, 30]
	// Note: Row1 is index 0, Row3 is index 2.
	dt.AddColUsingCCL("RangeRowNames", "ColA.\"Row1\":\"Row3\"")
	col2 := dt.GetColByName("RangeRowNames")
	if col2 != nil {
		val2 := col2.Data()[0]

		if s, ok := val2.([]any); ok {
			if len(s) != 3 {
				t.Errorf("RangeRowNames length mismatch: got %d, want 3", len(s))
			}
			if toFloat(s[0]) != 10.0 || toFloat(s[1]) != 20.0 || toFloat(s[2]) != 30.0 {
				t.Errorf("RangeRowNames value mismatch: got %v", s)
			}
		} else {
			t.Errorf("RangeRowNames type mismatch: got %T", val2)
		}
	} else {
		t.Errorf("RangeRowNames column not created")
	}

	// Test 3: Mixed Column Range A:['ColC']
	// Should return ColA, ColB, ColC for row 0 -> [10, 1, 100]
	// Assuming ColA is index 0, ColC is index 2.
	dt.AddColUsingCCL("RangeMixed", "A:['ColC'].0")
	col3 := dt.GetColByName("RangeMixed")
	if col3 != nil {
		val3 := col3.Data()[0]
		if s, ok := val3.([]any); ok {
			if len(s) != 3 {
				t.Errorf("RangeMixed length mismatch: got %d, want 3", len(s))
			}
			if toFloat(s[0]) != 10.0 || toFloat(s[1]) != 1.0 || toFloat(s[2]) != 100.0 {
				t.Errorf("RangeMixed value mismatch: got %v", s)
			}
		} else {
			t.Errorf("RangeMixed type mismatch: got %T", val3)
		}
	}

	// Test 4: Mixed Row Range "Row1":2
	// Should return rows 0, 1, 2 for ColA -> [10, 20, 30]
	dt.AddColUsingCCL("RangeRowMixed", "ColA.\"Row1\":2")
	col4 := dt.GetColByName("RangeRowMixed")
	if col4 != nil {
		val4 := col4.Data()[0]
		if s, ok := val4.([]any); ok {
			if len(s) != 3 {
				t.Errorf("RangeRowMixed length mismatch: got %d, want 3", len(s))
			}
			if toFloat(s[0]) != 10.0 || toFloat(s[1]) != 20.0 || toFloat(s[2]) != 30.0 {
				t.Errorf("RangeRowMixed value mismatch: got %v", s)
			}
		} else {
			t.Errorf("RangeRowMixed type mismatch: got %T", val4)
		}
	}

	// Test 5: Multi-char Column Index [AB]
	// We need enough columns to test this. Let's just test parsing logic via a mock or assume it works if no error.
	// Or we can add dummy columns.
	// Let's add columns up to AB (index 27).
	// Current cols: 0, 1, 2, 3 (RangeColNames), 4 (RangeRowNames), 5 (RangeMixed), 6 (RangeRowMixed).
	// We need 21 more columns.
	for i := 7; i <= 27; i++ {
		dt.AddColUsingCCL(fmt.Sprintf("Col%d", i), "0")
	}
	// Now we have column at index 27 (AB).
	// Set value at AB.0 to 999.
	// We can't easily set value via CCL without row index.
	// Let's just read it. It should be 0.
	dt.AddColUsingCCL("TestAB", "[AB].0")
	valAB := dt.GetColByName("TestAB").Data()[0]
	if toFloat(valAB) != 0.0 {
		t.Errorf("TestAB value mismatch: got %v, want 0", valAB)
	}
}
