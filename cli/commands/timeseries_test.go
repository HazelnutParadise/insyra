package commands

import (
	"bytes"
	"math"
	"reflect"
	"strings"
	"testing"

	insyra "github.com/HazelnutParadise/insyra"
)

// newWindowCtx builds an ExecContext seeded with a single DataList variable
// "x". Tests for the window commands all share this shape.
func newWindowCtx(values ...any) *ExecContext {
	return &ExecContext{
		Vars:   map[string]any{"x": insyra.NewDataList(values...)},
		Output: &bytes.Buffer{},
	}
}

// resultDL retrieves the named DataList result and fails the test if it is
// missing or of the wrong type.
func resultDL(t *testing.T, ctx *ExecContext, name string) *insyra.DataList {
	t.Helper()
	v, ok := ctx.Vars[name]
	if !ok {
		t.Fatalf("expected variable %q to be set", name)
	}
	dl, ok := v.(*insyra.DataList)
	if !ok {
		t.Fatalf("expected %q to be *DataList, got %T", name, v)
	}
	return dl
}

func resultDT(t *testing.T, ctx *ExecContext, name string) *insyra.DataTable {
	t.Helper()
	v, ok := ctx.Vars[name]
	if !ok {
		t.Fatalf("expected variable %q to be set", name)
	}
	dt, ok := v.(*insyra.DataTable)
	if !ok {
		t.Fatalf("expected %q to be *DataTable, got %T", name, v)
	}
	return dt
}

// approxEqualAny compares two slices element-wise; numeric values use a
// tolerance, nil compares to nil only.
func approxEqualAny(got, want []any, tol float64) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] == nil && want[i] == nil {
			continue
		}
		if got[i] == nil || want[i] == nil {
			return false
		}
		gf, gok := insyra.ToFloat64Safe(got[i])
		wf, wok := insyra.ToFloat64Safe(want[i])
		if gok && wok {
			if math.Abs(gf-wf) > tol {
				return false
			}
			continue
		}
		if !reflect.DeepEqual(got[i], want[i]) {
			return false
		}
	}
	return true
}

func TestShiftCommand_LagAndLead(t *testing.T) {
	ctx := newWindowCtx(10.0, 20.0, 30.0, 40.0, 50.0)
	if err := runShiftCommand(ctx, []string{"x", "1", "as", "lag"}); err != nil {
		t.Fatalf("shift lag failed: %v", err)
	}
	if !approxEqualAny(resultDL(t, ctx, "lag").Data(), []any{nil, 10.0, 20.0, 30.0, 40.0}, 1e-9) {
		t.Errorf("lag wrong: %v", resultDL(t, ctx, "lag").Data())
	}

	if err := runShiftCommand(ctx, []string{"x", "-1", "as", "lead"}); err != nil {
		t.Fatalf("shift lead failed: %v", err)
	}
	if !approxEqualAny(resultDL(t, ctx, "lead").Data(), []any{20.0, 30.0, 40.0, 50.0, nil}, 1e-9) {
		t.Errorf("lead wrong: %v", resultDL(t, ctx, "lead").Data())
	}
}

func TestShiftCommand_WithFill(t *testing.T) {
	ctx := newWindowCtx(10, 20, 30)
	if err := runShiftCommand(ctx, []string{"x", "2", "fill", "0", "as", "y"}); err != nil {
		t.Fatalf("shift fill failed: %v", err)
	}
	if !approxEqualAny(resultDL(t, ctx, "y").Data(), []any{0, 0, 10}, 1e-9) {
		t.Errorf("fill wrong: %v", resultDL(t, ctx, "y").Data())
	}
}

func TestShiftCommand_BadOption(t *testing.T) {
	ctx := newWindowCtx(1, 2, 3)
	if err := runShiftCommand(ctx, []string{"x", "1", "zoinks", "yo"}); err == nil {
		t.Error("expected unknown-option error, got nil")
	}
}

func TestDiffNCommand_PositivePeriods(t *testing.T) {
	ctx := newWindowCtx(10.0, 13.0, 18.0, 17.0, 25.0)
	if err := runDiffNCommand(ctx, []string{"x", "1", "as", "d"}); err != nil {
		t.Fatalf("diffn failed: %v", err)
	}
	if !approxEqualAny(resultDL(t, ctx, "d").Data(), []any{nil, 3.0, 5.0, -1.0, 8.0}, 1e-9) {
		t.Errorf("diffn(1) wrong: %v", resultDL(t, ctx, "d").Data())
	}
}

func TestDiffNCommand_RejectsZero(t *testing.T) {
	ctx := newWindowCtx(1.0, 2.0, 3.0)
	if err := runDiffNCommand(ctx, []string{"x", "0"}); err == nil {
		t.Error("expected error for periods=0")
	}
}

func TestPctChangeCommand(t *testing.T) {
	ctx := newWindowCtx(100.0, 110.0, 99.0)
	if err := runPctChangeCommand(ctx, []string{"x", "1", "as", "r"}); err != nil {
		t.Fatalf("pctchange failed: %v", err)
	}
	if !approxEqualAny(resultDL(t, ctx, "r").Data(), []any{nil, 0.1, -0.1}, 1e-9) {
		t.Errorf("pctchange wrong: %v", resultDL(t, ctx, "r").Data())
	}
}

func TestCumSumCommand(t *testing.T) {
	ctx := newWindowCtx(10.0, 20.0, 30.0)
	if err := runCumSumCommand(ctx, []string{"x", "as", "c"}); err != nil {
		t.Fatalf("cumsum failed: %v", err)
	}
	if !approxEqualAny(resultDL(t, ctx, "c").Data(), []any{10.0, 30.0, 60.0}, 1e-9) {
		t.Errorf("cumsum wrong: %v", resultDL(t, ctx, "c").Data())
	}
}

func TestCumMaxCommand(t *testing.T) {
	ctx := newWindowCtx(3.0, 1.0, 4.0, 1.0, 5.0, 9.0, 2.0)
	if err := runCumMaxCommand(ctx, []string{"x", "as", "m"}); err != nil {
		t.Fatalf("cummax failed: %v", err)
	}
	if !approxEqualAny(resultDL(t, ctx, "m").Data(), []any{3.0, 3.0, 4.0, 4.0, 5.0, 9.0, 9.0}, 1e-9) {
		t.Errorf("cummax wrong: %v", resultDL(t, ctx, "m").Data())
	}
}

func TestRollingCommand_Mean(t *testing.T) {
	ctx := newWindowCtx(1.0, 2.0, 3.0, 4.0, 5.0)
	if err := runRollingCommand(ctx, []string{"x", "3", "mean", "as", "ma3"}); err != nil {
		t.Fatalf("rolling mean failed: %v", err)
	}
	if !approxEqualAny(resultDL(t, ctx, "ma3").Data(), []any{nil, nil, 2.0, 3.0, 4.0}, 1e-9) {
		t.Errorf("rolling mean wrong: %v", resultDL(t, ctx, "ma3").Data())
	}
}

func TestRollingCommand_MinObs(t *testing.T) {
	ctx := newWindowCtx(1.0, 2.0, 3.0, 4.0, 5.0)
	if err := runRollingCommand(ctx, []string{"x", "3", "mean", "minobs", "1", "as", "soft"}); err != nil {
		t.Fatalf("rolling minobs failed: %v", err)
	}
	if !approxEqualAny(resultDL(t, ctx, "soft").Data(), []any{1.0, 1.5, 2.0, 3.0, 4.0}, 1e-9) {
		t.Errorf("rolling minobs wrong: %v", resultDL(t, ctx, "soft").Data())
	}
}

func TestRollingCommand_Center(t *testing.T) {
	ctx := newWindowCtx(1.0, 2.0, 3.0, 4.0, 5.0)
	if err := runRollingCommand(ctx, []string{"x", "3", "mean", "center", "yes", "as", "c"}); err != nil {
		t.Fatalf("rolling center failed: %v", err)
	}
	// pandas: nil, 2, 3, 4, nil
	if !approxEqualAny(resultDL(t, ctx, "c").Data(), []any{nil, 2.0, 3.0, 4.0, nil}, 1e-9) {
		t.Errorf("rolling center wrong: %v", resultDL(t, ctx, "c").Data())
	}
}

func TestRollingCommand_StdAndVar(t *testing.T) {
	ctx := newWindowCtx(1.0, 2.0, 3.0, 4.0, 5.0)
	if err := runRollingCommand(ctx, []string{"x", "3", "std", "as", "rs"}); err != nil {
		t.Fatalf("rolling std failed: %v", err)
	}
	if !approxEqualAny(resultDL(t, ctx, "rs").Data(), []any{nil, nil, 1.0, 1.0, 1.0}, 1e-9) {
		t.Errorf("rolling std wrong: %v", resultDL(t, ctx, "rs").Data())
	}
	if err := runRollingCommand(ctx, []string{"x", "3", "var", "as", "rv"}); err != nil {
		t.Fatalf("rolling var failed: %v", err)
	}
	if !approxEqualAny(resultDL(t, ctx, "rv").Data(), []any{nil, nil, 1.0, 1.0, 1.0}, 1e-9) {
		t.Errorf("rolling var wrong: %v", resultDL(t, ctx, "rv").Data())
	}
}

func TestRollingCommand_BadReducer(t *testing.T) {
	ctx := newWindowCtx(1.0, 2.0, 3.0)
	if err := runRollingCommand(ctx, []string{"x", "2", "bogus"}); err == nil {
		t.Error("expected unknown-reducer error")
	}
}

func TestExpandingCommand_Mean(t *testing.T) {
	ctx := newWindowCtx(1.0, 2.0, 3.0, 4.0)
	if err := runExpandingCommand(ctx, []string{"x", "1", "mean", "as", "e"}); err != nil {
		t.Fatalf("expanding mean failed: %v", err)
	}
	if !approxEqualAny(resultDL(t, ctx, "e").Data(), []any{1.0, 1.5, 2.0, 2.5}, 1e-9) {
		t.Errorf("expanding mean wrong: %v", resultDL(t, ctx, "e").Data())
	}
}

func TestExpandingCommand_MinObsThreshold(t *testing.T) {
	ctx := newWindowCtx(1.0, 2.0, 3.0, 4.0)
	if err := runExpandingCommand(ctx, []string{"x", "3", "mean", "as", "e"}); err != nil {
		t.Fatalf("expanding minobs failed: %v", err)
	}
	if !approxEqualAny(resultDL(t, ctx, "e").Data(), []any{nil, nil, 2.0, 2.5}, 1e-9) {
		t.Errorf("expanding minobs wrong: %v", resultDL(t, ctx, "e").Data())
	}
}

func TestFillNACommand_Interpolate(t *testing.T) {
	ctx := newWindowCtx(nil, 1.0, nil, 3.0, math.NaN())
	if err := runFillNACommand(ctx, []string{"x", "interpolate", "extrapolate", "yes", "as", "filled"}); err != nil {
		t.Fatalf("fillna interpolate failed: %v", err)
	}
	if !approxEqualAny(resultDL(t, ctx, "filled").Data(), []any{0.0, 1.0, 2.0, 3.0, 4.0}, 1e-9) {
		t.Errorf("fillna interpolate wrong: %v", resultDL(t, ctx, "filled").Data())
	}
}

func TestFillNACommand_MissingNaNPreservesNil(t *testing.T) {
	ctx := newWindowCtx(1.0, math.NaN(), nil, 4.0)
	if err := runFillNACommand(ctx, []string{"x", "ffill", "missing", "nan", "as", "filled"}); err != nil {
		t.Fatalf("fillna missing=nan failed: %v", err)
	}
	if !approxEqualAny(resultDL(t, ctx, "filled").Data(), []any{1.0, 1.0, nil, 4.0}, 1e-9) {
		t.Errorf("fillna missing=nan wrong: %v", resultDL(t, ctx, "filled").Data())
	}
}

func TestFillNACommand_MissingNilPreservesNaN(t *testing.T) {
	ctx := newWindowCtx(1.0, math.NaN(), nil, 4.0)
	if err := runFillNACommand(ctx, []string{"x", "ffill", "missing", "nil", "as", "filled"}); err != nil {
		t.Fatalf("fillna missing=nil failed: %v", err)
	}
	got := resultDL(t, ctx, "filled").Data()
	if len(got) != 4 || got[0] != 1.0 || got[2] != 1.0 || got[3] != 4.0 {
		t.Errorf("fillna missing=nil wrong shape: %v", got)
	}
	if f, ok := got[1].(float64); !ok || !math.IsNaN(f) {
		t.Errorf("fillna missing=nil should keep NaN at index 1, got %v", got[1])
	}
}

func TestFillNACommand_TableMedianCols(t *testing.T) {
	ctx := &ExecContext{
		Vars: map[string]any{
			"t": insyra.NewDataTable(
				insyra.NewDataList(1.0, nil, 3.0).SetName("num"),
				insyra.NewDataList("x", nil, "z").SetName("text"),
			),
		},
		Output: &bytes.Buffer{},
	}
	if err := runFillNACommand(ctx, []string{"t", "median", "cols", "num,text", "as", "filled"}); err != nil {
		t.Fatalf("fillna median failed: %v", err)
	}
	result := resultDT(t, ctx, "filled")
	if !approxEqualAny(result.GetColByName("num").Data(), []any{1.0, 2.0, 3.0}, 1e-9) {
		t.Errorf("fillna numeric col wrong: %v", result.GetColByName("num").Data())
	}
	if !approxEqualAny(result.GetColByName("text").Data(), []any{"x", nil, "z"}, 1e-9) {
		t.Errorf("fillna string col should be skipped: %v", result.GetColByName("text").Data())
	}
}

func TestFillNaNLegacy_MeanOnly(t *testing.T) {
	ctx := newWindowCtx(1.0, math.NaN(), 3.0)
	if err := runFillNaNCommand(ctx, []string{"x", "mean", "as", "filled"}); err != nil {
		t.Fatalf("fillnan mean failed: %v", err)
	}
	if !approxEqualAny(resultDL(t, ctx, "filled").Data(), []any{1.0, 2.0, 3.0}, 1e-9) {
		t.Errorf("fillnan mean wrong: %v", resultDL(t, ctx, "filled").Data())
	}
	out := ctx.Output.(*bytes.Buffer).String()
	if !strings.Contains(out, "deprecated") || !strings.Contains(out, "fillna") {
		t.Errorf("expected deprecation warning pointing to fillna, got: %q", out)
	}
}

func TestFillNaNLegacy_RejectsOtherStrategies(t *testing.T) {
	ctx := newWindowCtx(1.0, math.NaN(), 3.0)
	if err := runFillNaNCommand(ctx, []string{"x", "median"}); err == nil {
		t.Error("expected error for non-mean strategy on legacy fillnan")
	}
}

// Variable-resolution / type-check parity with existing commands.

func TestShiftCommand_MissingVar(t *testing.T) {
	ctx := &ExecContext{Vars: map[string]any{}, Output: &bytes.Buffer{}}
	if err := runShiftCommand(ctx, []string{"missing", "1"}); err == nil {
		t.Error("expected missing-var error")
	}
}

func TestShiftCommand_WrongType(t *testing.T) {
	ctx := &ExecContext{
		Vars:   map[string]any{"t": insyra.NewDataTable()},
		Output: &bytes.Buffer{},
	}
	if err := runShiftCommand(ctx, []string{"t", "1"}); err == nil {
		t.Error("expected wrong-type error")
	}
}
