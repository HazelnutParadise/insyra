package commands

import (
	"math"
	"testing"

	insyra "github.com/HazelnutParadise/insyra"
)

func scaleCLITable() *insyra.DataTable {
	return insyra.NewDataTable(
		insyra.NewDataList(1.0, 2.0, 3.0, 4.0).SetName("Age"),
		insyra.NewDataList(10.0, 20.0, 30.0, 40.0).SetName("Income"),
	)
}

func colFloats(t *testing.T, dt *insyra.DataTable, name string) []float64 {
	t.Helper()
	raw := dt.GetColByName(name).Data()
	out := make([]float64, len(raw))
	for i, v := range raw {
		f, ok := insyra.ToFloat64Safe(v)
		if !ok {
			t.Fatalf("non-numeric %v at %d in %s", v, i, name)
		}
		out[i] = f
	}
	return out
}

func TestScaleCLIFitTransformInverseFlow(t *testing.T) {
	ctx := newTestExecContext(t)
	ctx.Vars["train"] = scaleCLITable()

	if err := runScaleCommand(ctx, []string{"fit", "std", "sc", "train", "cols", "Age,Income"}); err != nil {
		t.Fatalf("fit: %v", err)
	}
	if _, ok := ctx.Vars["sc"].(insyra.Scaler); !ok {
		t.Fatalf("sc is not a scaler: %T", ctx.Vars["sc"])
	}

	if err := runScaleCommand(ctx, []string{"transform", "sc", "train", "as", "train_scaled"}); err != nil {
		t.Fatalf("transform: %v", err)
	}
	scaled, ok := ctx.Vars["train_scaled"].(*insyra.DataTable)
	if !ok {
		t.Fatalf("train_scaled is not a DataTable")
	}
	var sum float64
	for _, v := range colFloats(t, scaled, "Age") {
		sum += v
	}
	if math.Abs(sum/4) > 1e-9 {
		t.Fatalf("scaled Age mean = %v, want ~0", sum/4)
	}

	if err := runScaleCommand(ctx, []string{"inverse", "sc", "train_scaled", "as", "restored"}); err != nil {
		t.Fatalf("inverse: %v", err)
	}
	restored := ctx.Vars["restored"].(*insyra.DataTable)
	got := colFloats(t, restored, "Age")
	want := []float64{1, 2, 3, 4}
	for i := range want {
		if math.Abs(got[i]-want[i]) > 1e-9 {
			t.Fatalf("restored Age = %v, want %v", got, want)
		}
	}
}

func TestScaleCLIMinMaxRangeTrainTestNoLeakage(t *testing.T) {
	ctx := newTestExecContext(t)
	ctx.Vars["train"] = insyra.NewDataTable(insyra.NewDataList(0.0, 10.0).SetName("x"))
	ctx.Vars["test"] = insyra.NewDataTable(insyra.NewDataList(5.0, 20.0).SetName("x"))

	if err := runScaleCommand(ctx, []string{"fit", "minmax", "sc", "train", "range", "0", "1", "cols", "x"}); err != nil {
		t.Fatalf("fit minmax: %v", err)
	}
	if err := runScaleCommand(ctx, []string{"transform", "sc", "test", "as", "test_scaled"}); err != nil {
		t.Fatalf("transform test: %v", err)
	}
	out := ctx.Vars["test_scaled"].(*insyra.DataTable)
	got := colFloats(t, out, "x")
	// scaled with train min/max (0..10): 5 -> 0.5, 20 -> 2.0
	if math.Abs(got[0]-0.5) > 1e-9 || math.Abs(got[1]-2.0) > 1e-9 {
		t.Fatalf("test scaled = %v, want [0.5 2.0]", got)
	}
}

func TestScaleCLIErrors(t *testing.T) {
	ctx := newTestExecContext(t)
	ctx.Vars["train"] = scaleCLITable()

	// transform before fit -> scaler var not found
	if err := runScaleCommand(ctx, []string{"transform", "sc", "train", "as", "out"}); err == nil {
		t.Fatalf("expected error transforming with unknown scaler")
	}

	// fit without cols
	if err := runScaleCommand(ctx, []string{"fit", "std", "sc", "train"}); err == nil {
		t.Fatalf("expected error fitting without cols")
	}

	// fit on non-existent column
	if err := runScaleCommand(ctx, []string{"fit", "std", "sc", "train", "cols", "Nope"}); err == nil {
		t.Fatalf("expected error fitting unknown column")
	}

	// unknown method
	if err := runScaleCommand(ctx, []string{"fit", "bogus", "sc", "train", "cols", "Age"}); err == nil {
		t.Fatalf("expected error for unknown method")
	}

	// range on non-minmax
	if err := runScaleCommand(ctx, []string{"fit", "std", "sc", "train", "range", "0", "1", "cols", "Age"}); err == nil {
		t.Fatalf("expected error using range with std")
	}

	// unknown subcommand
	if err := runScaleCommand(ctx, []string{"bogus"}); err == nil {
		t.Fatalf("expected error for unknown subcommand")
	}
}

func TestScaleCLIShowAndVars(t *testing.T) {
	ctx := newTestExecContext(t)
	ctx.Vars["train"] = scaleCLITable()
	if err := runScaleCommand(ctx, []string{"fit", "robust", "sc", "train", "cols", "Age,Income"}); err != nil {
		t.Fatalf("fit: %v", err)
	}
	if err := runShowCommand(ctx, []string{"sc"}); err != nil {
		t.Fatalf("show scaler: %v", err)
	}
}
