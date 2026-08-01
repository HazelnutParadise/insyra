package insyra

import (
	"math"
	"testing"
)

func TestDataTableDescribeSkipsNonNumericByDefault(t *testing.T) {
	dt := NewDataTable(
		NewDataList(1, 2, 3).SetName("x"),
		NewDataList("a", "b", "a").SetName("label"),
		NewDataList(10, nil, math.NaN()).SetName("y"),
	)

	desc := dt.Describe()

	if got := desc.ColNames(); len(got) != 2 || got[0] != "x" || got[1] != "y" {
		t.Fatalf("expected only numeric columns x,y, got %v", got)
	}
	assertDescribeValue(t, desc, "mean", "x", 2.0)
	assertDescribeValue(t, desc, "count", "y", 1)
	assertDescribeValue(t, desc, "missing", "y", 2)
}

func TestDataTableDescribeIncludeAll(t *testing.T) {
	dt := NewDataTable(
		NewDataList(1, 2, 3).SetName("x"),
		NewDataList("a", "b", "a", nil).SetName("label"),
		NewDataList(1, "1", true).SetName("mixed"),
	)

	desc := dt.Describe(DescribeOptions{IncludeAll: true})

	if got := desc.ColNames(); len(got) != 3 {
		t.Fatalf("expected all columns, got %v", got)
	}
	assertDescribeValue(t, desc, "unique", "label", 2)
	assertDescribeValue(t, desc, "top", "label", "a")
	assertDescribeValue(t, desc, "freq", "label", 2)
	assertDescribeValue(t, desc, "unique", "mixed", 3)
	assertDescribeNil(t, desc, "mean", "mixed")
}

func TestDataTableDescribeCustomPercentiles(t *testing.T) {
	dt := NewDataTable(NewDataList(1, 2, 3, 4, 5).SetName("x"))

	desc := dt.Describe(DescribeOptions{Percentiles: []float64{0.9, 0.1, 0.1}})

	if _, ok := desc.GetRowIndexByName("10%"); !ok {
		t.Fatal("expected 10% row")
	}
	if _, ok := desc.GetRowIndexByName("90%"); !ok {
		t.Fatal("expected 90% row")
	}
	assertDescribeValue(t, desc, "10%", "x", 1.4)
	assertDescribeValue(t, desc, "90%", "x", 4.6)
}

func TestDataTableDescribeAllMissingColumn(t *testing.T) {
	dt := NewDataTable(NewDataList(nil, math.NaN()).SetName("x"))

	desc := dt.Describe()
	if desc.NumCols() != 0 {
		t.Fatalf("expected all-missing column to be skipped by default, got %v", desc.ColNames())
	}

	all := dt.Describe(DescribeOptions{IncludeAll: true})
	assertDescribeValue(t, all, "count", "x", 0)
	assertDescribeValue(t, all, "missing", "x", 2)
	assertDescribeValue(t, all, "unique", "x", 0)
}

func assertDescribeValue(t *testing.T, dt *DataTable, rowName, colName string, want any) {
	t.Helper()
	row, ok := dt.GetRowIndexByName(rowName)
	if !ok {
		t.Fatalf("row %q not found", rowName)
	}
	got := dt.GetColByName(colName).Get(row)
	switch w := want.(type) {
	case float64:
		g, ok := got.(float64)
		if !ok || math.Abs(g-w) > 1e-9 {
			t.Fatalf("row %s col %s: got %v (%T), want %v", rowName, colName, got, got, want)
		}
	case int:
		if got != w {
			t.Fatalf("row %s col %s: got %v (%T), want %v", rowName, colName, got, got, want)
		}
	default:
		if got != want {
			t.Fatalf("row %s col %s: got %v (%T), want %v", rowName, colName, got, got, want)
		}
	}
}

func assertDescribeNil(t *testing.T, dt *DataTable, rowName, colName string) {
	t.Helper()
	row, ok := dt.GetRowIndexByName(rowName)
	if !ok {
		t.Fatalf("row %q not found", rowName)
	}
	if got := dt.GetColByName(colName).Get(row); got != nil {
		t.Fatalf("row %s col %s: got %v, want nil", rowName, colName, got)
	}
}
