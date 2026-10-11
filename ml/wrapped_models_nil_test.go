package ml_test

import (
	"bytes"
	"testing"

	"github.com/HazelnutParadise/insyra/ml"
)

func TestWrappedModelsAreSafeWhenNil(t *testing.T) {
	table := testFeatures().table
	models := []struct {
		name  string
		model ml.Model
	}{
		{"LinearModel", (*ml.LinearModel)(nil)},
		{"PolynomialModel", (*ml.PolynomialModel)(nil)},
		{"WeightedLinearModel", (*ml.WeightedLinearModel)(nil)},
		{"RidgeModel", (*ml.RidgeModel)(nil)},
		{"LassoModel", (*ml.LassoModel)(nil)},
		{"ExponentialModel", (*ml.ExponentialModel)(nil)},
		{"LogarithmicModel", (*ml.LogarithmicModel)(nil)},
		{"LogisticModel", (*ml.LogisticModel)(nil)},
		{"PoissonModel", (*ml.PoissonModel)(nil)},
		{"GLMModel", (*ml.GLMModel)(nil)},
		{"KMeansModel", (*ml.KMeansModel)(nil)},
		{"KNNClassifier", (*ml.KNNClassifier)(nil)},
		{"KNNRegressor", (*ml.KNNRegressor)(nil)},
	}
	for _, tc := range models {
		t.Run(tc.name, func(t *testing.T) {
			// Features stays a value method promoted from modelBase on this
			// line; see TestTreeModelsAreSafeWhenNil.
			if _, err := tc.model.Predict(table); err == nil {
				t.Error("Predict returned no error")
			}
		})
	}

	exporters := []struct {
		name  string
		model ml.Exporter
	}{
		{"LinearModel", (*ml.LinearModel)(nil)},
		{"WeightedLinearModel", (*ml.WeightedLinearModel)(nil)},
		{"RidgeModel", (*ml.RidgeModel)(nil)},
		{"LassoModel", (*ml.LassoModel)(nil)},
		{"LogisticModel", (*ml.LogisticModel)(nil)},
	}
	for _, tc := range exporters {
		t.Run(tc.name+" ExportONNX", func(t *testing.T) {
			var buffer bytes.Buffer
			if err := tc.model.ExportONNX(&buffer); err == nil || buffer.Len() != 0 {
				t.Errorf("ExportONNX: err = %v, wrote %d bytes; want an error and no output", err, buffer.Len())
			}
		})
	}

	var pca *ml.PCATransformer
	if got := pca.Features(); got != nil {
		t.Errorf("PCATransformer.Features() = %v, want nil", got)
	}
	if _, err := pca.Transform(table); err == nil {
		t.Error("PCATransformer.Transform returned no error")
	}
}
