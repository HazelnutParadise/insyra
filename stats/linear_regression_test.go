package stats

import (
	"math"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

func TestLinearRegression(t *testing.T) {
	dlX := insyra.NewDataList(1, 4, 9, 110, 8)
	dlY := insyra.NewDataList(3, 6, 8, 6, 2)
	result := LinearRegression(dlX, dlY)
	const tolerance = 1e-9 // 誤差容許範圍

	var expectedValues = LinearRegressionResult{
		Slope:            0.013102128241352597,
		Intercept:        4.654103814428291,
		RSquared:         0.06278103115648115,
		PValue:           0.6843455560821862,
		StandardError:    0.029227221522309617,
		TValue:           0.4482851108974395,
		AdjustedRSquared: -0.2496252917913584,
		Residuals:        []float64{-1.6672059426696437, 1.2934876726062985, 3.2279770313995355, -0.0953379209770766, -2.758920840359112},
	}

	if result == nil {
		t.Error("LinearRegression returned nil")
	} else if !compareFloat64(result.Slope, expectedValues.Slope, tolerance) {
		t.Errorf("Slope: expected %.4f, got %.4f", expectedValues.Slope, result.Slope)
	} else if !compareFloat64(result.Intercept, expectedValues.Intercept, tolerance) {
		t.Errorf("Intercept: expected %.4f, got %.4f", expectedValues.Intercept, result.Intercept)
	} else if !compareFloat64(result.RSquared, expectedValues.RSquared, tolerance) {
		t.Errorf("RSquared: expected %.4f, got %.4f", expectedValues.RSquared, result.RSquared)
	} else if !compareFloat64(result.PValue, expectedValues.PValue, tolerance) {
		t.Errorf("PValue: expected %.4f, got %.4f", expectedValues.PValue, result.PValue)
	} else if !compareFloat64(result.StandardError, expectedValues.StandardError, tolerance) {
		t.Errorf("StandardError: expected %.4f, got %.4f", expectedValues.StandardError, result.StandardError)
	} else if !compareFloat64(result.TValue, expectedValues.TValue, tolerance) {
		t.Errorf("TValue: expected %.4f, got %.4f", expectedValues.TValue, result.TValue)
	} else if !compareFloat64(result.AdjustedRSquared, expectedValues.AdjustedRSquared, tolerance) {
		t.Errorf("AdjustedRSquared: expected %.4f, got %.4f", expectedValues.AdjustedRSquared, result.AdjustedRSquared)
	} else if !compareFloat64Slice(result.Residuals, expectedValues.Residuals, tolerance) {
		t.Errorf("Residuals: expected %v, got %v", expectedValues.Residuals, result.Residuals)
	}

}

func compareFloat64(a, b, tolerance float64) bool {
	return math.Abs(a-b) <= tolerance
}

func compareFloat64Slice(a, b []float64, tolerance float64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !compareFloat64(a[i], b[i], tolerance) {
			return false
		}
	}
	return true
}
