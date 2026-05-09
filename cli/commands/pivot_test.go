package commands

import (
	"reflect"
	"testing"

	insyra "github.com/HazelnutParadise/insyra"
)

func pivotTestTable() *insyra.DataTable {
	region := insyra.NewDataList("APAC", "APAC", "EMEA").SetName("region")
	product := insyra.NewDataList("A", "B", "A").SetName("product")
	sales := insyra.NewDataList(10, 20, 30).SetName("sales")
	t := insyra.NewDataTable()
	t.AppendCols(region, product, sales)
	return t
}

func pivotDuplicateTestTable() *insyra.DataTable {
	region := insyra.NewDataList("APAC", "APAC", "APAC", "EMEA").SetName("region")
	product := insyra.NewDataList("A", "A", "B", "A").SetName("product")
	sales := insyra.NewDataList(10, 5, 20, 30).SetName("sales")
	t := insyra.NewDataTable()
	t.AppendCols(region, product, sales)
	return t
}

func TestRunPivotCommand_HappyPath(t *testing.T) {
	ctx := newTestExecContext(t)
	ctx.Vars["sales"] = pivotTestTable()
	err := runPivotCommand(ctx, []string{
		"sales",
		"index", "region",
		"columns", "product",
		"values", "sales",
		"fillna", "0",
		"sortcols", "true",
		"as", "wide",
	})
	if err != nil {
		t.Fatalf("runPivotCommand failed: %v", err)
	}
	wide, ok := ctx.Vars["wide"].(*insyra.DataTable)
	if !ok {
		t.Fatalf("expected wide DataTable, got %T", ctx.Vars["wide"])
	}
	if got := wide.ColNames(); !reflect.DeepEqual(got, []string{"region", "A", "B"}) {
		t.Errorf("headers = %v", got)
	}
	if wide.NumRows() != 2 {
		t.Errorf("rows = %d, want 2", wide.NumRows())
	}
}

func TestRunPivotCommand_AggSum(t *testing.T) {
	ctx := newTestExecContext(t)
	ctx.Vars["sales"] = pivotDuplicateTestTable()
	err := runPivotCommand(ctx, []string{
		"sales",
		"index", "region",
		"columns", "product",
		"values", "sales",
		"agg", "sum",
		"fillna", "0",
		"as", "wide",
	})
	if err != nil {
		t.Fatalf("runPivotCommand failed: %v", err)
	}
	wide := ctx.Vars["wide"].(*insyra.DataTable)
	colA := wide.GetColByName("A").Data()
	if got, _ := insyra.ToFloat64Safe(colA[0]); got != 15 {
		t.Errorf("APAC.A sum = %v, want 15", colA[0])
	}
}

func TestRunPivotCommand_DefaultAlias(t *testing.T) {
	ctx := newTestExecContext(t)
	ctx.Vars["sales"] = pivotTestTable()
	err := runPivotCommand(ctx, []string{
		"sales",
		"index", "region",
		"columns", "product",
		"values", "sales",
	})
	if err != nil {
		t.Fatalf("runPivotCommand failed: %v", err)
	}
	if _, ok := ctx.Vars["$result"].(*insyra.DataTable); !ok {
		t.Errorf("expected default $result alias, got %T", ctx.Vars["$result"])
	}
}

func TestRunPivotCommand_MissingRequiredOptions(t *testing.T) {
	ctx := newTestExecContext(t)
	ctx.Vars["sales"] = pivotTestTable()
	if err := runPivotCommand(ctx, []string{"sales", "columns", "product", "values", "sales"}); err == nil {
		t.Errorf("expected error when index is missing")
	}
	if err := runPivotCommand(ctx, []string{"sales", "index", "region", "values", "sales"}); err == nil {
		t.Errorf("expected error when columns is missing")
	}
	if err := runPivotCommand(ctx, []string{"sales", "index", "region", "columns", "product"}); err == nil {
		t.Errorf("expected error when values is missing")
	}
}

func TestRunPivotCommand_UnknownOption(t *testing.T) {
	ctx := newTestExecContext(t)
	ctx.Vars["sales"] = pivotTestTable()
	err := runPivotCommand(ctx, []string{
		"sales",
		"index", "region",
		"columns", "product",
		"values", "sales",
		"foo", "bar",
	})
	if err == nil {
		t.Errorf("expected error for unknown option")
	}
}

func TestRunPivotCommand_UnknownVar(t *testing.T) {
	ctx := newTestExecContext(t)
	err := runPivotCommand(ctx, []string{
		"missing",
		"index", "region",
		"columns", "product",
		"values", "sales",
	})
	if err == nil {
		t.Fatalf("expected error for unknown variable")
	}
}

func TestRunPivotCommand_DuplicateWithoutAggErrors(t *testing.T) {
	ctx := newTestExecContext(t)
	ctx.Vars["sales"] = pivotDuplicateTestTable()
	err := runPivotCommand(ctx, []string{
		"sales",
		"index", "region",
		"columns", "product",
		"values", "sales",
	})
	if err == nil {
		t.Errorf("expected error when duplicates need agg")
	}
}

func unpivotTestTable() *insyra.DataTable {
	id := insyra.NewDataList(1, 2).SetName("id")
	q1 := insyra.NewDataList(5, 6).SetName("Q1")
	q2 := insyra.NewDataList(4, 7).SetName("Q2")
	q3 := insyra.NewDataList(3, 8).SetName("Q3")
	t := insyra.NewDataTable()
	t.AppendCols(id, q1, q2, q3)
	return t
}

func TestRunUnpivotCommand_HappyPath(t *testing.T) {
	ctx := newTestExecContext(t)
	ctx.Vars["wide"] = unpivotTestTable()
	err := runUnpivotCommand(ctx, []string{
		"wide",
		"idvars", "id",
		"valuevars", "Q1,Q2,Q3",
		"varname", "question",
		"valuename", "score",
		"as", "long",
	})
	if err != nil {
		t.Fatalf("runUnpivotCommand failed: %v", err)
	}
	long := ctx.Vars["long"].(*insyra.DataTable)
	if long.NumRows() != 6 {
		t.Errorf("rows = %d, want 6", long.NumRows())
	}
	if got := long.ColNames(); !reflect.DeepEqual(got, []string{"id", "question", "score"}) {
		t.Errorf("headers = %v", got)
	}
}

func TestRunUnpivotCommand_DefaultsAndDropNA(t *testing.T) {
	ctx := newTestExecContext(t)
	ctx.Vars["wide"] = unpivotTestTable()
	err := runUnpivotCommand(ctx, []string{
		"wide",
		"idvars", "id",
		"dropna", "no",
		"as", "long",
	})
	if err != nil {
		t.Fatalf("runUnpivotCommand failed: %v", err)
	}
	long := ctx.Vars["long"].(*insyra.DataTable)
	if got := long.ColNames(); !reflect.DeepEqual(got, []string{"id", "variable", "value"}) {
		t.Errorf("headers = %v", got)
	}
}

func TestRunUnpivotCommand_MissingIDVars(t *testing.T) {
	ctx := newTestExecContext(t)
	ctx.Vars["wide"] = unpivotTestTable()
	if err := runUnpivotCommand(ctx, []string{"wide", "valuevars", "Q1"}); err == nil {
		t.Errorf("expected error when idvars missing")
	}
}

func TestRunUnpivotCommand_UnknownOption(t *testing.T) {
	ctx := newTestExecContext(t)
	ctx.Vars["wide"] = unpivotTestTable()
	err := runUnpivotCommand(ctx, []string{
		"wide",
		"idvars", "id",
		"foo", "bar",
	})
	if err == nil {
		t.Errorf("expected error for unknown option")
	}
}
