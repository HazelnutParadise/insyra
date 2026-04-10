package commands

import (
	"bytes"
	"testing"

	insyra "github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

func clusteringDataTable() *insyra.DataTable {
	return insyra.NewDataTable(
		insyra.NewDataList(0.0, 0.0, 1.0, 10.0, 10.0, 11.0).SetName("x"),
		insyra.NewDataList(0.0, 1.0, 0.0, 10.0, 11.0, 10.0).SetName("y"),
	)
}

func TestRunKMeansCommandStoresArtifacts(t *testing.T) {
	ctx := &ExecContext{
		Vars:   map[string]any{"t": clusteringDataTable()},
		Output: &bytes.Buffer{},
	}

	if err := runKMeansCommand(ctx, []string{"t", "2", "seed", "7", "as", "km"}); err != nil {
		t.Fatalf("runKMeansCommand failed: %v", err)
	}

	if _, ok := ctx.Vars["km"].(*insyra.DataList); !ok {
		t.Fatalf("expected km labels DataList, got %T", ctx.Vars["km"])
	}
	if _, ok := ctx.Vars["km_centers"].(*insyra.DataTable); !ok {
		t.Fatalf("expected km_centers DataTable, got %T", ctx.Vars["km_centers"])
	}
	if _, ok := ctx.Vars["km_size"].([]int); !ok {
		t.Fatalf("expected km_size []int, got %T", ctx.Vars["km_size"])
	}
}

func TestRunHClustAndCutTreeCommands(t *testing.T) {
	ctx := &ExecContext{
		Vars:   map[string]any{"t": clusteringDataTable()},
		Output: &bytes.Buffer{},
	}

	if err := runHClustCommand(ctx, []string{"t", "complete", "as", "hc"}); err != nil {
		t.Fatalf("runHClustCommand failed: %v", err)
	}
	tree, ok := ctx.Vars["hc"].(*stats.HierarchicalResult)
	if !ok || tree == nil {
		t.Fatalf("expected hc tree result, got %T", ctx.Vars["hc"])
	}

	if err := runCutTreeCommand(ctx, []string{"hc", "k", "2", "as", "labs"}); err != nil {
		t.Fatalf("runCutTreeCommand failed: %v", err)
	}
	if _, ok := ctx.Vars["labs"].(*insyra.DataList); !ok {
		t.Fatalf("expected labs DataList, got %T", ctx.Vars["labs"])
	}
}

func TestRunDBSCANAndSilhouetteCommands(t *testing.T) {
	ctx := &ExecContext{
		Vars: map[string]any{
			"t":      clusteringDataTable(),
			"labels": insyra.NewDataList(1, 1, 1, 2, 2, 2),
		},
		Output: &bytes.Buffer{},
	}

	if err := runDBSCANCommand(ctx, []string{"t", "1.5", "2", "as", "db"}); err != nil {
		t.Fatalf("runDBSCANCommand failed: %v", err)
	}
	if _, ok := ctx.Vars["db"].(*insyra.DataList); !ok {
		t.Fatalf("expected db labels DataList, got %T", ctx.Vars["db"])
	}
	if _, ok := ctx.Vars["db_isseed"].([]bool); !ok {
		t.Fatalf("expected db_isseed []bool, got %T", ctx.Vars["db_isseed"])
	}

	if err := runSilhouetteCommand(ctx, []string{"t", "labels", "as", "sil"}); err != nil {
		t.Fatalf("runSilhouetteCommand failed: %v", err)
	}
	if _, ok := ctx.Vars["sil"].(*insyra.DataList); !ok {
		t.Fatalf("expected sil widths DataList, got %T", ctx.Vars["sil"])
	}
	if _, ok := ctx.Vars["sil_avg"].(float64); !ok {
		t.Fatalf("expected sil_avg float64, got %T", ctx.Vars["sil_avg"])
	}
}
