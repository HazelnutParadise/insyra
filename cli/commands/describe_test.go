package commands

import (
	"bytes"
	"strings"
	"testing"

	insyra "github.com/HazelnutParadise/insyra"
)

func describeTestTable() *insyra.DataTable {
	return insyra.NewDataTable(
		insyra.NewDataList("east", "east", "west").SetName("region"),
		insyra.NewDataList(10, 20, 30).SetName("revenue"),
		insyra.NewDataList("retail", "online", "retail").SetName("segment"),
	)
}

func TestRunDescribeCommand_DataTableAsAlias(t *testing.T) {
	ctx := &ExecContext{Vars: map[string]any{"sales": describeTestTable()}, Output: &bytes.Buffer{}}

	if err := runDescribeCommand(ctx, []string{"sales", "as", "summary"}); err != nil {
		t.Fatalf("runDescribeCommand failed: %v", err)
	}

	out, ok := ctx.Vars["summary"].(*insyra.DataTable)
	if !ok {
		t.Fatalf("expected summary DataTable, got %T", ctx.Vars["summary"])
	}
	if got := out.ColNames(); len(got) != 1 || got[0] != "revenue" {
		t.Fatalf("expected only revenue by default, got %v", got)
	}
}

func TestRunDescribeCommand_DefaultResultAndPercentiles(t *testing.T) {
	ctx := &ExecContext{Vars: map[string]any{"x": insyra.NewDataList(1, 2, 3, 4, 5)}, Output: &bytes.Buffer{}}

	if err := runDescribeCommand(ctx, []string{"x", "percentiles", "0.1,0.9"}); err != nil {
		t.Fatalf("runDescribeCommand failed: %v", err)
	}

	out, ok := ctx.Vars["$result"].(*insyra.DataTable)
	if !ok {
		t.Fatalf("expected $result DataTable, got %T", ctx.Vars["$result"])
	}
	if _, ok := out.GetRowIndexByName("10%"); !ok {
		t.Fatal("expected 10% row")
	}
	if !strings.Contains(ctx.Output.(*bytes.Buffer).String(), "saved description as $result") {
		t.Fatal("expected save message for $result")
	}
}

func TestRunDescribeCommand_GroupByIncludeAll(t *testing.T) {
	ctx := &ExecContext{Vars: map[string]any{"sales": describeTestTable()}, Output: &bytes.Buffer{}}

	if err := runDescribeCommand(ctx, []string{"sales", "by", "region", "all", "true", "as", "summary"}); err != nil {
		t.Fatalf("runDescribeCommand failed: %v", err)
	}

	out := ctx.Vars["summary"].(*insyra.DataTable)
	if got := out.GetColByName("segment_unique").Data(); got[0] != 2 || got[1] != 1 {
		t.Fatalf("unexpected segment_unique: %v", got)
	}
}

func TestRunDescribeCommand_Errors(t *testing.T) {
	ctx := &ExecContext{Vars: map[string]any{"x": insyra.NewDataList(1, 2, 3)}, Output: &bytes.Buffer{}}
	if err := runDescribeCommand(ctx, []string{"missing"}); err == nil {
		t.Fatal("expected unknown variable error")
	}
	if err := runDescribeCommand(ctx, []string{"x", "by", "region"}); err == nil {
		t.Fatal("expected DataList by error")
	}
	if err := runDescribeCommand(ctx, []string{"x", "percentiles", "1.5"}); err == nil {
		t.Fatal("expected invalid percentile error")
	}
	if err := runDescribeCommand(ctx, []string{"x", "all"}); err == nil {
		t.Fatal("expected missing option value error")
	}
}
