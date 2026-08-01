package insyra

import (
	"testing"
)

// buildSeqTable builds a small two-column table for CCL sequence-function
// tests: a price column "B" with no NaN.
func buildSeqTable() *DataTable {
	t := NewDataTable()
	price := NewDataList(10.0, 12.0, 11.0, 13.0, 18.0)
	price.SetName("price")
	t.AppendCols(price)
	return t
}

func TestDataTable_CCL_LAG_AddCol(t *testing.T) {
	dt := buildSeqTable()
	dt.AddColUsingCCL("prev", "LAG(A, 1)")
	col := dt.GetColByName("prev")
	if col == nil {
		t.Fatal("prev column missing")
	}
	got := col.Data()
	want := []any{nil, 10.0, 12.0, 11.0, 13.0}
	sliceEqualApprox(t, got, want, 1e-9)
}

func TestDataTable_CCL_CUMSUM_AddCol(t *testing.T) {
	dt := buildSeqTable()
	dt.AddColUsingCCL("cum", "CUMSUM(A)")
	got := dt.GetColByName("cum").Data()
	want := []any{10.0, 22.0, 33.0, 46.0, 64.0}
	sliceEqualApprox(t, got, want, 1e-9)
}

func TestDataTable_CCL_ROLLING_MEAN_AddCol(t *testing.T) {
	dt := buildSeqTable()
	dt.AddColUsingCCL("ma3", "ROLLING_MEAN(A, 3)")
	got := dt.GetColByName("ma3").Data()
	want := []any{nil, nil, 11.0, 12.0, 14.0}
	sliceEqualApprox(t, got, want, 1e-9)
}

func TestDataTable_CCL_DIFF_AddCol(t *testing.T) {
	dt := buildSeqTable()
	dt.AddColUsingCCL("d1", "DIFF(A, 1)")
	got := dt.GetColByName("d1").Data()
	want := []any{nil, 2.0, -1.0, 2.0, 5.0}
	sliceEqualApprox(t, got, want, 1e-9)
}

func TestDataTable_CCL_PCT_CHANGE_AddCol(t *testing.T) {
	dt := buildSeqTable()
	dt.AddColUsingCCL("pct", "PCT_CHANGE(A, 1)")
	got := dt.GetColByName("pct").Data()
	want := []any{nil, 0.2, -1.0 / 12.0, 2.0 / 11.0, 5.0 / 13.0}
	sliceEqualApprox(t, got, want, 1e-9)
}

func TestDataTable_CCL_ExecuteCCL_NEW(t *testing.T) {
	dt := buildSeqTable()
	dt.ExecuteCCL("NEW('cummax') = CUMMAX(A)")
	got := dt.GetColByName("cummax").Data()
	want := []any{10.0, 12.0, 12.0, 13.0, 18.0}
	sliceEqualApprox(t, got, want, 1e-9)
}

func TestDataTable_CCL_ByColumnName(t *testing.T) {
	dt := buildSeqTable()
	dt.AddColUsingCCL("rolling_via_name", "ROLLING_SUM(['price'], 2)")
	got := dt.GetColByName("rolling_via_name").Data()
	want := []any{nil, 22.0, 23.0, 24.0, 31.0}
	sliceEqualApprox(t, got, want, 1e-9)
}

// Regression: confirm that scalar / aggregate CCL functions still work after
// the evaluator dispatch change.
func TestDataTable_CCL_RegressionScalarStillWorks(t *testing.T) {
	dt := buildSeqTable()
	dt.AddColUsingCCL("abs_neg", "ABS(-1 * A)")
	got := dt.GetColByName("abs_neg").Data()
	want := []any{10.0, 12.0, 11.0, 13.0, 18.0}
	sliceEqualApprox(t, got, want, 1e-9)
}

func TestDataTable_CCL_RegressionAggregateStillWorks(t *testing.T) {
	dt := buildSeqTable()
	dt.AddColUsingCCL("total", "SUM(A)")
	col := dt.GetColByName("total")
	if col == nil {
		t.Fatal("total col missing")
	}
	got := col.Data()
	// Aggregate broadcasts to every row.
	for i, v := range got {
		f, ok := ToFloat64Safe(v)
		if !ok || f != 64.0 {
			t.Errorf("[%d] expected 64.0, got %v", i, v)
		}
	}
}
