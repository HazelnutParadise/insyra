package insyra

import "testing"

// buildPanelTable constructs a small panel dataset:
//
//	id | t | price
//	---+---+------
//	A  | 1 | 10
//	A  | 2 | 12
//	A  | 3 | 11
//	B  | 1 | 100
//	B  | 2 | 110
//	B  | 3 | 105
//
// Rows are interleaved (A1, B1, A2, B2, ...) so we can verify the group-aware
// transforms reconstruct the original row order on output.
func buildPanelTable() *DataTable {
	id := NewDataList("A", "B", "A", "B", "A", "B")
	id.SetName("id")
	tCol := NewDataList(1, 1, 2, 2, 3, 3)
	tCol.SetName("t")
	price := NewDataList(10.0, 100.0, 12.0, 110.0, 11.0, 105.0)
	price.SetName("price")
	dt := NewDataTable()
	dt.AppendCols(id, tCol, price)
	return dt
}

func TestGroupedDataTable_ShiftCol(t *testing.T) {
	dt := buildPanelTable()
	out := dt.GroupBy("id").ShiftCol("price", 1).As("prev_price")
	got := out.Data()
	// Per group, lag-1: A -> [nil, 10, 12]; B -> [nil, 100, 110]
	// Interleaved back: A1, B1, A2, B2, A3, B3
	// -> [nil(A1), nil(B1), 10(A2), 100(B2), 12(A3), 110(B3)]
	want := []any{nil, nil, 10.0, 100.0, 12.0, 110.0}
	sliceEqualApprox(t, got, want, 1e-9)
}

func TestGroupedDataTable_DiffCol(t *testing.T) {
	dt := buildPanelTable()
	out := dt.GroupBy("id").DiffCol("price", 1).As("d1")
	got := out.Data()
	// A: [nil, 12-10=2, 11-12=-1]; B: [nil, 110-100=10, 105-110=-5]
	// Interleaved: [nil(A1), nil(B1), 2(A2), 10(B2), -1(A3), -5(B3)]
	want := []any{nil, nil, 2.0, 10.0, -1.0, -5.0}
	sliceEqualApprox(t, got, want, 1e-9)
}

func TestGroupedDataTable_CumSumCol(t *testing.T) {
	dt := buildPanelTable()
	out := dt.GroupBy("id").CumSumCol("price").As("cum")
	got := out.Data()
	// A: cumsum([10, 12, 11]) = [10, 22, 33]
	// B: cumsum([100, 110, 105]) = [100, 210, 315]
	// Interleaved: [10, 100, 22, 210, 33, 315]
	want := []any{10.0, 100.0, 22.0, 210.0, 33.0, 315.0}
	sliceEqualApprox(t, got, want, 1e-9)
}

func TestGroupedDataTable_RollingCol_Mean(t *testing.T) {
	dt := buildPanelTable()
	out := dt.GroupBy("id").RollingCol("price", RollingOptions{Window: 2}).Mean().As("ma2")
	got := out.Data()
	// A: rolling mean window=2 of [10,12,11] = [nil, 11, 11.5]
	// B: rolling mean window=2 of [100,110,105] = [nil, 105, 107.5]
	// Interleaved: [nil, nil, 11, 105, 11.5, 107.5]
	want := []any{nil, nil, 11.0, 105.0, 11.5, 107.5}
	sliceEqualApprox(t, got, want, 1e-9)
}

func TestGroupedDataTable_ExpandingCol_Mean(t *testing.T) {
	dt := buildPanelTable()
	out := dt.GroupBy("id").ExpandingCol("price", 1).Mean().As("emean")
	got := out.Data()
	// A: [10, 11, 11]
	// B: [100, 105, 105]
	// Interleaved: [10, 100, 11, 105, 11, 105]
	want := []any{10.0, 100.0, 11.0, 105.0, 11.0, 105.0}
	sliceEqualApprox(t, got, want, 1e-9)
}

func TestGroupedDataTable_MissingCol(t *testing.T) {
	dt := buildPanelTable()
	out := dt.GroupBy("id").CumSumCol("missing").As("x")
	if len(out.Data()) != 0 {
		t.Errorf("expected empty result, got %v", out.Data())
	}
}

func TestGroupedDataTable_BadGroupBy(t *testing.T) {
	dt := buildPanelTable()
	out := dt.GroupBy("nope").ShiftCol("price", 1).As("x")
	if len(out.Data()) != 0 {
		t.Errorf("expected empty result on bad GroupBy, got %v", out.Data())
	}
}
