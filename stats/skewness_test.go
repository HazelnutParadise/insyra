package stats

import (
	"math"
	"testing"
)

func almostEqual(a, b, tol float64) bool {
	return math.Abs(a-b) < tol
}

func TestSkewness_AllMethods(t *testing.T) {
	data := []float64{2.0, 4.0, 7.0, 1.0, 8.0, 3.0, 9.0, 2.0}
	tolerance := 1e-6

	tests := []struct {
		method SkewnessMethod
		expect float64
	}{
		{SkewnessG1, 0.3798057948124316},
		{SkewnessAdjusted, 0.47370105256649414},
		{SkewnessBiasAdjusted, 0.3108663157467618},
	}

	for _, tt := range tests {
		got := Skewness(data, tt.method)
		if !almostEqual(got, tt.expect, tolerance) {
			t.Errorf("Skewness(data, method=%v) = %f; want %f", tt.method, got, tt.expect)
		}
	}
}

func TestSkewness_DefaultMethod(t *testing.T) {
	data := []float64{2.0, 4.0, 7.0, 1.0, 8.0, 3.0, 9.0, 2.0}
	expect := 0.3798057948124316
	got := Skewness(data)
	tolerance := 1e-6
	if !almostEqual(got, expect, tolerance) {
		t.Errorf("Skewness(data) default = %f; want %f", got, expect)
	}
}

func TestSkewness_InvalidOrEmpty(t *testing.T) {
	empty := []float64{}
	if !math.IsNaN(Skewness(empty)) {
		t.Errorf("Expected NaN for empty slice")
	}

	data := []float64{1.0, 2.0}
	// 多個 method 輸入時應回傳 NaN
	if !math.IsNaN(Skewness(data, SkewnessG1, SkewnessAdjusted)) {
		t.Errorf("Expected NaN for multiple methods input")
	}
}
