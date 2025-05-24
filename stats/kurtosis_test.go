package stats_test

import (
	"math"
	"testing"

	"github.com/HazelnutParadise/insyra/stats"
)

func TestKurtosis_AllMethods(t *testing.T) {
	data := []float64{2.0, 4.0, 7.0, 1.0, 8.0, 3.0, 9.0, 2.0}
	tolerance := 1e-6

	tests := []struct {
		method stats.KurtosisMethod
		expect float64
	}{
		{stats.KurtosisG2, -1.4710743801652892},
		{stats.KurtosisAdjusted, -1.6892561983471073},
		{stats.KurtosisBiasAdjusted, -1.8294163223140496},
	}

	for _, tt := range tests {
		got := stats.Kurtosis(data, tt.method)
		if !almostEqual(got, tt.expect, tolerance) {
			t.Errorf("Kurtosis(data, method=%v) = %f; want %f", tt.method, got, tt.expect)
		}
	}
}

func TestKurtosis_DefaultMethod(t *testing.T) {
	data := []float64{2.0, 4.0, 7.0, 1.0, 8.0, 3.0, 9.0, 2.0}
	expect := -1.4710743801652892
	got := stats.Kurtosis(data)
	tolerance := 1e-6
	if !almostEqual(got, expect, tolerance) {
		t.Errorf("Kurtosis(data) default = %f; want %f", got, expect)
	}
}

func TestKurtosis_InvalidOrEmpty(t *testing.T) {
	if !math.IsNaN(stats.Kurtosis([]float64{})) {
		t.Errorf("Expected NaN for empty data")
	}

	if !math.IsNaN(stats.Kurtosis([]float64{1.0, 2.0}, stats.KurtosisG2, stats.KurtosisAdjusted)) {
		t.Errorf("Expected NaN for multiple method args")
	}
}
