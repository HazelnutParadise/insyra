package insyra

import (
	"math"
	"strings"
	"testing"
)

// TestCCLKeepsIntegersExactOnATable is the end-to-end form of #358: an ID
// column past 2^53 came out of A + 0 with its last digit changed and as
// float64, and SUM over it was off as well.
func TestCCLKeepsIntegersExactOnATable(t *testing.T) {
	const id = int64(9007199254740993)
	dt := NewDataTable(NewDataList(id, id+2, nil).SetName("id"))

	dt.AddColUsingCCL("same", "A + 0")
	dt.AddColUsingCCL("next", "A + 1")
	dt.AddColUsingCCL("total", "SUM(A)")
	dt.AddColUsingCCL("biggest", "MAX(A)")
	dt.AddColUsingCCL("is_first", "A == 9007199254740993")
	dt.AddColUsingCCL("row", "#")
	if err := dt.Err(); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		col  string
		want []any
	}{
		{"same", []any{id, id + 2, int64(0)}},
		{"next", []any{id + 1, id + 3, int64(1)}},
		{"total", []any{2*id + 2, 2*id + 2, 2*id + 2}},
		{"biggest", []any{id + 2, id + 2, id + 2}},
		{"is_first", []any{true, false, false}},
		{"row", []any{int64(0), int64(1), int64(2)}},
	} {
		got := dt.GetColByName(c.col).Data()
		for i := range c.want {
			if got[i] != c.want[i] {
				t.Errorf("%s row %d = %#v (%T), want %#v (%T)", c.col, i, got[i], got[i], c.want[i], c.want[i])
			}
		}
	}

	// A decimal operand still computes in float64.
	f := NewDataTable(NewDataList(int64(3)).SetName("n"))
	f.AddColUsingCCL("half", "A / 2")
	f.AddColUsingCCL("plus", "A + 0.5")
	if err := f.Err(); err != nil {
		t.Fatal(err)
	}
	if got := f.GetColByName("half").Get(0); got != 1.5 {
		t.Errorf("A / 2 = %#v, want 1.5", got)
	}
	if got := f.GetColByName("plus").Get(0); got != 3.5 {
		t.Errorf("A + 0.5 = %#v, want 3.5", got)
	}

	// An integer result past int64 is an error, never a wrapped number.
	o := NewDataTable(NewDataList(int64(math.MaxInt64)).SetName("n"))
	o.AddColUsingCCL("x", "A + 1")
	if err := o.PopErr(); err == nil || !strings.Contains(err.Error(), "overflow") {
		t.Errorf("A + 1 on MaxInt64: error %v, want an overflow error", err)
	}
}
