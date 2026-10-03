package commands

import (
	"bytes"
	"testing"

	insyra "github.com/HazelnutParadise/insyra"
)

func knnTrainTable() *insyra.DataTable {
	return insyra.NewDataTable(
		insyra.NewDataList(0.0, 0.0, 1.0, 10.0, 10.0, 11.0).SetName("x"),
		insyra.NewDataList(0.0, 1.0, 0.0, 10.0, 11.0, 10.0).SetName("y"),
	)
}

func knnTestTable() *insyra.DataTable {
	return insyra.NewDataTable(
		insyra.NewDataList(0.1, 10.2).SetName("x"),
		insyra.NewDataList(0.2, 10.1).SetName("y"),
	)
}

func TestRunKNNCommands(t *testing.T) {
	ctx := &ExecContext{
		Vars: map[string]any{
			"train":  knnTrainTable(),
			"test":   knnTestTable(),
			"labels": insyra.NewDataList("red", "red", "red", "blue", "blue", "blue"),
			"target": insyra.NewDataList(1.0, 1.5, 1.2, 9.0, 9.5, 9.2),
		},
		Output: &bytes.Buffer{},
	}

	if err := runKNNClassifyCommand(ctx, []string{"train", "labels", "test", "3", "weighting", "distance", "algorithm", "kd_tree", "as", "cls"}); err != nil {
		t.Fatalf("runKNNClassifyCommand failed: %v", err)
	}
	if _, ok := ctx.Vars["cls"].(*insyra.DataList); !ok {
		t.Fatalf("expected cls DataList, got %T", ctx.Vars["cls"])
	}
	if _, ok := ctx.Vars["cls_probs"].(*insyra.DataTable); !ok {
		t.Fatalf("expected cls_probs DataTable, got %T", ctx.Vars["cls_probs"])
	}
	if _, ok := ctx.Vars["cls_classes"].(*insyra.DataList); !ok {
		t.Fatalf("expected cls_classes DataList, got %T", ctx.Vars["cls_classes"])
	}

	if err := runKNNRegressCommand(ctx, []string{"train", "target", "test", "2", "algorithm", "ball_tree", "as", "reg"}); err != nil {
		t.Fatalf("runKNNRegressCommand failed: %v", err)
	}
	if _, ok := ctx.Vars["reg"].(*insyra.DataList); !ok {
		t.Fatalf("expected reg DataList, got %T", ctx.Vars["reg"])
	}

	if err := runKNNNeighborsCommand(ctx, []string{"train", "test", "2", "algorithm", "kd_tree", "as", "nbr"}); err != nil {
		t.Fatalf("runKNNNeighborsCommand failed: %v", err)
	}
	if _, ok := ctx.Vars["nbr"].(*insyra.DataTable); !ok {
		t.Fatalf("expected nbr DataTable, got %T", ctx.Vars["nbr"])
	}
	if _, ok := ctx.Vars["nbr_distances"].(*insyra.DataTable); !ok {
		t.Fatalf("expected nbr_distances DataTable, got %T", ctx.Vars["nbr_distances"])
	}
}
