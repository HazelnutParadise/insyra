package ml_test

import (
	"bytes"
	"io"
	"testing"

	"github.com/HazelnutParadise/insyra/ml"
)

type exportableImportances interface {
	ml.Importances
	ExportONNX(w io.Writer) error
}

// The tree and ensemble Fit functions have always returned concrete pointers,
// so a failed fit reached through an interface is a nil pointer there too.
// Every method must answer it with an error or an empty value.
func TestTreeModelsAreSafeWhenNil(t *testing.T) {
	table := testFeatures().table
	var (
		treeClassifier    *ml.DecisionTreeClassifier
		treeRegressor     *ml.DecisionTreeRegressor
		forestClassifier  *ml.RandomForestClassifier
		forestRegressor   *ml.RandomForestRegressor
		boostedClassifier *ml.GradientBoostingClassifier
		boostedRegressor  *ml.GradientBoostingRegressor
	)
	models := map[string]exportableImportances{
		"DecisionTreeClassifier":     treeClassifier,
		"DecisionTreeRegressor":      treeRegressor,
		"RandomForestClassifier":     forestClassifier,
		"RandomForestRegressor":      forestRegressor,
		"GradientBoostingClassifier": boostedClassifier,
		"GradientBoostingRegressor":  boostedRegressor,
	}
	classifiers := map[string]ml.ProbaModel{
		"DecisionTreeClassifier":     treeClassifier,
		"RandomForestClassifier":     forestClassifier,
		"GradientBoostingClassifier": boostedClassifier,
	}
	noPanic := func(t *testing.T, call func()) {
		t.Helper()
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("panicked on a nil model: %v", r)
			}
		}()
		call()
	}
	for name, model := range models {
		t.Run(name, func(t *testing.T) {
			noPanic(t, func() {
				if got := model.Features(); got != nil {
					t.Errorf("Features() = %v, want nil", got)
				}
				if got := model.FeatureImportances(); got != nil {
					t.Errorf("FeatureImportances() = %v, want nil", got)
				}
				if _, err := model.Predict(table); err == nil {
					t.Error("Predict returned no error")
				}
				var buffer bytes.Buffer
				if err := model.ExportONNX(&buffer); err == nil || buffer.Len() != 0 {
					t.Errorf("ExportONNX: err = %v, wrote %d bytes; want an error and nothing written", err, buffer.Len())
				}
			})
		})
	}
	for name, model := range classifiers {
		t.Run(name+" as a classifier", func(t *testing.T) {
			noPanic(t, func() {
				classes := model.Classes()
				if classes == nil || classes.Len() != 0 || classes.Err() == nil {
					t.Errorf("Classes() = %v, want an empty list carrying an error", classes)
				}
				if _, err := model.PredictProba(table); err == nil {
					t.Error("PredictProba returned no error")
				}
			})
		})
	}
	t.Run("LeafValues", func(t *testing.T) {
		noPanic(t, func() {
			if got := treeClassifier.LeafValues(); got != nil {
				t.Errorf("classifier LeafValues() = %v, want nil", got)
			}
			if got := treeRegressor.LeafValues(); got != nil {
				t.Errorf("regressor LeafValues() = %v, want nil", got)
			}
		})
	})
}
