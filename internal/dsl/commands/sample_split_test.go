package commands

import (
	"bytes"
	"testing"

	insyra "github.com/HazelnutParadise/insyra"
)

func TestRunSampleCommand_LegacyTableSyntax(t *testing.T) {
	ctx := &ExecContext{
		Vars:   map[string]any{"t": cliSamplingTable()},
		Output: &bytes.Buffer{},
	}
	if err := runSampleCommand(ctx, []string{"t", "2", "as", "out"}); err != nil {
		t.Fatalf("runSampleCommand error: %v", err)
	}
	out, ok := ctx.Vars["out"].(*insyra.DataTable)
	if !ok {
		t.Fatalf("expected DataTable, got %T", ctx.Vars["out"])
	}
	if out.NumRows() != 2 {
		t.Fatalf("sample rows = %d, want 2", out.NumRows())
	}
}

func TestRunSampleCommand_FracSeedReproducible(t *testing.T) {
	ctx := &ExecContext{
		Vars:   map[string]any{"t": cliSamplingTable()},
		Output: &bytes.Buffer{},
	}
	if err := runSampleCommand(ctx, []string{"t", "frac", "0.5", "seed", "42", "as", "a"}); err != nil {
		t.Fatalf("sample a error: %v", err)
	}
	if err := runSampleCommand(ctx, []string{"t", "frac", "0.5", "seed", "42", "as", "b"}); err != nil {
		t.Fatalf("sample b error: %v", err)
	}
	a := ctx.Vars["a"].(*insyra.DataTable)
	b := ctx.Vars["b"].(*insyra.DataTable)
	if a.NumRows() != 2 {
		t.Fatalf("sample rows = %d, want 2", a.NumRows())
	}
	for i := range a.NumRows() {
		if a.GetElementByNumberIndex(i, 0) != b.GetElementByNumberIndex(i, 0) {
			t.Fatalf("seeded frac samples differ")
		}
	}
}

func TestRunSampleCommand_ShuffleDataList(t *testing.T) {
	ctx := &ExecContext{
		Vars:   map[string]any{"x": insyra.NewDataList(1, 2, 3, 4)},
		Output: &bytes.Buffer{},
	}
	if err := runSampleCommand(ctx, []string{"x", "shuffle", "seed", "42", "as", "out"}); err != nil {
		t.Fatalf("shuffle error: %v", err)
	}
	out, ok := ctx.Vars["out"].(*insyra.DataList)
	if !ok {
		t.Fatalf("expected DataList, got %T", ctx.Vars["out"])
	}
	if out.Len() != 4 {
		t.Fatalf("shuffle len = %d, want 4", out.Len())
	}
}

func TestRunSplitCommand_SeededAndPreserveOrder(t *testing.T) {
	ctx := &ExecContext{
		Vars:   map[string]any{"t": cliSamplingTable()},
		Output: &bytes.Buffer{},
	}
	if err := runSplitCommand(ctx, []string{"t", "train", "0.8", "seed", "42", "as", "train", "test"}); err != nil {
		t.Fatalf("split error: %v", err)
	}
	train := ctx.Vars["train"].(*insyra.DataTable)
	test := ctx.Vars["test"].(*insyra.DataTable)
	if train.NumRows() != 4 || test.NumRows() != 1 {
		t.Fatalf("split rows = (%d,%d), want (4,1)", train.NumRows(), test.NumRows())
	}

	if err := runSplitCommand(ctx, []string{"t", "train", "0.8", "shuffle", "false", "as", "orderedTrain", "orderedTest"}); err != nil {
		t.Fatalf("ordered split error: %v", err)
	}
	orderedTrain := ctx.Vars["orderedTrain"].(*insyra.DataTable)
	orderedTest := ctx.Vars["orderedTest"].(*insyra.DataTable)
	if got := orderedTrain.GetElementByNumberIndex(0, 0); got != 1 {
		t.Fatalf("preserve-order train first id = %v, want 1", got)
	}
	if got := orderedTest.GetElementByNumberIndex(0, 0); got != 5 {
		t.Fatalf("preserve-order test first id = %v, want 5", got)
	}
}

func cliSamplingTable() *insyra.DataTable {
	dt := insyra.NewDataTable()
	dt.AppendCols(
		insyra.NewDataList(1, 2, 3, 4, 5).SetName("id"),
		insyra.NewDataList(10, 20, 30, 40, 50).SetName("value"),
	)
	dt.SetRowNames([]string{"r1", "r2", "r3", "r4", "r5"})
	return dt
}
