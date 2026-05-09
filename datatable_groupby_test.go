package insyra

import (
	"math"
	"reflect"
	"testing"
)

// helper: build a DataTable with named columns from parallel slices.
func buildSalesTable() *DataTable {
	region := NewDataList("east", "east", "west", "west", "east", "south")
	region.SetName("region")
	product := NewDataList("a", "b", "a", "b", "a", "a")
	product.SetName("product")
	revenue := NewDataList(100, 200, 50, 75, 150, 300)
	revenue.SetName("revenue")
	qty := NewDataList(1, 2, 3, 4, 5, 6)
	qty.SetName("qty")
	status := NewDataList("ok", "ok", "ok", nil, "ok", "fail")
	status.SetName("status")
	dt := NewDataTable()
	dt.AppendCols(region, product, revenue, qty, status)
	return dt
}

func mustGetFloat(t *testing.T, dl *DataList, idx int) float64 {
	t.Helper()
	if dl == nil {
		t.Fatalf("DataList is nil")
	}
	d := dl.Data()
	if idx < 0 || idx >= len(d) {
		t.Fatalf("index %d out of range (len=%d)", idx, len(d))
	}
	f, ok := ToFloat64Safe(d[idx])
	if !ok {
		t.Fatalf("value %v is not numeric", d[idx])
	}
	return f
}

func TestDataTable_GroupBy_SingleKey_MultiAgg(t *testing.T) {
	dt := buildSalesTable()
	out := dt.GroupBy("region").Aggregate(
		AggregateConfig{SourceCol: "revenue", Op: OpSum, As: "total_rev"},
		AggregateConfig{SourceCol: "revenue", Op: OpMean, As: "avg_rev"},
		AggregateConfig{SourceCol: "qty", Op: OpSum, As: "total_qty"},
		AggregateConfig{SourceCol: "status", Op: OpCount, As: "n_orders"},
	)

	if out.NumRows() != 3 {
		t.Fatalf("expected 3 groups (east/west/south), got %d", out.NumRows())
	}
	if out.NumCols() != 5 {
		t.Fatalf("expected 5 cols (region + 4 aggregates), got %d", out.NumCols())
	}

	got := []string{}
	for _, v := range out.GetColByName("region").Data() {
		got = append(got, v.(string))
	}
	expected := []string{"east", "west", "south"}
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("first-seen order broken: got %v want %v", got, expected)
	}

	totalRev := out.GetColByName("total_rev").Data()
	if mustGetFloat(t, NewDataList(totalRev[0]), 0) != 450 {
		t.Errorf("east total_rev = %v want 450", totalRev[0])
	}
	if mustGetFloat(t, NewDataList(totalRev[1]), 0) != 125 {
		t.Errorf("west total_rev = %v want 125", totalRev[1])
	}
	if mustGetFloat(t, NewDataList(totalRev[2]), 0) != 300 {
		t.Errorf("south total_rev = %v want 300", totalRev[2])
	}

	avgRev := out.GetColByName("avg_rev").Data()
	if mustGetFloat(t, NewDataList(avgRev[0]), 0) != 150 {
		t.Errorf("east avg_rev = %v want 150", avgRev[0])
	}

	nOrders := out.GetColByName("n_orders").Data()
	// east: 3 non-nil; west: 1 non-nil (one nil); south: 1 non-nil.
	if nOrders[0].(int) != 3 || nOrders[1].(int) != 1 || nOrders[2].(int) != 1 {
		t.Errorf("n_orders = %v want [3 1 1]", nOrders)
	}
}

func TestDataTable_GroupBy_MultiKey_AutoAlias(t *testing.T) {
	dt := buildSalesTable()
	out := dt.GroupBy("region", "product").Aggregate(
		AggregateConfig{SourceCol: "revenue", Op: OpSum},
		AggregateConfig{SourceCol: "qty", Op: OpMean},
	)

	if out.NumCols() != 4 {
		t.Fatalf("expected 4 cols, got %d", out.NumCols())
	}
	headers := out.ColNames()
	wantHeaders := []string{"region", "product", "revenue_sum", "qty_mean"}
	if !reflect.DeepEqual(headers, wantHeaders) {
		t.Fatalf("headers = %v want %v", headers, wantHeaders)
	}

	// Groups (in first-seen order):
	// east+a, east+b, west+a, west+b, south+a => 5 groups.
	if out.NumRows() != 5 {
		t.Fatalf("expected 5 groups, got %d", out.NumRows())
	}
	regions := out.GetColByName("region").Data()
	products := out.GetColByName("product").Data()
	wantRegions := []any{"east", "east", "west", "west", "south"}
	wantProducts := []any{"a", "b", "a", "b", "a"}
	if !reflect.DeepEqual(regions, wantRegions) {
		t.Errorf("regions = %v want %v", regions, wantRegions)
	}
	if !reflect.DeepEqual(products, wantProducts) {
		t.Errorf("products = %v want %v", products, wantProducts)
	}
	revSum := out.GetColByName("revenue_sum").Data()
	// east+a: 100+150=250, east+b: 200, west+a: 50, west+b: 75, south+a: 300.
	want := []float64{250, 200, 50, 75, 300}
	for i, v := range want {
		if got := mustGetFloat(t, NewDataList(revSum[i]), 0); got != v {
			t.Errorf("revenue_sum[%d] = %v want %v", i, got, v)
		}
	}
}

func TestDataTable_GroupBy_OpsCoverage(t *testing.T) {
	dt := buildSalesTable()
	out := dt.GroupBy("region").Aggregate(
		AggregateConfig{SourceCol: "revenue", Op: OpMin, As: "rmin"},
		AggregateConfig{SourceCol: "revenue", Op: OpMax, As: "rmax"},
		AggregateConfig{SourceCol: "revenue", Op: OpMedian, As: "rmed"},
		AggregateConfig{SourceCol: "revenue", Op: OpStdev, As: "rstd"},
		AggregateConfig{SourceCol: "revenue", Op: OpStdevP, As: "rstdp"},
		AggregateConfig{SourceCol: "revenue", Op: OpVar, As: "rvar"},
		AggregateConfig{SourceCol: "revenue", Op: OpVarP, As: "rvarp"},
		AggregateConfig{SourceCol: "status", Op: OpFirst, As: "first_status"},
		AggregateConfig{SourceCol: "status", Op: OpLast, As: "last_status"},
		AggregateConfig{SourceCol: "product", Op: OpNUnique, As: "n_products"},
		AggregateConfig{Op: OpCountAll, As: "n_rows"},
	)

	rmin := out.GetColByName("rmin").Data()
	rmax := out.GetColByName("rmax").Data()
	rmed := out.GetColByName("rmed").Data()
	if mustGetFloat(t, NewDataList(rmin[0]), 0) != 100 {
		t.Errorf("east min = %v want 100", rmin[0])
	}
	if mustGetFloat(t, NewDataList(rmax[0]), 0) != 200 {
		t.Errorf("east max = %v want 200", rmax[0])
	}
	if mustGetFloat(t, NewDataList(rmed[0]), 0) != 150 {
		t.Errorf("east median = %v want 150", rmed[0])
	}

	// south has 1 element; sample stdev/var must be NaN.
	rstd := out.GetColByName("rstd").Data()
	rstdp := out.GetColByName("rstdp").Data()
	rvar := out.GetColByName("rvar").Data()
	rvarp := out.GetColByName("rvarp").Data()
	if v, _ := ToFloat64Safe(rstd[2]); !math.IsNaN(v) {
		t.Errorf("south stdev should be NaN got %v", v)
	}
	if v, _ := ToFloat64Safe(rstdp[2]); v != 0 {
		t.Errorf("south stdevP should be 0 (single-element population), got %v", v)
	}
	if v, _ := ToFloat64Safe(rvar[2]); !math.IsNaN(v) {
		t.Errorf("south var should be NaN got %v", v)
	}
	if v, _ := ToFloat64Safe(rvarp[2]); v != 0 {
		t.Errorf("south varP should be 0 got %v", v)
	}

	firstStatus := out.GetColByName("first_status").Data()
	lastStatus := out.GetColByName("last_status").Data()
	// west has values [ok, nil] -> first=ok, last=ok (skipping the nil)
	if firstStatus[1] != "ok" {
		t.Errorf("west first_status = %v want ok", firstStatus[1])
	}
	if lastStatus[1] != "ok" {
		t.Errorf("west last_status = %v want ok", lastStatus[1])
	}

	nProducts := out.GetColByName("n_products").Data()
	// east has products a,b,a -> 2 unique
	if nProducts[0].(int) != 2 {
		t.Errorf("east n_products = %v want 2", nProducts[0])
	}
	if nProducts[2].(int) != 1 {
		t.Errorf("south n_products = %v want 1", nProducts[2])
	}

	nRows := out.GetColByName("n_rows").Data()
	wantRows := []int{3, 2, 1}
	for i, w := range wantRows {
		if nRows[i].(int) != w {
			t.Errorf("n_rows[%d] = %v want %v", i, nRows[i], w)
		}
	}
}

func TestDataTable_GroupBy_Custom(t *testing.T) {
	dt := buildSalesTable()
	out := dt.GroupBy("region").Aggregate(
		AggregateConfig{
			SourceCol: "revenue",
			As:        "rev_x_qty",
			Op:        OpCustom,
			Custom: func(group *DataList) any {
				// Sum of squares of revenue, ignoring nil.
				s := 0.0
				for _, v := range group.Data() {
					if v == nil {
						continue
					}
					f, ok := ToFloat64Safe(v)
					if !ok {
						continue
					}
					s += f * f
				}
				return s
			},
		},
	)
	got := out.GetColByName("rev_x_qty").Data()
	// east revenue [100,200,150]; sumsq = 10000+40000+22500 = 72500
	if mustGetFloat(t, NewDataList(got[0]), 0) != 72500 {
		t.Errorf("east sumsq = %v want 72500", got[0])
	}
	// west [50,75]; 2500+5625 = 8125
	if mustGetFloat(t, NewDataList(got[1]), 0) != 8125 {
		t.Errorf("west sumsq = %v want 8125", got[1])
	}
	// south [300]; 90000
	if mustGetFloat(t, NewDataList(got[2]), 0) != 90000 {
		t.Errorf("south sumsq = %v want 90000", got[2])
	}
}

func TestDataTable_GroupBy_CustomNil_ReturnsEmpty(t *testing.T) {
	dt := buildSalesTable()
	dt.ClearErr()
	out := dt.GroupBy("region").Aggregate(
		AggregateConfig{SourceCol: "revenue", Op: OpCustom, As: "bad"},
	)
	if out.NumCols() == 0 {
		t.Fatalf("expected key column to still be emitted")
	}
	bad := out.GetColByName("bad").Data()
	for _, v := range bad {
		if v != nil {
			t.Errorf("bad column should be nil-filled when Custom is missing, got %v", v)
		}
	}
	if dt.Err() == nil {
		t.Errorf("expected parent DataTable to record an error when Custom is nil")
	}
}

func TestDataTable_GroupBy_UnknownKey(t *testing.T) {
	dt := buildSalesTable()
	dt.ClearErr()
	out := dt.GroupBy("nonexistent").Aggregate(
		AggregateConfig{SourceCol: "revenue", Op: OpSum},
	)
	if out.NumRows() != 0 || out.NumCols() != 0 {
		t.Fatalf("expected empty DataTable for unknown key, got %dx%d", out.NumRows(), out.NumCols())
	}
	if dt.Err() == nil {
		t.Errorf("expected parent DataTable to record an error for unknown key")
	}
}

func TestDataTable_GroupBy_UnknownSource(t *testing.T) {
	dt := buildSalesTable()
	dt.ClearErr()
	out := dt.GroupBy("region").Aggregate(
		AggregateConfig{SourceCol: "nope", Op: OpSum, As: "bad"},
		AggregateConfig{SourceCol: "revenue", Op: OpSum, As: "good"},
	)
	if out.NumRows() != 3 {
		t.Fatalf("expected 3 groups, got %d", out.NumRows())
	}
	bad := out.GetColByName("bad").Data()
	for _, v := range bad {
		if v != nil {
			t.Errorf("bad column should be nil for unresolved source, got %v", v)
		}
	}
	good := out.GetColByName("good").Data()
	if mustGetFloat(t, NewDataList(good[0]), 0) != 450 {
		t.Errorf("good column was contaminated by sibling failure")
	}
	if dt.Err() == nil {
		t.Errorf("expected parent DataTable to flag the unresolved source col")
	}
}

func TestDataTable_GroupBy_NilKeys(t *testing.T) {
	a := NewDataList("x", nil, "x", nil)
	a.SetName("k")
	b := NewDataList(1, 2, 3, 4)
	b.SetName("v")
	dt := NewDataTable()
	dt.AppendCols(a, b)
	out := dt.GroupBy("k").Aggregate(
		AggregateConfig{SourceCol: "v", Op: OpSum, As: "vs"},
	)
	if out.NumRows() != 2 {
		t.Fatalf("nil should form its own group; expected 2 groups, got %d", out.NumRows())
	}
	keys := out.GetColByName("k").Data()
	if keys[0] != "x" || keys[1] != nil {
		t.Errorf("group keys preserve original typed values: got %v", keys)
	}
	vs := out.GetColByName("vs").Data()
	if mustGetFloat(t, NewDataList(vs[0]), 0) != 4 {
		t.Errorf("x sum = %v want 4", vs[0])
	}
	if mustGetFloat(t, NewDataList(vs[1]), 0) != 6 {
		t.Errorf("nil-key sum = %v want 6", vs[1])
	}
}

func TestDataTable_GroupBy_NumericVsStringKey(t *testing.T) {
	// Strings "1" and integer 1 should not collide.
	k := NewDataList(1, "1", 1, "1")
	k.SetName("k")
	v := NewDataList(10, 20, 30, 40)
	v.SetName("v")
	dt := NewDataTable()
	dt.AppendCols(k, v)
	out := dt.GroupBy("k").Aggregate(
		AggregateConfig{SourceCol: "v", Op: OpSum, As: "vs"},
	)
	if out.NumRows() != 2 {
		t.Fatalf("typed keys should distinguish int 1 from string \"1\", got %d groups", out.NumRows())
	}
	vs := out.GetColByName("vs").Data()
	if mustGetFloat(t, NewDataList(vs[0]), 0) != 40 {
		t.Errorf("int-key sum = %v want 40", vs[0])
	}
	if mustGetFloat(t, NewDataList(vs[1]), 0) != 60 {
		t.Errorf("string-key sum = %v want 60", vs[1])
	}
}

func TestDataTable_GroupBy_EmptyTable(t *testing.T) {
	dt := NewDataTable()
	dt.AppendCols(NewDataList().SetName("k"), NewDataList().SetName("v"))
	out := dt.GroupBy("k").Aggregate(
		AggregateConfig{SourceCol: "v", Op: OpSum, As: "vs"},
	)
	if out.NumRows() != 0 {
		t.Fatalf("empty table should produce zero groups, got %d", out.NumRows())
	}
	if out.NumCols() != 2 {
		t.Fatalf("expected key + agg columns, got %d", out.NumCols())
	}
}

func TestDataTable_GroupBy_SingleRow(t *testing.T) {
	k := NewDataList("only")
	k.SetName("k")
	v := NewDataList(42)
	v.SetName("v")
	dt := NewDataTable()
	dt.AppendCols(k, v)
	out := dt.GroupBy("k").Aggregate(
		AggregateConfig{SourceCol: "v", Op: OpSum, As: "vs"},
	)
	if out.NumRows() != 1 {
		t.Fatalf("expected 1 group, got %d", out.NumRows())
	}
	if mustGetFloat(t, out.GetColByName("vs"), 0) != 42 {
		t.Errorf("single-row sum incorrect")
	}
}

func TestDataTable_GroupBy_SingleGroupAllRows(t *testing.T) {
	k := NewDataList("same", "same", "same")
	k.SetName("k")
	v := NewDataList(1, 2, 3)
	v.SetName("v")
	dt := NewDataTable()
	dt.AppendCols(k, v)
	out := dt.GroupBy("k").Aggregate(
		AggregateConfig{SourceCol: "v", Op: OpSum, As: "vs"},
		AggregateConfig{SourceCol: "v", Op: OpMean, As: "vm"},
	)
	if out.NumRows() != 1 {
		t.Fatalf("expected 1 group, got %d", out.NumRows())
	}
	if mustGetFloat(t, out.GetColByName("vs"), 0) != 6 {
		t.Errorf("sum should be 6")
	}
	if mustGetFloat(t, out.GetColByName("vm"), 0) != 2 {
		t.Errorf("mean should be 2")
	}
}

func TestDataTable_GroupBy_KeyByExcelIndex(t *testing.T) {
	dt := buildSalesTable() // region is column A
	out := dt.GroupBy("A").Aggregate(
		AggregateConfig{SourceCol: "C", Op: OpSum, As: "rev_sum"},
	)
	if out.NumRows() != 3 {
		t.Fatalf("expected 3 groups, got %d", out.NumRows())
	}
	got := out.GetColByName("rev_sum").Data()
	if mustGetFloat(t, NewDataList(got[0]), 0) != 450 {
		t.Errorf("east rev_sum = %v want 450", got[0])
	}
}

func TestDataTable_GroupBy_AggregateAll(t *testing.T) {
	dt := buildSalesTable()
	out := dt.GroupBy("region").AggregateAll(OpSum)
	headers := out.ColNames()
	// region + every non-key column
	want := []string{"region", "product_sum", "revenue_sum", "qty_sum", "status_sum"}
	if !reflect.DeepEqual(headers, want) {
		t.Fatalf("headers = %v want %v", headers, want)
	}
	revSum := out.GetColByName("revenue_sum").Data()
	if mustGetFloat(t, NewDataList(revSum[0]), 0) != 450 {
		t.Errorf("east revenue_sum = %v want 450", revSum[0])
	}
}

func TestDataTable_GroupBy_AggregateAll_RejectsCustom(t *testing.T) {
	dt := buildSalesTable()
	dt.ClearErr()
	out := dt.GroupBy("region").AggregateAll(OpCustom)
	if out.NumRows() != 0 {
		t.Errorf("AggregateAll(OpCustom) should return empty table")
	}
	if dt.Err() == nil {
		t.Errorf("expected parent error for AggregateAll(OpCustom)")
	}
}

func TestDataTable_GroupBy_Count(t *testing.T) {
	dt := buildSalesTable()
	out := dt.GroupBy("region").Count()
	if out.NumCols() != 2 {
		t.Fatalf("expected region + count, got %d cols", out.NumCols())
	}
	count := out.GetColByName("count").Data()
	want := []int{3, 2, 1}
	for i, w := range want {
		if count[i].(int) != w {
			t.Errorf("count[%d] = %v want %v", i, count[i], w)
		}
	}
}

func TestDataTable_GroupBy_NoConfigs(t *testing.T) {
	dt := buildSalesTable()
	dt.ClearErr()
	out := dt.GroupBy("region").Aggregate()
	if out.NumRows() != 0 || out.NumCols() != 0 {
		t.Fatalf("expected empty table when no configs, got %dx%d", out.NumRows(), out.NumCols())
	}
	if dt.Err() == nil {
		t.Errorf("expected parent error when no configs are passed")
	}
}

func TestDataTable_GroupBy_NoKeys(t *testing.T) {
	dt := buildSalesTable()
	dt.ClearErr()
	out := dt.GroupBy().Aggregate(AggregateConfig{SourceCol: "revenue", Op: OpSum})
	if out.NumRows() != 0 || out.NumCols() != 0 {
		t.Fatalf("expected empty table when no keys, got %dx%d", out.NumRows(), out.NumCols())
	}
	if dt.Err() == nil {
		t.Errorf("expected parent error when no keys are passed")
	}
}

func TestDataTable_GroupBy_PipelineWithSortBy(t *testing.T) {
	dt := buildSalesTable()
	out := dt.GroupBy("region").Aggregate(
		AggregateConfig{SourceCol: "revenue", Op: OpSum, As: "total_rev"},
	)
	out.SortBy(DataTableSortConfig{ColumnName: "total_rev", Descending: true})
	regions := out.GetColByName("region").Data()
	if regions[0] != "east" {
		t.Errorf("after sort, top region should be east (450), got %v", regions[0])
	}
	if regions[2] != "west" {
		t.Errorf("after sort, bottom region should be west (125), got %v", regions[2])
	}
}

func TestAggregateOp_String(t *testing.T) {
	cases := map[AggregateOp]string{
		OpSum:      "sum",
		OpMean:     "mean",
		OpMedian:   "median",
		OpMin:      "min",
		OpMax:      "max",
		OpCount:    "count",
		OpCountAll: "countall",
		OpStdev:    "stdev",
		OpStdevP:   "stdevp",
		OpVar:      "var",
		OpVarP:     "varp",
		OpFirst:    "first",
		OpLast:     "last",
		OpNUnique:  "nunique",
		OpCustom:   "custom",
	}
	for op, want := range cases {
		if got := op.String(); got != want {
			t.Errorf("op %d -> %q want %q", int(op), got, want)
		}
	}
}
