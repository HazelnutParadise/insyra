package quant

import (
	"math"
	"testing"
)

// refSharpe is an independent reimplementation used only to validate the
// gonum-backed SharpeRatio (different code path, same definition).
func refSharpe(returns []float64, rf, ppy float64) float64 {
	n := float64(len(returns))
	sum := 0.0
	for _, r := range returns {
		sum += r - rf
	}
	mean := sum / n
	ss := 0.0
	for _, r := range returns {
		d := (r - rf) - mean
		ss += d * d
	}
	sd := math.Sqrt(ss / (n - 1))
	return mean / sd * math.Sqrt(ppy)
}

func TestSharpeRatio(t *testing.T) {
	const tol = 1e-12
	returns := []float64{0.01, -0.02, 0.03, 0.00, 0.02}

	got, err := SharpeRatio(toDL(returns...), 0, 252)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := refSharpe(returns, 0, 252); math.Abs(got-want) > tol {
		t.Errorf("SharpeRatio = %v, want %v", got, want)
	}

	// Annualization scales by √periodsPerYear: ppy=4 is exactly 2× ppy=1.
	s1, _ := SharpeRatio(toDL(returns...), 0, 1)
	s4, _ := SharpeRatio(toDL(returns...), 0, 4)
	if math.Abs(s4-2*s1) > 1e-12 {
		t.Errorf("annualization scaling: s4=%v, 2*s1=%v", s4, 2*s1)
	}

	// Risk-free shift: SharpeRatio(r, rf) equals SharpeRatio(r-rf, 0).
	rf := 0.001
	shifted := make([]float64, len(returns))
	for i, r := range returns {
		shifted[i] = r - rf
	}
	a, _ := SharpeRatio(toDL(returns...), rf, 252)
	b, _ := SharpeRatio(toDL(shifted...), 0, 252)
	if math.Abs(a-b) > 1e-12 {
		t.Errorf("rf shift invariance: %v vs %v", a, b)
	}
}

func TestSharpeRatioErrors(t *testing.T) {
	if _, err := SharpeRatio(toDL(0.01), 0, 252); err == nil {
		t.Error("expected error for <2 returns")
	}
	if _, err := SharpeRatio(toDL(0.01, 0.02), 0, 0); err == nil {
		t.Error("expected error for non-positive periodsPerYear")
	}
	if _, err := SharpeRatio(toDL(0.01, 0.01, 0.01), 0, 252); err == nil {
		t.Error("expected error for zero volatility")
	}
}

func TestMaxDrawdown(t *testing.T) {
	const tol = 1e-12

	// Peak 120 → trough 80 is the worst: (120-80)/120 = 1/3.
	got, err := MaxDrawdown(toDL(100, 120, 90, 110, 80, 130))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := 1.0 / 3.0; math.Abs(got-want) > tol {
		t.Errorf("MaxDrawdown = %v, want %v", got, want)
	}

	// Monotonically increasing curve: no drawdown.
	up, _ := MaxDrawdown(toDL(1, 2, 3, 4))
	if up != 0 {
		t.Errorf("MaxDrawdown(increasing) = %v, want 0", up)
	}

	if _, err := MaxDrawdown(toDL()); err == nil {
		t.Error("expected error for empty equity")
	}
}

func TestAnnualizedReturn(t *testing.T) {
	const tol = 1e-12

	// Doubling over exactly one year (365 days) → 100% annualized.
	got, err := AnnualizedReturn(toDL(100, 150, 200), 365)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if math.Abs(got-1.0) > tol {
		t.Errorf("AnnualizedReturn(2x, 365d) = %v, want 1.0", got)
	}

	// +21% over two years (730 days) → (1.21)^(1/2) - 1 = 10% annualized.
	got2, _ := AnnualizedReturn(toDL(100, 121), 730)
	if math.Abs(got2-0.1) > tol {
		t.Errorf("AnnualizedReturn(1.21x, 730d) = %v, want 0.1", got2)
	}

	for _, tc := range []struct {
		name   string
		equity []float64
		days   int
	}{
		{"too short", []float64{100}, 365},
		{"non-positive days", []float64{100, 200}, 0},
		{"non-positive begin", []float64{0, 200}, 365},
		{"non-positive end", []float64{100, -5}, 365},
	} {
		if _, err := AnnualizedReturn(toDL(tc.equity...), tc.days); err == nil {
			t.Errorf("%s: expected error", tc.name)
		}
	}
}
