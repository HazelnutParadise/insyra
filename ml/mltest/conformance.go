// Package mltest contains checks for implementations of the ml protocol.
package mltest

import (
	"fmt"
	"math"
	"reflect"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/ml"
)

// RunConformance exercises the protocol rules against a fitted model.
func RunConformance(t *testing.T, model ml.Model, x *insyra.DataTable, y *insyra.DataList) {
	t.Helper()
	if model == nil {
		t.Fatal("model must not be nil")
	}
	if x == nil {
		t.Fatal("training table must not be nil")
	}

	features := model.Features()
	if len(features) == 0 {
		t.Fatal("Features must not be empty")
	}
	seen := make(map[string]struct{}, len(features))
	for _, feature := range features {
		if _, ok := seen[feature]; ok {
			t.Fatalf("Features contains duplicate %q", feature)
		}
		seen[feature] = struct{}{}
	}

	got, err := model.Predict(x)
	if err != nil {
		t.Fatalf("Predict(training): %v", err)
	}
	if got.Len() != x.NumRows() {
		t.Fatalf("Predict(training) returned %d rows, want %d", got.Len(), x.NumRows())
	}

	renamed := x.Clone()
	renamedNames := append([]string(nil), features...)
	renamedNames[0] += "_renamed"
	renamed.SetColNames(renamedNames)
	if _, err := model.Predict(renamed); err == nil || !contains(err.Error(), features[0]) {
		t.Fatalf("Predict(renamed column) error = %v, want an error naming %q", err, features[0])
	}

	reordered := reorderByFeatures(x, features)
	reorderedPrediction, err := model.Predict(reordered)
	if err != nil {
		t.Fatalf("Predict(reordered): %v", err)
	}
	if !reflect.DeepEqual(got.Data(), reorderedPrediction.Data()) {
		t.Fatalf("reordering input columns changed predictions: original=%v reordered=%v", got.Data(), reorderedPrediction.Data())
	}

	extra := x.Clone().AppendCols(insyra.NewDataList(make([]any, x.NumRows())...).SetName("__extra"))
	extraPrediction, err := model.Predict(extra)
	if err != nil {
		t.Fatalf("Predict(extra column): %v", err)
	}
	if !reflect.DeepEqual(got.Data(), extraPrediction.Data()) {
		t.Fatalf("extra input columns changed predictions: original=%v extra=%v", got.Data(), extraPrediction.Data())
	}

	if proba, ok := model.(ml.ProbaModel); ok {
		classes := proba.Classes()
		if classes == nil || classes.Len() == 0 {
			t.Fatal("ProbaModel.Classes must not be empty")
		}
		probabilities, err := proba.PredictProba(x)
		if err != nil {
			t.Fatalf("PredictProba: %v", err)
		}
		if probabilities.NumCols() != classes.Len() {
			t.Fatalf("PredictProba returned %d columns, want %d classes", probabilities.NumCols(), classes.Len())
		}
		for i := 0; i < classes.Len(); i++ {
			wantName := fmt.Sprint(classes.Get(i))
			if gotName := probabilities.ColNames()[i]; gotName != wantName {
				t.Fatalf("probability column %d = %q, want class %q", i, gotName, wantName)
			}
		}
		for row := 0; row < probabilities.NumRows(); row++ {
			sum := 0.0
			for col := 0; col < probabilities.NumCols(); col++ {
				sum += probabilities.GetColByNumber(col).ToF64Slice()[row]
			}
			if math.Abs(sum-1) > 1e-12 {
				t.Fatalf("probability row %d sums to %.17g, want 1", row, sum)
			}
		}
	}

	_ = y
}

func reorderByFeatures(x *insyra.DataTable, features []string) *insyra.DataTable {
	columns := make([]*insyra.DataList, len(features))
	for i := range features {
		columns[i] = x.GetColByName(features[len(features)-1-i]).Clone().SetName(features[len(features)-1-i])
	}
	return insyra.NewDataTable(columns...)
}

func contains(value, want string) bool {
	for i := 0; i+len(want) <= len(value); i++ {
		if value[i:i+len(want)] == want {
			return true
		}
	}
	return false
}
