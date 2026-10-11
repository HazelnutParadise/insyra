package commands

import (
	"math"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

// fillna ... missing nan on a table picks its columns the way the library's
// FillForward does: an unnamed column is filled, a column given by name is
// found, and a column that does not exist is skipped rather than failing the
// command.
func TestFillNATableMissingNaNResolvesColumnsLikeTheLibrary(t *testing.T) {
	nan := math.NaN()
	dt := insyra.NewDataTable(
		insyra.NewDataList(1.0, nan, nil),
		insyra.NewDataList(2.0, nan, 3.0).SetName("price"),
	)
	ctx := newTimeSeriesContext(t, map[string]any{"t": dt})
	if err := runFillNACommand(ctx, []string{"t", "ffill", "missing", "nan", "as", "all"}); err != nil {
		t.Fatalf("fillna over every column: %v", err)
	}
	all := ctx.Vars["all"].(*insyra.DataTable)
	if got := all.GetColByNumber(0).Data(); got[1] != 1.0 || got[2] != nil {
		t.Errorf("unnamed column = %v, want [1 1 nil]", got)
	}
	if got := all.GetColByNumber(1).Data(); got[1] != 2.0 {
		t.Errorf("price column = %v, want NaN filled with 2", got)
	}

	if err := runFillNACommand(ctx, []string{"t", "ffill", "missing", "nan", "cols", "price,nope", "as", "some"}); err != nil {
		t.Fatalf("fillna with a missing column: %v", err)
	}
	some := ctx.Vars["some"].(*insyra.DataTable)
	if got := some.GetColByNumber(1).Data(); got[1] != 2.0 {
		t.Errorf("price column = %v, want NaN filled with 2", got)
	}
	if got := some.GetColByNumber(0).Data(); !math.IsNaN(got[1].(float64)) {
		t.Errorf("unselected column = %v, want it untouched", got)
	}
}
