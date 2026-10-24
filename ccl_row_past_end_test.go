package insyra

import "testing"

func TestCCLRowPastTheEndIsAnError(t *testing.T) {
	dt := NewDataTable(NewDataList(1, 2, 3))
	dt.AddColUsingCCL("n", "COUNT(@.5000)")
	if dt.Err() == nil {
		t.Fatal("@.5000 on a three-row table returned no error")
	}
	dt = NewDataTable(NewDataList(1, 2, 3))
	dt.AddColUsingCCL("n", "COUNT(@.(# + 1))")
	if dt.Err() == nil {
		t.Fatal("@.(# + 1) on the last row returned no error")
	}
	dt = NewDataTable(NewDataList(1, 2, 3), NewDataList(4, 5, 6))
	dt.AddColUsingCCL("n", "SUM(@.#)")
	if err := dt.Err(); err != nil {
		t.Fatalf("a row inside the table failed: %v", err)
	}
}
