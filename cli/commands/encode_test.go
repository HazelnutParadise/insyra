package commands

import (
	"reflect"
	"strings"
	"testing"

	insyra "github.com/HazelnutParadise/insyra"
)

func encodeCommandTestTable() *insyra.DataTable {
	return insyra.NewDataTable(
		insyra.NewDataList("red", "blue", "red").SetName("color"),
		insyra.NewDataList("S", "M", "S").SetName("size"),
		insyra.NewDataList(10, 20, 30).SetName("value"),
	)
}

func assertEncodeCommandData(t *testing.T, dl *insyra.DataList, want []any) {
	t.Helper()
	if dl == nil {
		t.Fatalf("got nil DataList")
	}
	if got := dl.Data(); !reflect.DeepEqual(got, want) {
		t.Fatalf("data = %#v, want %#v", got, want)
	}
}

func TestRunEncodeCommandOneHot(t *testing.T) {
	ctx := newTestExecContext(t)
	ctx.Vars["sales"] = encodeCommandTestTable()
	err := runEncodeCommand(ctx, []string{
		"sales",
		"onehot",
		"color",
		"dropfirst", "true",
		"as", "x",
	})
	if err != nil {
		t.Fatalf("runEncodeCommand onehot failed: %v", err)
	}
	got, ok := ctx.Vars["x"].(*insyra.DataTable)
	if !ok {
		t.Fatalf("expected DataTable, got %T", ctx.Vars["x"])
	}
	if cols := got.ColNames(); !reflect.DeepEqual(cols, []string{"color_blue", "size", "value"}) {
		t.Fatalf("columns = %#v", cols)
	}
	assertEncodeCommandData(t, got.GetColByName("color_blue"), []any{0, 1, 0})
}

func TestRunEncodeCommandLabel(t *testing.T) {
	ctx := newTestExecContext(t)
	ctx.Vars["sales"] = encodeCommandTestTable()
	err := runEncodeCommand(ctx, []string{
		"sales",
		"label",
		"color",
		"newcol", "color_id",
		"sortby", "lex",
		"keeporiginal", "true",
		"as", "labeled",
	})
	if err != nil {
		t.Fatalf("runEncodeCommand label failed: %v", err)
	}
	got := ctx.Vars["labeled"].(*insyra.DataTable)
	if cols := got.ColNames(); !reflect.DeepEqual(cols, []string{"color", "color_id", "size", "value"}) {
		t.Fatalf("columns = %#v", cols)
	}
	assertEncodeCommandData(t, got.GetColByName("color_id"), []any{1, 0, 1})
}

func TestRunEncodeCommandOrdinal(t *testing.T) {
	ctx := newTestExecContext(t)
	ctx.Vars["sales"] = encodeCommandTestTable()
	err := runEncodeCommand(ctx, []string{
		"sales",
		"ordinal",
		"size",
		"order", "S,M,L",
		"newcol", "size_rank",
		"unknown", "error",
		"as", "ranked",
	})
	if err != nil {
		t.Fatalf("runEncodeCommand ordinal failed: %v", err)
	}
	got := ctx.Vars["ranked"].(*insyra.DataTable)
	if cols := got.ColNames(); !reflect.DeepEqual(cols, []string{"color", "size_rank", "value"}) {
		t.Fatalf("columns = %#v", cols)
	}
	assertEncodeCommandData(t, got.GetColByName("size_rank"), []any{0, 1, 0})
}

func TestRunEncodeCommandErrors(t *testing.T) {
	ctx := newTestExecContext(t)
	ctx.Vars["sales"] = encodeCommandTestTable()
	if err := runEncodeCommand(ctx, []string{"sales", "onehot", "color", "bogus", "x"}); err == nil || !strings.Contains(err.Error(), "unknown option") {
		t.Fatalf("expected unknown option error, got %v", err)
	}
	if err := runEncodeCommand(ctx, []string{"sales", "label", "color", "sortby", "bad"}); err == nil || !strings.Contains(err.Error(), "invalid value for sortby") {
		t.Fatalf("expected bad sortby error, got %v", err)
	}
	if err := runEncodeCommand(ctx, []string{"sales", "ordinal", "size", "order", "S,M", "unknown", "new"}); err == nil || !strings.Contains(err.Error(), "invalid value for unknown") {
		t.Fatalf("expected bad ordinal unknown error, got %v", err)
	}
	if err := runEncodeCommand(ctx, []string{"sales", "onehot", "color", "nan", "bad"}); err == nil || !strings.Contains(err.Error(), "invalid value for nan") {
		t.Fatalf("expected bad nan error, got %v", err)
	}
}
