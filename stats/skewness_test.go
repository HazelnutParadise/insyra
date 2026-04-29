package stats_test

import (
	"math"
	"testing"

	"github.com/HazelnutParadise/insyra/stats"
)

const tolSkew = 1e-12

func TestSkewness_R(t *testing.T) {
	cases := []struct {
		name   string
		data   []float64
		prefix string
		// skipAdjusted is set when n < 3 (Adjusted is undefined)
		skipAdjusted bool
	}{
		{"basic", []float64{2, 4, 7, 1, 8, 3, 9, 2}, "sk_basic", false},
		{"minN_n3", []float64{10, 12, 8}, "sk_minN", false},
		{"n4_with_outlier", []float64{1, 2, 3, 100}, "sk_n4", false},
		{"symmetric_n9", []float64{1, 2, 3, 4, 5, 6, 7, 8, 9}, "sk_symmetric", false},
		{"negative_skew", []float64{-100, 1, 2, 3, 4, 5}, "sk_negSkew", false},
		{"largeN_n100", func() []float64 {
			out := make([]float64, 100)
			for i := range out {
				out[i] = float64(i + 1)
			}
			return out
		}(), "sk_largeN", false},
		{"huge_magnitude", []float64{1e6, 2e6, 1.5e6, 3e6, 1.2e6}, "sk_huge", false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			gotG1, err := stats.Skewness(c.data, stats.SkewnessG1)
			if err != nil {
				t.Fatalf("G1 error: %v", err)
			}
			expG1 := momentRef.get(t, c.prefix+".g1")
			if !mClose(gotG1, expG1, tolSkew) {
				t.Errorf("G1: got %.17g, want %.17g (Δ=%g)",
					gotG1, expG1, math.Abs(gotG1-expG1))
			}

			if !c.skipAdjusted {
				gotAdj, err := stats.Skewness(c.data, stats.SkewnessAdjusted)
				if err != nil {
					t.Fatalf("Adjusted error: %v", err)
				}
				expAdj := momentRef.get(t, c.prefix+".adj")
				if !mClose(gotAdj, expAdj, tolSkew) {
					t.Errorf("Adjusted: got %.17g, want %.17g", gotAdj, expAdj)
				}
			}

			gotBA, err := stats.Skewness(c.data, stats.SkewnessBiasAdjusted)
			if err != nil {
				t.Fatalf("BiasAdjusted error: %v", err)
			}
			expBA := momentRef.get(t, c.prefix+".biasadj")
			if !mClose(gotBA, expBA, tolSkew) {
				t.Errorf("BiasAdjusted: got %.17g, want %.17g", gotBA, expBA)
			}
		})
	}
}

func TestSkewness_DefaultMethod(t *testing.T) {
	// Default (no method specified) must equal SkewnessG1.
	data := []float64{2, 4, 7, 1, 8, 3, 9, 2}
	def, err := stats.Skewness(data)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	g1, _ := stats.Skewness(data, stats.SkewnessG1)
	if def != g1 {
		t.Errorf("default %.17g != G1 %.17g", def, g1)
	}
}

func TestSkewness_Errors(t *testing.T) {
	if _, err := stats.Skewness([]float64{}); err == nil {
		t.Error("expected error for empty data")
	}
	if got, _ := stats.Skewness([]float64{}); !math.IsNaN(got) {
		t.Errorf("expected NaN for empty data, got %v", got)
	}
	// Adjusted requires n >= 3
	if _, err := stats.Skewness([]float64{1, 2}, stats.SkewnessAdjusted); err == nil {
		t.Error("expected error for SkewnessAdjusted with n<3")
	}
	// Multiple methods: error
	if _, err := stats.Skewness([]float64{1, 2, 3}, stats.SkewnessG1, stats.SkewnessAdjusted); err == nil {
		t.Error("expected error for multiple methods")
	}
	// Zero variance: error
	if _, err := stats.Skewness([]float64{5, 5, 5, 5}); err == nil {
		t.Error("expected error for zero-variance data")
	}
}
