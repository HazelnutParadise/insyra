package insyra

import (
	"testing"
)

func buildPriceTable() *DataTable {
	date := NewDataList("D1", "D2", "D3", "D4", "D5")
	date.SetName("date")
	price := NewDataList(100.0, 110.0, 120.0, 115.0, 130.0)
	price.SetName("price")
	dt := NewDataTable()
	dt.AppendCols(date, price)
	return dt
}

func TestDataTable_ShiftCol_ByName(t *testing.T) {
	dt := buildPriceTable()
	got := dt.ShiftCol("price", 1).Data()
	want := []any{nil, 100.0, 110.0, 120.0, 115.0}
	sliceEqualApprox(t, got, want, 1e-9)
}

func TestDataTable_ShiftCol_ByIndex(t *testing.T) {
	dt := buildPriceTable()
	got := dt.ShiftCol("B", 1).Data()
	want := []any{nil, 100.0, 110.0, 120.0, 115.0}
	sliceEqualApprox(t, got, want, 1e-9)
}

func TestDataTable_ShiftCol_MissingColumn(t *testing.T) {
	dt := buildPriceTable()
	got := dt.ShiftCol("missing", 1).Data()
	if len(got) != 0 {
		t.Errorf("expected empty result, got %v", got)
	}
	if err := dt.Err(); err == nil {
		t.Error("expected error to be recorded on dt")
	}
}

func TestDataTable_DiffCol(t *testing.T) {
	dt := buildPriceTable()
	got := dt.DiffCol("price", 1).Data()
	want := []any{nil, 10.0, 10.0, -5.0, 15.0}
	sliceEqualApprox(t, got, want, 1e-9)
}

func TestDataTable_PctChangeCol(t *testing.T) {
	dt := buildPriceTable()
	got := dt.PctChangeCol("price", 1).Data()
	// (110-100)/100, (120-110)/110, (115-120)/120, (130-115)/115
	want := []any{nil, 0.1, 10.0 / 110.0, -5.0 / 120.0, 15.0 / 115.0}
	sliceEqualApprox(t, got, want, 1e-9)
}

func TestDataTable_CumSumCol(t *testing.T) {
	dt := buildPriceTable()
	got := dt.CumSumCol("price").Data()
	want := []any{100.0, 210.0, 330.0, 445.0, 575.0}
	sliceEqualApprox(t, got, want, 1e-9)
}

func TestDataTable_CumMaxCol(t *testing.T) {
	dt := buildPriceTable()
	got := dt.CumMaxCol("price").Data()
	want := []any{100.0, 110.0, 120.0, 120.0, 130.0}
	sliceEqualApprox(t, got, want, 1e-9)
}

func TestDataTable_RollingCol_Mean(t *testing.T) {
	dt := buildPriceTable()
	got := dt.RollingCol("price", RollingOptions{Window: 3}).Mean().Data()
	want := []any{nil, nil, 110.0, 115.0, 121.66666666666667}
	sliceEqualApprox(t, got, want, 1e-9)
}

func TestDataTable_ExpandingCol_Mean(t *testing.T) {
	dt := buildPriceTable()
	got := dt.ExpandingCol("price", 1).Mean().Data()
	want := []any{100.0, 105.0, 110.0, 111.25, 115.0}
	sliceEqualApprox(t, got, want, 1e-9)
}

func TestDataTable_RollingCol_AppendsBack(t *testing.T) {
	// End-to-end: compute, attach, verify table layout.
	dt := buildPriceTable()
	mavg := dt.RollingCol("price", RollingOptions{Window: 3}).Mean()
	mavg.SetName("ma3")
	dt.AppendCols(mavg)

	if dt.NumCols() != 3 {
		t.Fatalf("expected 3 columns after append, got %d", dt.NumCols())
	}
	if dt.GetColByName("ma3") == nil {
		t.Fatal("ma3 column missing after AppendCols")
	}
}
