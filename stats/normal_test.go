package stats

import (
	"math"
	"testing"
)

func TestNormCDF(t *testing.T) {
	const tol = 1e-9
	cases := []struct {
		x    float64
		want float64
	}{
		{0, 0.5},
		{1.959963984540054, 0.975},  // classic two-sided 95% critical value
		{-1.959963984540054, 0.025}, // symmetry
		{1, 0.8413447460685429},     // Φ(1)
		{-1, 0.15865525393145707},   // Φ(-1)
		{math.Inf(1), 1},            // +Inf → 1
		{math.Inf(-1), 0},           // -Inf → 0
	}
	for _, c := range cases {
		got := NormCDF(c.x)
		if math.Abs(got-c.want) > tol {
			t.Errorf("NormCDF(%v) = %v, want %v", c.x, got, c.want)
		}
	}

	// Symmetry: Φ(-x) = 1 - Φ(x).
	for _, x := range []float64{0.3, 1.2, 2.5, 4.0} {
		if d := math.Abs(NormCDF(-x) - (1 - NormCDF(x))); d > tol {
			t.Errorf("symmetry broken at x=%v: |Φ(-x) - (1-Φ(x))| = %v", x, d)
		}
	}

	// NaN in → NaN out.
	if !math.IsNaN(NormCDF(math.NaN())) {
		t.Errorf("NormCDF(NaN) = %v, want NaN", NormCDF(math.NaN()))
	}
}

func TestNormPPF(t *testing.T) {
	const tol = 1e-9
	cases := []struct {
		p    float64
		want float64
	}{
		{0.5, 0},
		{0.975, 1.959963984540054},
		{0.025, -1.959963984540054},
		{0.8413447460685429, 1},
	}
	for _, c := range cases {
		got, err := NormPPF(c.p)
		if err != nil {
			t.Errorf("NormPPF(%v) unexpected error: %v", c.p, err)
			continue
		}
		if math.Abs(got-c.want) > tol {
			t.Errorf("NormPPF(%v) = %v, want %v", c.p, got, c.want)
		}
	}

	// Boundaries return the infinite quantiles.
	if got, err := NormPPF(0); err != nil || !math.IsInf(got, -1) {
		t.Errorf("NormPPF(0) = (%v, %v), want (-Inf, nil)", got, err)
	}
	if got, err := NormPPF(1); err != nil || !math.IsInf(got, 1) {
		t.Errorf("NormPPF(1) = (%v, %v), want (+Inf, nil)", got, err)
	}

	// Invalid inputs return an error.
	for _, p := range []float64{-0.01, 1.01, math.NaN(), math.Inf(1)} {
		if _, err := NormPPF(p); err == nil {
			t.Errorf("NormPPF(%v) expected error, got nil", p)
		}
	}
}

func TestNormCDFPPFRoundTrip(t *testing.T) {
	const tol = 1e-9
	for _, x := range []float64{-3, -1.5, -0.2, 0, 0.7, 1.3, 2.8} {
		p := NormCDF(x)
		back, err := NormPPF(p)
		if err != nil {
			t.Fatalf("NormPPF(NormCDF(%v)) error: %v", x, err)
		}
		if math.Abs(back-x) > tol {
			t.Errorf("round trip at x=%v: got %v", x, back)
		}
	}
}
