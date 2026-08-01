// Package mltest contains checks for implementations of the ml protocol.
package mltest

import (
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/ml"
)

// RunConformance exercises the protocol rules against a fitted model.
//
// A model that also implements ml.ProbaModel is held to the ordering rule by
// value, not only by column name: the class Predict returns for a row must be
// the class whose probability column holds that row's largest probability.
// Ties are allowed. A model that decides its class by any rule other than the
// largest probability does not satisfy this check.
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
	// The renamed column is a superstring of the feature it replaced, so every
	// mention of it is stripped before looking for the missing feature. Without
	// that, an error naming only the incoming column would satisfy the check.
	if _, err := model.Predict(renamed); err == nil || !contains(strings.ReplaceAll(err.Error(), renamedNames[0], ""), features[0]) {
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
		if probabilities.NumRows() != x.NumRows() {
			t.Fatalf("PredictProba returned %d rows, want %d", probabilities.NumRows(), x.NumRows())
		}
		columns := make([][]float64, probabilities.NumCols())
		for col := range columns {
			columns[col] = probabilities.GetColByNumber(col).ToF64Slice()
		}
		for row := 0; row < probabilities.NumRows(); row++ {
			sum := 0.0
			for col := 0; col < probabilities.NumCols(); col++ {
				sum += columns[col][row]
			}
			if math.Abs(sum-1) > 1e-12 {
				t.Fatalf("probability row %d sums to %.17g, want 1", row, sum)
			}
		}
		// Matching column names against class names only proves the columns are
		// labelled in class order. It cannot see values that have been filled in
		// under the wrong label, because a model that derives its column names
		// from its own Classes() list satisfies the name check by construction.
		// Tying each row's predicted class back to its own column is what makes
		// the ordering rule bite.
		classIndex := make(map[string]int, classes.Len())
		for i := 0; i < classes.Len(); i++ {
			classIndex[fmt.Sprint(classes.Get(i))] = i
		}
		predicted := got.Data()
		for row := 0; row < probabilities.NumRows(); row++ {
			label := fmt.Sprint(predicted[row])
			index, ok := classIndex[label]
			if !ok {
				t.Fatalf("Predict returned %q at row %d, which is not one of the classes %v", label, row, classes.Data())
			}
			chosen := columns[index][row]
			for col := 0; col < probabilities.NumCols(); col++ {
				if columns[col][row] > chosen+1e-12 {
					t.Fatalf("row %d: Predict chose class %q whose column holds %.17g, but column %d (class %q) holds a larger %.17g",
						row, label, chosen, col, fmt.Sprint(classes.Get(col)), columns[col][row])
				}
			}
		}
	}

	// The labels the model was fitted on. `Classifier` is documented as
	// predicting "one of a known set of classes" (ml/interfaces.go:22), and a
	// set that omits a label the model was trained on is not that set — the
	// model can never predict it. This is the only thing y is here for, and
	// checking it is what stops the parameter from being decoration.
	if classifier, ok := model.(ml.Classifier); ok && y != nil {
		classes := classifier.Classes()
		known := make(map[string]struct{}, classes.Len())
		for i := 0; i < classes.Len(); i++ {
			known[fmt.Sprint(classes.Get(i))] = struct{}{}
		}
		seen := make(map[string]struct{}, y.Len())
		for i := 0; i < y.Len(); i++ {
			label := fmt.Sprint(y.Get(i))
			if _, dup := seen[label]; dup {
				continue
			}
			seen[label] = struct{}{}
			if _, ok := known[label]; !ok {
				t.Fatalf("Classes() omits %q, which the model was fitted on; it can never predict that label", label)
			}
		}
	}
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
