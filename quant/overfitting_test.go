package quant

import (
	"math"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
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

func TestDeflatedSharpeRatioRejectsUnreadableInput(t *testing.T) {
	cases := []struct {
		name   string
		values []any
		row    string
	}{
		{"NaN trial", []any{0.5, math.NaN(), 0.7}, "row 2"},
		{"Inf trial", []any{0.5, math.Inf(1), 0.7}, "row 2"},
		{"nil trial", []any{0.5, nil, 0.7}, "row 2"},
		{"text trial", []any{0.5, 0.6, "n/a"}, "row 3"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := DeflatedSharpeRatio(1.0, 100, 0, 3, insyra.NewDataList(tc.values...))
			if err == nil {
				t.Fatalf("DeflatedSharpeRatio returned nil error, got %v", got)
			}
			if !strings.Contains(err.Error(), "trialSharpes") || !strings.Contains(err.Error(), tc.row) {
				t.Errorf("error = %q, want trialSharpes and %s", err, tc.row)
			}
			if !math.IsNaN(got) {
				t.Errorf("DeflatedSharpeRatio = %v on refusal, want NaN", got)
			}
		})
	}

	if got, err := DeflatedSharpeRatio(1.0, 100, 0, 3, nil); err == nil || !strings.Contains(err.Error(), "trialSharpes is nil") {
		t.Errorf("DeflatedSharpeRatio(nil) = %v, %v; want an error naming \"trialSharpes is nil\"", got, err)
	}
}

func TestPBORejectsUnreadableInput(t *testing.T) {
	// Column 1 (zero-based, the second column) holds a string in row 3
	// (one-based). The error must name both.
	colA := insyra.NewDataList(0.01, 0.02, 0.03, 0.04)
	colB := insyra.NewDataList(0.02, 0.01, "x", 0.03)
	got, err := PBO(insyra.NewDataTable(colA, colB), 2)
	if err == nil {
		t.Fatalf("PBO returned nil error, got %v", got)
	}
	if !strings.Contains(err.Error(), "column 1") || !strings.Contains(err.Error(), "row 3") {
		t.Errorf("PBO error = %q, want column 1 and row 3", err)
	}
	if !math.IsNaN(got) {
		t.Errorf("PBO = %v on refusal, want NaN", got)
	}

	// A non-finite cell in the first column is refused the same way.
	nanCol := insyra.NewDataList(0.01, math.NaN(), 0.03, 0.04)
	if _, err := PBO(insyra.NewDataTable(nanCol, colA), 2); err == nil ||
		!strings.Contains(err.Error(), "column 0") || !strings.Contains(err.Error(), "row 2") {
		t.Errorf("PBO error = %v, want column 0 and row 2", err)
	}
}
