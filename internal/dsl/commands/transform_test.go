package commands

import (
	"bytes"
	"reflect"
	"testing"

	insyra "github.com/HazelnutParadise/insyra"
)

func TestRunRankCommandAscendingDefault(t *testing.T) {
	ctx := &ExecContext{
		Vars:   map[string]any{"x": insyra.NewDataList(10, 30, 20, 20)},
		Output: &bytes.Buffer{},
	}

	if err := runRankCommand(ctx, []string{"x", "as", "r"}); err != nil {
		t.Fatalf("runRankCommand failed: %v", err)
	}

	v, ok := ctx.Vars["r"]
	if !ok {
		t.Fatalf("expected alias variable r")
	}
	ranked, ok := v.(*insyra.DataList)
	if !ok {
		t.Fatalf("expected *DataList result, got %T", v)
	}

	if !reflect.DeepEqual(ranked.Data(), []any{1.0, 4.0, 2.5, 2.5}) {
		t.Fatalf("unexpected ascending rank result: %v", ranked.Data())
	}
}

func TestRunRankCommandDescending(t *testing.T) {
	ctx := &ExecContext{
		Vars:   map[string]any{"x": insyra.NewDataList(10, 30, 20, 20)},
		Output: &bytes.Buffer{},
	}

	if err := runRankCommand(ctx, []string{"x", "desc", "as", "r"}); err != nil {
		t.Fatalf("runRankCommand failed: %v", err)
	}

	v, ok := ctx.Vars["r"]
	if !ok {
		t.Fatalf("expected alias variable r")
	}
	ranked, ok := v.(*insyra.DataList)
	if !ok {
		t.Fatalf("expected *DataList result, got %T", v)
	}

	if !reflect.DeepEqual(ranked.Data(), []any{4.0, 1.0, 2.5, 2.5}) {
		t.Fatalf("unexpected descending rank result: %v", ranked.Data())
	}
}

func TestRunRankCommandInvalidDirection(t *testing.T) {
	ctx := &ExecContext{
		Vars:   map[string]any{"x": insyra.NewDataList(1, 2, 3)},
		Output: &bytes.Buffer{},
	}

	err := runRankCommand(ctx, []string{"x", "sideways"})
	if err == nil {
		t.Fatalf("expected error for invalid direction")
	}
}
