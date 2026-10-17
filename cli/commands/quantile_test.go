package commands

import (
	"bytes"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

// `quantile <var> <p>` takes p on the 0..1 scale and gives the same number as
// `percentile <var> <100p>`.
func TestQuantileCommandTakesAFraction(t *testing.T) {
	ctx := newTestExecContext(t)
	ctx.Vars["x"] = insyra.NewDataList(1, 2, 3, 4, 10)

	if err := Dispatch(ctx, "quantile", []string{"x", "0.75"}); err != nil {
		t.Fatalf("quantile: %v", err)
	}
	if err := Dispatch(ctx, "percentile", []string{"x", "75"}); err != nil {
		t.Fatalf("percentile: %v", err)
	}
	lines := strings.Fields(ctx.Output.(*bytes.Buffer).String())
	if len(lines) != 2 || lines[0] != "4" || lines[1] != "4" {
		t.Fatalf("quantile x 0.75 and percentile x 75 printed %q, want 4 and 4", lines)
	}
}
