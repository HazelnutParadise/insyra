package insyra

import (
	"math"
	"testing"
)

// The DataTable context hands aggregates its snapshot of a column instead of a
// copy, and a row-invariant aggregate now runs before the row loop. A function
// that reorders its input in place would therefore change what row access reads
// on every row. MEDIAN sorts; it must sort its own copy.
func TestCCLAggregateDoesNotReorderTheSnapshot(t *testing.T) {
	quietLogs(t)

	dt := NewDataTable(NewDataList(3.0, 1.0, 2.0).SetName("a"))
	dt.AddColUsingCCL("r", "A.0 + 0 * MEDIAN(A) + A.2")
	if err := dt.PopErr(); err != nil {
		t.Fatal(err)
	}
	for i, v := range dt.GetColByName("r").Data() {
		if v != 5.0 {
			t.Fatalf("row %d = %v, want 5 (3 + 2 from the unsorted column)", i, v)
		}
	}
}

// Folding is invisible at the DataTable level: a column computed with a
// row-invariant aggregate matches the same arithmetic done by hand.
func TestCCLFoldedAggregateMatchesManualResult(t *testing.T) {
	quietLogs(t)

	values := []any{4.0, 1.5, 2.5, 8.0}
	dt := NewDataTable(NewDataList(values...).SetName("a"))
	dt.AddColUsingCCL("share", "A / SUM(A)")
	if err := dt.PopErr(); err != nil {
		t.Fatal(err)
	}
	var sum float64
	for _, v := range values {
		sum += v.(float64)
	}
	for i, got := range dt.GetColByName("share").Data() {
		want := values[i].(float64) / sum
		if math.Float64bits(got.(float64)) != math.Float64bits(want) {
			t.Fatalf("row %d = %v, want %v", i, got, want)
		}
	}
}
