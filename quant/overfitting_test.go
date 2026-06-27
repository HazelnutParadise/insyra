package quant

import (
	"math"
	"testing"

	"github.com/HazelnutParadise/insyra/stats"
)

func TestProbabilisticSharpeRatio(t *testing.T) {
	const tol = 1e-12
	// Normal returns (skew 0, kurt 3), benchmark 0.
	got, err := ProbabilisticSharpeRatio(0.1, 0, 100, 0, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	varTerm := 1 - 0*0.1 + (3-1)/4.0*0.1*0.1
	z := 0.1 * math.Sqrt(99) / math.Sqrt(varTerm)
	if want := stats.NormCDF(z); math.Abs(got-want) > tol {
		t.Errorf("PSR = %v, want %v", got, want)
	}

	// SR̂ == SR* gives exactly 0.5.
	if v, _ := ProbabilisticSharpeRatio(0.2, 0.2, 50, 0, 3); math.Abs(v-0.5) > tol {
		t.Errorf("PSR(SR==SR*) = %v, want 0.5", v)
	}

	if _, err := ProbabilisticSharpeRatio(0.1, 0, 1, 0, 3); err == nil {
		t.Error("expected error for n < 2")
	}
	// Extreme skew/kurtosis can drive the variance term non-positive
	// (1 - 2·1 + (1-1)/4·1² = -1 here); the function must reject it.
	if _, err := ProbabilisticSharpeRatio(1, 0, 100, 2, 1); err == nil {
		t.Error("expected error for non-positive variance term")
	}
}

func TestExpectedMaxSharpe(t *testing.T) {
	const tol = 1e-12

	if v, err := ExpectedMaxSharpe(1, 1); err != nil || v != 0 {
		t.Errorf("ExpectedMaxSharpe(_, 1) = (%v, %v), want (0, nil)", v, err)
	}

	got, err := ExpectedMaxSharpe(1, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	q1, _ := stats.NormPPF(0.5)
	q2, _ := stats.NormPPF(1 - 1/(2*math.E))
	want := (1-eulerMascheroni)*q1 + eulerMascheroni*q2
	if math.Abs(got-want) > tol {
		t.Errorf("ExpectedMaxSharpe(1, 2) = %v, want %v", got, want)
	}

	// SR₀ grows with both the spread of trial Sharpes and the trial count.
	a, _ := ExpectedMaxSharpe(1, 10)
	b, _ := ExpectedMaxSharpe(1, 100)
	if !(b > a) {
		t.Errorf("expected SR0 to grow with nTrials: a=%v b=%v", a, b)
	}

	if _, err := ExpectedMaxSharpe(-1, 5); err == nil {
		t.Error("expected error for negative variance")
	}
}

func TestDeflatedSharpeRatio(t *testing.T) {
	// Strong, isolated result: high observed SR, few low-variance trials →
	// survives deflation (DSR ≈ 1).
	high, err := DeflatedSharpeRatio(2.0, 250, 0, 3, toDL(0.1, 0.2, 0.0, -0.1, 0.15))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if high < 0.99 {
		t.Errorf("strong result DSR = %v, want > 0.99", high)
	}

	// Weak result drowned in many spread-out trials → fails deflation.
	trials := make([]float64, 50)
	for i := range trials {
		trials[i] = 0.6 * float64(i) / 49.0 // 0 .. 0.6
	}
	low, err := DeflatedSharpeRatio(0.3, 120, 0, 3, toDL(trials...))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if low > 0.5 {
		t.Errorf("weak result DSR = %v, want < 0.5", low)
	}

	if _, err := DeflatedSharpeRatio(1, 100, 0, 3, toDL()); err == nil {
		t.Error("expected error for empty trialSharpes")
	}
}

func TestPBONoOverfit(t *testing.T) {
	// Strategy 0 dominates in every period; strategy 1 is flat. The IS
	// winner is always the OOS winner → PBO = 0.
	rows := 8
	perf := make([][]float64, rows)
	for i := range rows {
		s0 := 0.01
		if i%2 == 0 {
			s0 = 0.03
		}
		perf[i] = []float64{s0, 0.005}
	}
	got, err := PBO(toDT(perf), 4)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 0 {
		t.Errorf("PBO(dominant strategy) = %v, want 0", got)
	}
}

func TestPBORange(t *testing.T) {
	// Whatever the data, PBO is a probability in [0, 1].
	perf := [][]float64{
		{0.01, -0.01, 0.02, 0.00},
		{-0.02, 0.03, -0.01, 0.01},
		{0.03, 0.00, 0.01, -0.02},
		{0.00, 0.02, -0.03, 0.03},
		{0.02, -0.01, 0.01, 0.00},
		{-0.01, 0.01, 0.02, -0.01},
		{0.01, 0.00, -0.01, 0.02},
		{0.00, 0.02, 0.01, -0.01},
	}
	got, err := PBO(toDT(perf), 4)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got < 0 || got > 1 {
		t.Errorf("PBO = %v, want in [0, 1]", got)
	}
}

func TestPBOErrors(t *testing.T) {
	good := [][]float64{{1, 2}, {3, 4}, {5, 6}, {7, 8}}
	cases := []struct {
		name    string
		perf    [][]float64
		nSplits int
	}{
		{"empty", nil, 4},
		{"one strategy", [][]float64{{1}, {2}, {3}, {4}}, 2},
		{"odd nSplits", good, 3},
		{"nSplits > rows", good, 8},
	}
	for _, c := range cases {
		if _, err := PBO(toDT(c.perf), c.nSplits); err == nil {
			t.Errorf("%s: expected error", c.name)
		}
	}
}
