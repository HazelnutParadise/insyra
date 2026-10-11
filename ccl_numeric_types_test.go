package insyra

import (
	"fmt"
	"testing"
)

// TestCCLReadsNarrowIntegerColumns is the end-to-end form of the narrow-column
// defect on a loaded table: SUM answered 0 and a comparison answered false,
// neither with an error.
func TestCCLReadsNarrowIntegerColumns(t *testing.T) {
	dt := NewDataTable(NewDataList(int16(3), int16(4), uint8(5)).SetName("a"))

	dt.AddColUsingCCL("s", "SUM(A)")
	if err := dt.Err(); err != nil {
		t.Fatalf("AddColUsingCCL(\"s\", \"SUM(A)\"): %v", err)
	}
	if got := fmt.Sprint(dt.GetColByName("s").Get(0)); got != "12" {
		t.Fatalf("SUM(A) row 0 = %s, want 12", got)
	}

	dt.AddColUsingCCL("e", "A == 4")
	if err := dt.Err(); err != nil {
		t.Fatalf("AddColUsingCCL(\"e\", \"A == 4\"): %v", err)
	}
	if got := dt.GetColByName("e").Get(1); got != true {
		t.Fatalf("A == 4 row 1 = %#v, want true", got)
	}

	if err := dt.Err(); err != nil {
		t.Fatalf("the table carries an error: %v", err)
	}
}
