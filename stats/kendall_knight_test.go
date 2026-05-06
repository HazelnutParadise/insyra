package stats

import (
	"math/rand/v2"
	"testing"
)

// TestKendallKnightMatchesBrute proves the Knight (1966) O(n log n) pair
// counter agrees bit-exactly with the textbook O(n²) brute counter across
// a wide range of inputs: i.i.d. continuous, integer-valued (lots of ties),
// monotone, anti-monotone, and one-axis-constant. Both functions return
// integer-valued counts so equality is float-comparable without tolerance.
func TestKendallKnightMatchesBrute(t *testing.T) {
	type gen func(rng *rand.Rand, n int) (x, y []float64)
	cases := []struct {
		name string
		gen  gen
	}{
		{
			name: "continuous_iid",
			gen: func(rng *rand.Rand, n int) ([]float64, []float64) {
				x := make([]float64, n)
				y := make([]float64, n)
				for i := range n {
					x[i] = rng.NormFloat64()
					y[i] = rng.NormFloat64()
				}
				return x, y
			},
		},
		{
			name: "integer_with_ties",
			gen: func(rng *rand.Rand, n int) ([]float64, []float64) {
				x := make([]float64, n)
				y := make([]float64, n)
				for i := range n {
					x[i] = float64(rng.IntN(8))
					y[i] = float64(rng.IntN(5))
				}
				return x, y
			},
		},
		{
			name: "monotone",
			gen: func(_ *rand.Rand, n int) ([]float64, []float64) {
				x := make([]float64, n)
				y := make([]float64, n)
				for i := range n {
					x[i] = float64(i)
					y[i] = float64(i)
				}
				return x, y
			},
		},
		{
			name: "anti_monotone_with_jitter",
			gen: func(rng *rand.Rand, n int) ([]float64, []float64) {
				x := make([]float64, n)
				y := make([]float64, n)
				for i := range n {
					x[i] = float64(i) + 0.01*rng.NormFloat64()
					y[i] = float64(n-i) + 0.01*rng.NormFloat64()
				}
				return x, y
			},
		},
		{
			name: "y_constant",
			gen: func(rng *rand.Rand, n int) ([]float64, []float64) {
				x := make([]float64, n)
				y := make([]float64, n)
				for i := range n {
					x[i] = rng.NormFloat64()
					y[i] = 3.14
				}
				return x, y
			},
		},
		{
			name: "both_heavy_ties",
			gen: func(rng *rand.Rand, n int) ([]float64, []float64) {
				x := make([]float64, n)
				y := make([]float64, n)
				for i := range n {
					x[i] = float64(rng.IntN(3))
					y[i] = float64(rng.IntN(3))
				}
				return x, y
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			for _, n := range []int{2, 3, 5, 10, 50, 100, 200, 500, 1500} {
				rng := rand.New(rand.NewPCG(uint64(n), 0xCAFE_F00D_BABEface))
				x, y := c.gen(rng, n)
				bC, bD := kendallPairCountBruteSerial(x, y)
				kC, kD := kendallPairCountKnight(x, y)
				if bC != kC || bD != kD {
					t.Fatalf("n=%d: brute (nC=%v, nD=%v) != knight (nC=%v, nD=%v)", n, bC, bD, kC, kD)
				}
				// Also exercise the parallel brute path so we catch any
				// regression in the per-worker accumulator reduction.
				if n >= 192 {
					pC, pD := kendallPairCountBruteParallel(x, y)
					if pC != bC || pD != bD {
						t.Fatalf("n=%d: parallel brute (nC=%v, nD=%v) != serial brute (nC=%v, nD=%v)", n, pC, pD, bC, bD)
					}
				}
			}
		})
	}
}
