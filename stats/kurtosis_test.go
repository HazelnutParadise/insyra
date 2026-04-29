package stats_test

import (
	"math"
	"testing"

	"github.com/HazelnutParadise/insyra/stats"
)

const tolKurt = 1e-12

func TestKurtosis_R(t *testing.T) {
	cases := []struct {
		name   string
		data   []float64
		prefix string
		// skipAdjusted is set when n < 4 (Adjusted is undefined)
		skipAdjusted bool
	}{
		{"basic", []float64{2, 4, 7, 1, 8, 3, 9, 2}, "ku_basic", false},
		{"n5_uniform", []float64{1, 2, 3, 4, 5}, "ku_n5", false},
		{"n8_bimodal", []float64{10, 12, 23, 23, 16, 23, 21, 16}, "ku_n8", false},
		{"extreme_growth", []float64{1, 100, 1000, 10000, 100000}, "ku_extreme", false},
		{"tight_n5", []float64{2.5, 3.5, 2.8, 3.3, 3.0}, "ku_tight", false},
		{"largeN_n100_bimodal", func() []float64 {
			out := make([]float64, 100)
			for i := range 90 {
				out[i] = 0
			}
			for i := 90; i < 100; i++ {
				out[i] = 10
			}
			return out
		}(), "ku_largeN", false},
		{"n4_minimum_for_adjusted", []float64{1, 2, 3, 4}, "ku_n4", false},
		{"huge_magnitude", []float64{1e6, 1.001e6, 0.999e6, 1.0005e6, 1.0015e6}, "ku_huge", false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			gotG2, err := stats.Kurtosis(c.data, stats.KurtosisG2)
			if err != nil {
				t.Fatalf("G2 error: %v", err)
			}
			expG2 := momentRef.get(t, c.prefix+".g2")
			if !mClose(gotG2, expG2, tolKurt) {
				t.Errorf("G2: got %.17g, want %.17g (Δ=%g)",
					gotG2, expG2, math.Abs(gotG2-expG2))
			}

			if !c.skipAdjusted {
				gotAdj, err := stats.Kurtosis(c.data, stats.KurtosisAdjusted)
				if err != nil {
					t.Fatalf("Adjusted error: %v", err)
				}
				expAdj := momentRef.get(t, c.prefix+".adj")
				if !mClose(gotAdj, expAdj, tolKurt) {
					t.Errorf("Adjusted: got %.17g, want %.17g", gotAdj, expAdj)
				}
			}

			gotBA, err := stats.Kurtosis(c.data, stats.KurtosisBiasAdjusted)
			if err != nil {
				t.Fatalf("BiasAdjusted error: %v", err)
			}
			expBA := momentRef.get(t, c.prefix+".biasadj")
			if !mClose(gotBA, expBA, tolKurt) {
				t.Errorf("BiasAdjusted: got %.17g, want %.17g", gotBA, expBA)
			}
		})
	}
}

func TestKurtosis_DefaultMethod(t *testing.T) {
	data := []float64{2, 4, 7, 1, 8, 3, 9, 2}
	def, err := stats.Kurtosis(data)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	g2, _ := stats.Kurtosis(data, stats.KurtosisG2)
	if def != g2 {
		t.Errorf("default %.17g != G2 %.17g", def, g2)
	}
}

func TestKurtosis_Errors(t *testing.T) {
	if _, err := stats.Kurtosis([]float64{}); err == nil {
		t.Error("expected error for empty data")
	}
	if got, _ := stats.Kurtosis([]float64{}); !math.IsNaN(got) {
		t.Errorf("expected NaN for empty data, got %v", got)
	}
	// Adjusted requires n >= 4
	if _, err := stats.Kurtosis([]float64{1, 2, 3}, stats.KurtosisAdjusted); err == nil {
		t.Error("expected error for KurtosisAdjusted with n<4")
	}
	// Multiple methods
	if _, err := stats.Kurtosis([]float64{1, 2, 3}, stats.KurtosisG2, stats.KurtosisAdjusted); err == nil {
		t.Error("expected error for multiple methods")
	}
	// Zero variance
	if _, err := stats.Kurtosis([]float64{6, 6, 6, 6, 6}, stats.KurtosisG2); err == nil {
		t.Error("expected error for zero-variance data")
	}
}
