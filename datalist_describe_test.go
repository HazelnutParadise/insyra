package insyra

import (
	"math"
	"testing"
)

func TestDataListDescribeNumeric(t *testing.T) {
	dl := NewDataList(1, 2, 3, 4).SetName("x")

	desc := dl.Describe()

	if got := desc.GetColByName("x").Data(); len(got) != 12 {
		t.Fatalf("expected 12 summary rows, got %d: %v", len(got), got)
	}
	assertDescribeValue(t, desc, "count", "x", 4)
	assertDescribeValue(t, desc, "missing", "x", 0)
	assertDescribeValue(t, desc, "mean", "x", 2.5)
	assertDescribeValue(t, desc, "25%", "x", 1.75)
	assertDescribeValue(t, desc, "50%", "x", 2.5)
	assertDescribeValue(t, desc, "75%", "x", 3.25)
}

func TestDataListDescribeMissingAndString(t *testing.T) {
	dl := NewDataList("a", "b", "a", nil, math.NaN())

	desc := dl.Describe()

	assertDescribeValue(t, desc, "count", "value", 3)
	assertDescribeValue(t, desc, "missing", "value", 2)
	assertDescribeValue(t, desc, "unique", "value", 2)
	assertDescribeValue(t, desc, "top", "value", "a")
	assertDescribeValue(t, desc, "freq", "value", 2)
	assertDescribeNil(t, desc, "mean", "value")
}

func TestDataListDescribeInvalidPercentileSetsErr(t *testing.T) {
	dl := NewDataList(1, 2, 3)

	desc := dl.Describe(DescribeOptions{Percentiles: []float64{1.2}})

	if desc.NumCols() != 0 {
		t.Fatalf("expected empty description for invalid percentile, got %d columns", desc.NumCols())
	}
	if dl.Err() == nil {
		t.Fatal("expected invalid percentile to be recorded on DataList Err")
	}
}
