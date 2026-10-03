package commands

import (
	"path/filepath"
	"strings"
	"testing"

	insyra "github.com/HazelnutParadise/insyra"
)

// A chart the plot package cannot build reaches the user as that package's
// own error, naming the constructor and the reason, instead of a bare
// "failed to create chart".
func TestPlotReportsWhyTheChartCannotBeBuilt(t *testing.T) {
	ctx := tableCtx(t)
	ctx.Vars["empty"] = insyra.NewDataTable()
	err := Dispatch(ctx, "plot", []string{"line", "empty", "save", filepath.Join(t.TempDir(), "line.html")})
	if err == nil {
		t.Fatalf("plot drew a table with no columns; output was %q", outputOf(ctx))
	}
	if !strings.Contains(err.Error(), "CreateLineChart: no data to draw") {
		t.Errorf("error %q does not carry the constructor's reason", err)
	}
}

func TestPlotSavesAChart(t *testing.T) {
	ctx := tableCtx(t)
	ctx.Vars["series"] = insyra.NewDataList(1.0, 2.0, 3.0)
	path := filepath.Join(t.TempDir(), "bar.html")
	if err := Dispatch(ctx, "plot", []string{"bar", "series", "save", path}); err != nil {
		t.Fatalf("plot bar: %v", err)
	}
	if !strings.Contains(outputOf(ctx), "plot saved: "+path) {
		t.Errorf("output %q does not report the saved file", outputOf(ctx))
	}
}
