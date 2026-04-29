// fa/lbfgsb_blas_parity_test.go
//
// Compares our hand-rolled L-BFGS-B BLAS-1 against gonum's blas64. Both
// reductions are sequential left-folds on unit-stride inputs, so they
// should match bit-for-bit when gonum is built with `-tags noasm`. With
// gonum's default amd64 SIMD path, ddot accumulates with parallel lanes
// and diverges by a few ULP per ~1e3 elements — this is logged as a
// reference, not asserted.
package fa

import (
	"math"
	"math/rand"
	"testing"

	"gonum.org/v1/gonum/blas/blas64"
)

func TestDdotMatchesGonum(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	maxDelta := 0.0
	for _, n := range []int{1, 4, 5, 8, 17, 100, 1000} {
		x := make([]float64, n)
		y := make([]float64, n)
		for i := 0; i < n; i++ {
			x[i] = rng.NormFloat64()
			y[i] = rng.NormFloat64()
		}
		ours := ddot(n, x, 1, y, 1)
		theirs := blas64.Implementation().Ddot(n, x, 1, y, 1)
		if d := math.Abs(ours - theirs); d > maxDelta {
			maxDelta = d
		}
		if math.Float64bits(ours) != math.Float64bits(theirs) {
			t.Logf("n=%d: ours=%.20g gonum=%.20g (delta=%g) — expected on amd64 SIMD path",
				n, ours, theirs, ours-theirs)
		}
	}
	// Catastrophic divergence beyond a few hundred ULP would indicate a real bug.
	if maxDelta > 1e-12 {
		t.Errorf("ddot diverges from gonum Ddot by %g, beyond plausible SIMD-vs-scalar drift", maxDelta)
	}
}

func TestDaxpyMatchesGonum(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	for _, n := range []int{1, 3, 4, 5, 17, 100, 1000} {
		x := make([]float64, n)
		yOur := make([]float64, n)
		yGo := make([]float64, n)
		for i := 0; i < n; i++ {
			x[i] = rng.NormFloat64()
			yOur[i] = rng.NormFloat64()
			yGo[i] = yOur[i]
		}
		alpha := rng.NormFloat64()
		daxpy(n, alpha, x, 1, yOur, 1)
		blas64.Implementation().Daxpy(n, alpha, x, 1, yGo, 1)
		for i := 0; i < n; i++ {
			if math.Float64bits(yOur[i]) != math.Float64bits(yGo[i]) {
				t.Errorf("n=%d i=%d: our=%.20g gonum=%.20g (delta=%g)",
					n, i, yOur[i], yGo[i], yOur[i]-yGo[i])
				break
			}
		}
	}
}

func TestDscalMatchesGonum(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	for _, n := range []int{1, 4, 5, 8, 17, 100} {
		xOur := make([]float64, n)
		xGo := make([]float64, n)
		for i := 0; i < n; i++ {
			v := rng.NormFloat64()
			xOur[i] = v
			xGo[i] = v
		}
		alpha := rng.NormFloat64()
		dscal(n, alpha, xOur, 1)
		blas64.Implementation().Dscal(n, alpha, xGo, 1)
		for i := 0; i < n; i++ {
			if math.Float64bits(xOur[i]) != math.Float64bits(xGo[i]) {
				t.Errorf("n=%d i=%d: our=%.20g gonum=%.20g", n, i, xOur[i], xGo[i])
				break
			}
		}
	}
}

func TestDnrm2MatchesGonum(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	for _, n := range []int{1, 4, 5, 8, 17, 100, 1000} {
		x := make([]float64, n)
		for i := 0; i < n; i++ {
			x[i] = rng.NormFloat64()
		}
		ours := dnrm2(n, x, 1)
		theirs := blas64.Implementation().Dnrm2(n, x, 1)
		// Refblas dnrm2 (scale-then-square) and gonum Dnrm2 (Hammarling)
		// are different algorithms — compare with relative tolerance, not bits.
		if math.Abs(ours-theirs) > 1e-13*math.Max(math.Abs(ours), 1) {
			t.Errorf("n=%d: our dnrm2=%.20g gonum Dnrm2=%.20g (rel delta=%g)",
				n, ours, theirs, (ours-theirs)/theirs)
		}
	}
}
