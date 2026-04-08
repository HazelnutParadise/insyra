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
		got, err := Skewness(data, tt.method)
		if err != nil {
			t.Fatalf("Skewness(data, method=%v) error: %v", tt.method, err)
		}
		if !almostEqual(got, tt.expect, tolerance) {
			t.Errorf("Skewness(data, method=%v) = %f; want %f", tt.method, got, tt.expect)
		}
	}
}

func TestSkewness_DefaultMethod(t *testing.T) {
	data := []float64{2.0, 4.0, 7.0, 1.0, 8.0, 3.0, 9.0, 2.0}
	expect := 0.3798057948124316
	got, err := Skewness(data)
	if err != nil {
		t.Fatalf("Skewness(data) error: %v", err)
	}
	tolerance := 1e-6
	if !almostEqual(got, expect, tolerance) {
		t.Errorf("Skewness(data) default = %f; want %f", got, expect)
	}
}

func TestSkewness_InvalidOrEmpty(t *testing.T) {
	empty := []float64{}
	got, err := Skewness(empty)
	if err == nil {
		t.Errorf("Expected error for empty slice")
	}
	if !math.IsNaN(got) {
		t.Errorf("Expected NaN for empty slice")
	}

	data := []float64{1.0, 2.0}
	got, err = Skewness(data, SkewnessG1, SkewnessAdjusted)
	if err == nil {
		t.Errorf("Expected error for multiple methods input")
	}
	if !math.IsNaN(got) {
		t.Errorf("Expected NaN for multiple methods input")
	}
}
