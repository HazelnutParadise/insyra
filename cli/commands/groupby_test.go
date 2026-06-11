package commands

import (
	"bytes"
	"reflect"
	"testing"

	insyra "github.com/HazelnutParadise/insyra"
)

func groupbyTestTable() *insyra.DataTable {
	region := insyra.NewDataList("east", "east", "west", "west", "south")
	region.SetName("region")
	revenue := insyra.NewDataList(100, 200, 50, 75, 300)
	revenue.SetName("revenue")
	qty := insyra.NewDataList(1, 2, 3, 4, 5)
	qty.SetName("qty")
	t := insyra.NewDataTable()
	t.AppendCols(region, revenue, qty)
	return t
}

func TestRunGroupByCommand_HappyPath(t *testing.T) {
	ctx := &ExecContext{
		Vars:   map[string]any{"sales": groupbyTestTable()},
		Output: &bytes.Buffer{},
	}
	err := runGroupByCommand(ctx, []string{
		"sales", "by", "region",
		"agg", "revenue:sum:total_rev", "qty:mean", "as", "report",
	})
	if err != nil {
		t.Fatalf("runGroupByCommand failed: %v", err)
	}
	report, ok := ctx.Vars["report"].(*insyra.DataTable)
	if !ok {
		t.Fatalf("expected report DataTable, got %T", ctx.Vars["report"])
	}
	if report.NumRows() != 3 {
		t.Fatalf("expected 3 groups, got %d", report.NumRows())
	}
	headers := report.ColNames()
	wantHeaders := []string{"region", "total_rev", "qty_mean"}
	if !reflect.DeepEqual(headers, wantHeaders) {
		t.Fatalf("headers = %v want %v", headers, wantHeaders)
	}
	rev := report.GetColByName("total_rev").Data()
	if got, _ := insyra.ToFloat64Safe(rev[0]); got != 300 {
		t.Errorf("east total_rev = %v want 300", rev[0])
	}
}

func TestRunGroupByCommand_MultiKey(t *testing.T) {
	tbl := insyra.NewDataTable()
	region := insyra.NewDataList("east", "east", "west")
	region.SetName("region")
	product := insyra.NewDataList("a", "b", "a")
	product.SetName("product")
	revenue := insyra.NewDataList(10, 20, 30)
	revenue.SetName("revenue")
	tbl.AppendCols(region, product, revenue)

	ctx := &ExecContext{
		Vars:   map[string]any{"t": tbl},
		Output: &bytes.Buffer{},
	}
	err := runGroupByCommand(ctx, []string{
		"t", "by", "region,product",
		"agg", "revenue:sum",
	})
	if err != nil {
		t.Fatalf("runGroupByCommand failed: %v", err)
	}
	out, ok := ctx.Vars["$result"].(*insyra.DataTable)
	if !ok {
		t.Fatalf("expected default $result DataTable, got %T", ctx.Vars["$result"])
	}
	if out.NumRows() != 3 {
		t.Fatalf("expected 3 groups, got %d", out.NumRows())
	}
}

func TestRunGroupByCommand_CountShorthand(t *testing.T) {
	ctx := &ExecContext{
		Vars:   map[string]any{"sales": groupbyTestTable()},
		Output: &bytes.Buffer{},
	}
	err := runGroupByCommand(ctx, []string{
		"sales", "by", "region", "agg", "count", "as", "report",
	})
	if err != nil {
		t.Fatalf("runGroupByCommand failed: %v", err)
	}
	report := ctx.Vars["report"].(*insyra.DataTable)
	count := report.GetColByName("count").Data()
	want := []int{2, 2, 1}
	for i, w := range want {
		if count[i].(int) != w {
			t.Errorf("count[%d] = %v want %v", i, count[i], w)
		}
	}
}

func TestRunGroupByCommand_InvalidUsage(t *testing.T) {
	ctx := &ExecContext{Vars: map[string]any{}, Output: &bytes.Buffer{}}
	if err := runGroupByCommand(ctx, []string{"only"}); err == nil {
		t.Fatalf("expected error for too few args")
	}
	if err := runGroupByCommand(ctx, []string{"a", "wrong", "b", "agg", "x:sum"}); err == nil {
		t.Fatalf("expected error when 'by' keyword is missing")
	}
	if err := runGroupByCommand(ctx, []string{"a", "by", "k", "wrong", "x:sum"}); err == nil {
		t.Fatalf("expected error when 'agg' keyword is missing")
	}
}

func TestRunGroupByCommand_UnknownVar(t *testing.T) {
	ctx := &ExecContext{Vars: map[string]any{}, Output: &bytes.Buffer{}}
	err := runGroupByCommand(ctx, []string{"missing", "by", "k", "agg", "x:sum"})
	if err == nil {
		t.Fatalf("expected error for unknown variable")
	}
}

func TestParseAggregateSpec(t *testing.T) {
	cases := []struct {
		raw  string
		op   insyra.AggregateOp
		col  string
		as   string
		fail bool
	}{
		{"revenue:sum", insyra.OpSum, "revenue", "", false},
		{"revenue:sum:total", insyra.OpSum, "revenue", "total", false},
		{"qty:avg", insyra.OpMean, "qty", "", false},
		{"price:median:p", insyra.OpMedian, "price", "p", false},
		{"x:std", insyra.OpStdev, "x", "", false},
		{"x:stdp", insyra.OpStdevP, "x", "", false},
		{"x:var", insyra.OpVar, "x", "", false},
		{"x:varp", insyra.OpVarP, "x", "", false},
		{"x:nunique", insyra.OpNUnique, "x", "", false},
		{"x:first", insyra.OpFirst, "x", "", false},
		{"x:last", insyra.OpLast, "x", "", false},
		{"count", insyra.OpCountAll, "", "count", false},
		{":countall:total", insyra.OpCountAll, "", "total", false},
		{"x:countall", insyra.OpCountAll, "x", "", false},

		{"badspec", 0, "", "", true},
		{"x:nope", 0, "", "", true},
		{":sum", 0, "", "", true},
	}
	for _, tc := range cases {
		cfg, err := parseAggregateSpec(tc.raw)
		if tc.fail {
			if err == nil {
				t.Errorf("expected error for spec %q", tc.raw)
			}
			continue
		}
		if err != nil {
			t.Errorf("spec %q: unexpected error: %v", tc.raw, err)
			continue
		}
		if cfg.Op != tc.op {
			t.Errorf("spec %q: op = %v want %v", tc.raw, cfg.Op, tc.op)
		}
		if cfg.SourceCol != tc.col {
			t.Errorf("spec %q: col = %q want %q", tc.raw, cfg.SourceCol, tc.col)
		}
		if cfg.As != tc.as {
			t.Errorf("spec %q: as = %q want %q", tc.raw, cfg.As, tc.as)
		}
	}
}
