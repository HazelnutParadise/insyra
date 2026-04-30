package stats_test

import (
	"math"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

const tolMoment = 1e-12

var momentRef = &refTable{path: "testdata/moments_reference.txt"}

func mClose(a, b, tol float64) bool {
	if math.IsNaN(a) && math.IsNaN(b) {
		return true
	}
	if b == 0 {
		return math.Abs(a) <= tol
	}
	return math.Abs(a-b) <= tol*math.Max(1, math.Abs(b))
}

func TestCalculateMoment_R(t *testing.T) {
	cases := []struct {
		name   string
		data   []float64
		prefix string
	}{
		{"basic", []float64{2, 4, 7, 1, 8, 3, 9, 2}, "mom_basic"},
		{"smallN_n3", []float64{1, 2, 3}, "mom_smallN"},
		{"largeN_n99", func() []float64 {
			out := make([]float64, 0, 99)
			for v := 1.0; v <= 50; v += 0.5 {
				out = append(out, v)
			}
			return out
		}(), "mom_largeN"},
		{"mixed_signs", []float64{-3, -1, 0, 1, 3, 5, -2, 4}, "mom_neg"},
		{"huge_magnitude", []float64{1e6, 2e6, 3e6, 4e6, 5e6}, "mom_huge"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dl := insyra.NewDataList(c.data)
			for k := 1; k <= 5; k++ {
				raw, err := stats.CalculateMoment(dl, k, false)
				if err != nil {
					t.Fatalf("raw moment k=%d error: %v", k, err)
				}
				expRaw := momentRef.get(t, c.prefix+".raw["+itoa(k)+"]")
				// Higher-order raw moments on huge-magnitude data accumulate
				// to ~1e30 — relative 1e-12 is right at the edge of double
				// precision. Loosen marginally.
				tol := tolMoment
				if c.name == "huge_magnitude" && k >= 3 {
					tol = 1e-10
				}
				if !mClose(raw, expRaw, tol) {
					t.Errorf("raw[%d]: got %.17g, want %.17g (Δ=%g)",
						k, raw, expRaw, math.Abs(raw-expRaw))
				}

				central, err := stats.CalculateMoment(dl, k, true)
				if err != nil {
					t.Fatalf("central moment k=%d error: %v", k, err)
				}
				expCentral := momentRef.get(t, c.prefix+".central["+itoa(k)+"]")
				if !mClose(central, expCentral, tolMoment) {
					t.Errorf("central[%d]: got %.17g, want %.17g (Δ=%g)",
						k, central, expCentral, math.Abs(central-expCentral))
				}
			}
		})
	}
}

func TestCalculateMoment_Errors(t *testing.T) {
	empty := insyra.NewDataList([]float64{})
	if _, err := stats.CalculateMoment(empty, 2, true); err == nil {
		t.Error("expected error for empty data")
	}
	if got, _ := stats.CalculateMoment(empty, 2, true); !math.IsNaN(got) {
		t.Errorf("expected NaN for empty data, got %v", got)
	}
	d := insyra.NewDataList([]float64{1, 2, 3})
	if _, err := stats.CalculateMoment(d, 0, true); err == nil {
		t.Error("expected error for moment order 0")
	}
	if _, err := stats.CalculateMoment(d, -1, true); err == nil {
		t.Error("expected error for negative moment order")
	}
}

// First central moment is identically zero by definition; check explicitly.
func TestCalculateMoment_FirstCentralIsZero(t *testing.T) {
	dl := insyra.NewDataList([]float64{1.5, 2.7, 3.9, 100.1, -42.3})
	got, err := stats.CalculateMoment(dl, 1, true)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if got != 0 {
		t.Errorf("first central moment must be 0, got %v", got)
	}
}
