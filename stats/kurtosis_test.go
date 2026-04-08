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
		got, err := stats.Kurtosis(data, tt.method)
		if err != nil {
			t.Fatalf("Kurtosis(data, method=%v) error: %v", tt.method, err)
		}
		if !almostEqual(got, tt.expect, tolerance) {
			t.Errorf("Kurtosis(data, method=%v) = %f; want %f", tt.method, got, tt.expect)
		}
	}
}

func TestKurtosis_DefaultMethod(t *testing.T) {
	data := []float64{2.0, 4.0, 7.0, 1.0, 8.0, 3.0, 9.0, 2.0}
	expect := -1.4710743801652892
	got, err := stats.Kurtosis(data)
	if err != nil {
		t.Fatalf("Kurtosis(data) error: %v", err)
	}
	tolerance := 1e-6
	if !almostEqual(got, expect, tolerance) {
		t.Errorf("Kurtosis(data) default = %f; want %f", got, expect)
	}
}

func TestKurtosis_InvalidOrEmpty(t *testing.T) {
	got, err := stats.Kurtosis([]float64{})
	if err == nil {
		t.Errorf("Expected error for empty data")
	}
	if !math.IsNaN(got) {
		t.Errorf("Expected NaN for empty data")
	}

	got, err = stats.Kurtosis([]float64{1.0, 2.0}, stats.KurtosisG2, stats.KurtosisAdjusted)
	if err == nil {
		t.Errorf("Expected error for multiple method args")
	}
	if !math.IsNaN(got) {
		t.Errorf("Expected NaN for multiple method args")
	}
}

func TestKurtosis_MultipleDatasets(t *testing.T) {
	tolerance := 1e-6

	cases := []struct {
		data   []float64
		method stats.KurtosisMethod
		expect float64
	}{
		// Verified via scipy.stats.kurtosis(data, fisher=True)
		{[]float64{1, 2, 3, 4, 5}, stats.KurtosisG2, -1.3},
		{[]float64{10, 12, 23, 23, 16, 23, 21, 16}, stats.KurtosisG2, -1.3984375},
		{[]float64{6, 6, 6, 6, 6}, stats.KurtosisG2, math.NaN()}, // Variance = 0 → should return NaN
		{[]float64{1, 100, 1000, 10000, 100000}, stats.KurtosisG2, 0.2021924443706249},
		{[]float64{2.5, 3.5, 2.8, 3.3, 3.0}, stats.KurtosisG2, -1.292405371414662},
	}

	for i, c := range cases {
		got, err := stats.Kurtosis(c.data, c.method)
		if math.IsNaN(c.expect) {
			if err == nil {
				t.Errorf("Case %d: expected error for NaN case", i+1)
			}
			if !math.IsNaN(got) {
				t.Errorf("Case %d: Expected NaN but got %f", i+1, got)
			}
		} else {
			if err != nil {
				t.Errorf("Case %d: unexpected error: %v", i+1, err)
				continue
			}
			if !almostEqual(got, c.expect, tolerance) {
				t.Errorf("Case %d: Kurtosis(data, method=%v) = %f; want %f", i+1, c.method, got, c.expect)
			}
		}
	}
}
