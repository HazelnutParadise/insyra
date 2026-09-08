package quant

import (
	"math"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
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

func TestPerformanceRejectsUnreadableInput(t *testing.T) {
	cases := []struct {
		name   string
		values []any
		series string
		row    string
	}{
		{"nil cell", []any{0.01, nil, 0.02}, "returns", "row 2"},
		{"text cell", []any{0.01, "n/a", 0.02}, "returns", "row 2"},
		{"NaN cell", []any{0.01, math.NaN(), 0.02}, "returns", "row 2"},
		{"positive Inf cell", []any{0.01, math.Inf(1), 0.02}, "returns", "row 2"},
		{"negative Inf cell", []any{0.01, math.Inf(-1), 0.02}, "returns", "row 2"},
		{"unreadable last cell", []any{0.01, 0.02, "x"}, "returns", "row 3"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := SharpeRatio(insyra.NewDataList(tc.values...), 0, 252)
			if err == nil {
				t.Fatalf("SharpeRatio returned nil error, got %v", got)
			}
			if !strings.Contains(err.Error(), tc.series) || !strings.Contains(err.Error(), tc.row) {
				t.Errorf("SharpeRatio error = %q, want %s and %s", err, tc.series, tc.row)
			}
			if !math.IsNaN(got) {
				t.Errorf("SharpeRatio = %v on refusal, want NaN", got)
			}
		})
	}

	// The same cells are refused by the equity-curve entry points, which
	// label the series "equity".
	equityCases := []struct {
		name   string
		values []any
		row    string
	}{
		{"text cell", []any{100.0, "n/a", 90.0}, "row 2"},
		{"nil cell", []any{100.0, nil, 90.0}, "row 2"},
		{"NaN cell", []any{100.0, math.NaN(), 90.0}, "row 2"},
		{"Inf cell", []any{100.0, math.Inf(1), 90.0}, "row 2"},
	}
	for _, tc := range equityCases {
		t.Run("equity/"+tc.name, func(t *testing.T) {
			check := func(name string, got float64, err error) {
				t.Helper()
				if err == nil {
					t.Fatalf("%s returned nil error, got %v", name, got)
				}
				if !strings.Contains(err.Error(), "equity") || !strings.Contains(err.Error(), tc.row) {
					t.Errorf("%s error = %q, want equity and %s", name, err, tc.row)
				}
				if !math.IsNaN(got) {
					t.Errorf("%s = %v on refusal, want NaN", name, got)
				}
			}
			dd, ddErr := MaxDrawdown(insyra.NewDataList(tc.values...))
			check("MaxDrawdown", dd, ddErr)
			ar, arErr := AnnualizedReturn(insyra.NewDataList(tc.values...), 30)
			check("AnnualizedReturn", ar, arErr)
		})
	}
}

func TestPerformanceRejectsNilInput(t *testing.T) {
	if got, err := SharpeRatio(nil, 0, 252); err == nil || !strings.Contains(err.Error(), "returns is nil") {
		t.Errorf("SharpeRatio(nil) = %v, %v; want an error naming \"returns is nil\"", got, err)
	}
	if got, err := MaxDrawdown(nil); err == nil || !strings.Contains(err.Error(), "equity is nil") {
		t.Errorf("MaxDrawdown(nil) = %v, %v; want an error naming \"equity is nil\"", got, err)
	}
	if got, err := AnnualizedReturn(nil, 30); err == nil || !strings.Contains(err.Error(), "equity is nil") {
		t.Errorf("AnnualizedReturn(nil) = %v, %v; want an error naming \"equity is nil\"", got, err)
	}
}
