package stats_test

import (
	"math"
	"testing"

	"github.com/HazelnutParadise/insyra/stats"
	"github.com/HazelnutParadise/insyra/stats/internal/fa"
	"gonum.org/v1/gonum/mat"
)

// A factor solution is defined up to the order and sign of its factors. The
// parity comparison used to compare rotation matrices element by element, so
// a column psych had not sign-standardised counted as a difference of 2.0 —
// 1,311 of the 5,334 failing leaves on 2026-09-12.
func TestFactorAlignmentIgnoresOrderAndSign(t *testing.T) {
	L := [][]float64{{0.9, 0.1, 0.0}, {0.8, 0.2, 0.1}, {0.1, 0.9, 0.0}, {0.0, 0.85, 0.2}, {0.1, 0.0, 0.9}, {0.2, 0.1, 0.8}}
	// R's frame: columns 2, 0, 1 of ours, the middle one flipped.
	want := make([][]float64, len(L))
	for i := range L {
		want[i] = []float64{L[i][2], -L[i][0], L[i][1]}
	}
	al, ok := alignFactors(L, want)
	if !ok {
		t.Fatal("alignFactors refused matching shapes")
	}
	if d := maxAbsGrid(al.columns(L), want); d != 0 {
		t.Errorf("aligned loadings differ from R's by %.3e, want 0", d)
	}
	if al.identity() {
		t.Error("a permuted, flipped frame reported as the identity")
	}

	// Phi transforms on both axes: D·P'·Φ·P·D keeps it symmetric with a
	// unit diagonal and moves the correlation to the permuted pair.
	phi := [][]float64{{1, 0.3, 0.1}, {0.3, 1, 0.2}, {0.1, 0.2, 1}}
	got := al.both(phi)
	for i := range got {
		if got[i][i] != 1 {
			t.Errorf("phi[%d,%d] = %v after alignment, want 1", i, i, got[i][i])
		}
		for j := range got[i] {
			if got[i][j] != got[j][i] {
				t.Errorf("aligned phi is not symmetric at [%d,%d]", i, j)
			}
		}
	}
	// ours (2,0) = 0.1 → R's (0,1), with column 1 flipped: −0.1
	if got[0][1] != -0.1 {
		t.Errorf("aligned phi[0,1] = %v, want -0.1", got[0][1])
	}

	if v := al.vector([]float64{10, 20, 30}); v[0] != 30 || v[1] != 10 || v[2] != 20 {
		t.Errorf("aligned vector = %v, want [30 10 20]", v)
	}

	if _, ok := alignFactors(L, L[:3]); ok {
		t.Error("alignFactors accepted mismatched shapes")
	}
}

// Two solutions of a multimodal rotation are compared by the criterion the
// rotation minimises, and the lower value is the better answer whichever
// minimum R's unseeded starts happened to reach.
func TestCriterionOrdersSolutions(t *testing.T) {
	simple := mat.NewDense(4, 2, []float64{0.9, 0, 0.9, 0, 0, 0.9, 0, 0.9})
	mixed := mat.NewDense(4, 2, []float64{0.7, 0.5, 0.7, 0.5, 0.5, 0.7, 0.5, 0.7})
	for _, method := range []string{"varimax", "quartimax", "geominT", "geominQ", "bentlerT", "bentlerQ", "quartimin", "oblimin", "simplimax"} {
		fs, err := fa.Criterion(method, simple, 0, 0.01)
		if err != nil {
			t.Fatalf("%s: %v", method, err)
		}
		fm, err := fa.Criterion(method, mixed, 0, 0.01)
		if err != nil {
			t.Fatalf("%s: %v", method, err)
		}
		if !(fs < fm) {
			t.Errorf("%s: criterion of the simple structure (%.6g) is not below the mixed one (%.6g)", method, fs, fm)
		}
	}

	// oblimin at gamma 0 is quartimin.
	fo, _ := fa.Criterion("oblimin", mixed, 0, 0)
	fq, _ := fa.Criterion("quartimin", mixed, 0, 0)
	if math.Abs(fo-fq) > 1e-15 {
		t.Errorf("oblimin(gamma 0) = %.12g, quartimin = %.12g", fo, fq)
	}

	if _, err := fa.Criterion("promax", mixed, 0, 0); err == nil {
		t.Error("promax has no criterion, but Criterion returned one")
	}
	if !gpaCriterionRotations[stats.FactorRotationSimplimax] || gpaCriterionRotations[stats.FactorRotationPromax] {
		t.Error("gpaCriterionRotations does not match what Criterion accepts")
	}
}

// The cache key binds a baseline to the toolchain that produced it, so a
// psych upgrade regenerates the baselines instead of answering from the
// previous version's numbers.
func TestBaselineCacheKeyBindsToolchain(t *testing.T) {
	a := baselineCacheKey([]byte("script"), "Rscript", "R 4.5.1|psych 2.5.3|GPArotation 2025.3-1", "factor_analysis", []byte(`{"x":1}`))
	b := baselineCacheKey([]byte("script"), "Rscript", "R 4.5.1|psych 2.6.5|GPArotation 2026.8-2", "factor_analysis", []byte(`{"x":1}`))
	if a == b {
		t.Error("two toolchains produced the same cache key")
	}
	if c := baselineCacheKey([]byte("script"), "Rscript", "R 4.5.1|psych 2.6.5|GPArotation 2026.8-2", "factor_analysis", []byte(`{"x":1}`)); c != b {
		t.Error("the same inputs produced different keys")
	}
}
