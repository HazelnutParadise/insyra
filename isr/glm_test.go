package isr

import (
	"math"
	"testing"
)

func TestISRLogisticRegression(t *testing.T) {
	y := DL.From(0, 0, 1, 0, 1, 1, 0, 1, 1, 0)
	x := DL.From(-2.1, -1.5, -0.4, 0.1, 0.6, 1.2, 1.7, 2.1, 2.8, 3.3)
	got, err := LogisticRegressionWithOptions(LogisticRegressionOptions{MaxIter: 100, Tolerance: 1e-10}, y, x)
	if err != nil {
		t.Fatalf("LogisticRegressionWithOptions error: %v", err)
	}
	if len(got.Coefficients) != 2 {
		t.Fatalf("expected 2 coefficients, got %d", len(got.Coefficients))
	}
}

func TestISRPoissonRegression(t *testing.T) {
	y := DL.From(1, 2, 3, 4, 6, 8)
	x := DL.From(0.1, 0.4, 0.8, 1.2, 1.7, 2.0)
	got, err := PoissonRegressionWithOptions(PoissonRegressionOptions{MaxIter: 100, Tolerance: 1e-10}, y, x)
	if err != nil {
		t.Fatalf("PoissonRegressionWithOptions error: %v", err)
	}
	if len(got.IncidenceRateRatios) != 2 {
		t.Fatalf("expected 2 IRRs, got %d", len(got.IncidenceRateRatios))
	}
}

func TestISRGLM(t *testing.T) {
	y := DL.From(1.2, 2.1, 2.9, 4.2, 5.1, 5.8)
	x := DL.From(0.2, 0.7, 1.1, 1.8, 2.2, 2.9)
	got, err := GLM(GLMOptions{Family: Gaussian, Link: Identity, MaxIter: 100, Tolerance: 1e-10}, y, x)
	if err != nil {
		t.Fatalf("GLM error: %v", err)
	}
	pred, err := got.Predict(PredictResponse, DL.From(1.5, 2.5))
	if err != nil {
		t.Fatalf("Predict error: %v", err)
	}
	vals := pred.ToF64Slice()
	if len(vals) != 2 || math.IsNaN(vals[0]) || math.IsNaN(vals[1]) {
		t.Fatalf("unexpected predictions: %v", vals)
	}
}
