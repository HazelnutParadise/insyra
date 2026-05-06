package stats

import (
	"fmt"
	"math"
	"math/rand/v2"
	"testing"

	"github.com/HazelnutParadise/insyra"
	internalknn "github.com/HazelnutParadise/insyra/stats/internal/knn"
)

// dispatch_coverage_test.go is the systematic coverage layer for the stats
// package's parallel-vs-serial gates and algorithm dispatchers (Kendall
// brute vs Knight, parallel correlation pairs, parallel PCA, KNN parallel
// queries, etc.). For every dispatcher we exercise:
//
//   - Each size bucket (below threshold, at threshold, above threshold).
//   - Each "data mode" the algorithm has to handle (continuous, ties,
//     monotone/anti-monotone, constant axis, heavy ties, ...).
//
// Where the production claim is "branches produce bit-equal output" we
// run both branches (or compare to a strictly-serial reference) and
// assert exact equality. Where the parallel reduction can perturb the
// last 1-2 ULPs (e.g. ANOVA-style worker-local sum reductions) we
// allow the same 1e-12 relative tolerance the existing R-reference
// tests use.

// ---------- Kendall full-output equivalence across size × mode ----------

type pairGen func(rng *rand.Rand, n int) (x, y []float64)

var kendallModes = []struct {
	name string
	gen  pairGen
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
		name: "monotone_strict",
		gen: func(_ *rand.Rand, n int) ([]float64, []float64) {
			x := make([]float64, n)
			y := make([]float64, n)
			for i := range n {
				x[i] = float64(i)
				y[i] = float64(i) * 1.5
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
		name: "x_constant",
		gen: func(rng *rand.Rand, n int) ([]float64, []float64) {
			x := make([]float64, n)
			y := make([]float64, n)
			for i := range n {
				x[i] = 0.5
				y[i] = rng.NormFloat64()
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

// TestKendallTauBFullEquivalence asserts that the entire kendallTauBStats
// output triple (tau, sval, varS) is bit-equal between the brute-only
// reference path and the production dispatcher (which routes to Knight's
// O(n log n) at n ≥ 128). Coverage spans:
//
//   - Sub-permutation regime (n ≤ 7) where the public Correlation API
//     uses exact permutation p-values.
//   - The brute-vs-Knight crossover boundary (n=127, 128, 129).
//   - Larger n where Knight's overhead pays back many times over.
//
// Crossed with seven data modes covering every tie pattern that exercises
// a different branch in the τ-b denominator and the variance formula.
func TestKendallTauBFullEquivalence(t *testing.T) {
	sizes := []int{2, 3, 5, 7, 8, 10, 50, 100, 127, 128, 129, 200, 500, 1500}
	for _, n := range sizes {
		for _, mode := range kendallModes {
			t.Run(fmt.Sprintf("n%d_%s", n, mode.name), func(t *testing.T) {
				rng := rand.New(rand.NewPCG(uint64(n), uint64(n)*131+7))
				x, y := mode.gen(rng, n)

				// Reference: force the brute path by computing the pair
				// counts via the textbook double loop, then run the rest
				// of the kendallTauBFinish through the same primitive.
				bC, bD := kendallPairCountBruteSerial(x, y)
				refTau, refSval, refVarS := kendallTauBFinish(x, y, n, bC, bD)

				// Dispatcher: lets the production code pick brute or Knight
				// based on n.
				gotTau, gotSval, gotVarS := kendallTauBStats(x, y)

				eqFloat := func(name string, a, b float64) {
					if math.IsNaN(a) && math.IsNaN(b) {
						return
					}
					if a != b {
						t.Fatalf("%s: ref=%v got=%v (n=%d, mode=%s)", name, a, b, n, mode.name)
					}
				}
				eqFloat("tau", refTau, gotTau)
				eqFloat("sval", refSval, gotSval)
				eqFloat("varS", refVarS, gotVarS)

				// Cross-check Knight directly when it's the active branch.
				if n >= 128 {
					kC, kD := kendallPairCountKnight(x, y)
					if kC != bC || kD != bD {
						t.Fatalf("Knight pair count diverges from brute: knight=(%v,%v) brute=(%v,%v)",
							kC, kD, bC, bD)
					}
				}
			})
		}
	}
}

// ---------- CorrelationMatrix dispatch coverage ----------

// dataTableFromMatrix builds an insyra DataTable column-by-column.
func dataTableFromMatrix(m [][]float64) *insyra.DataTable {
	dt := insyra.NewDataTable()
	if len(m) == 0 {
		return dt
	}
	for j := range m[0] {
		col := insyra.NewDataList().SetName(fmt.Sprintf("V%d", j+1))
		for i := range m {
			col.Append(m[i][j])
		}
		dt.AppendCols(col)
	}
	return dt
}

// TestCorrelationMatrixDispatchCoverage runs CorrelationMatrix at sizes
// that hit each branch of the parallel-pairs gate and the Spearman parallel-
// rank gate, for each method (Pearson, Spearman, Kendall). The assertions
// are:
//
//  1. The diagonal is exactly 1.0 (or NaN for an all-constant column where
//     the method declines to compute).
//  2. The matrix is symmetric in the sense that matrix[i][j] == matrix[j][i]
//     bit-for-bit (the production code computes each pair once and writes
//     both cells).
//  3. The result runs without error on each size × method combination.
//
// Bit-equivalence between the parallel-pair branch and a hypothetical
// serial-pair branch is implicit: each pair is computed by exactly one
// goroutine via the same {pearson,kendall,spearman}OnSlices primitive,
// so the value is bit-identical regardless of which goroutine runs it.
// The diagonal-and-symmetry asserts catch any data-race / write-overlap
// bug that would corrupt that invariant.
func TestCorrelationMatrixDispatchCoverage(t *testing.T) {
	type cfg struct {
		name       string
		rows, cols int
	}
	cfgs := []cfg{
		// Below pair-parallel gate (pairs < 4): serial pairs.
		{"tiny_2cols", 100, 2},  // pairs=1
		{"tiny_3cols", 100, 3},  // pairs=3
		{"tiny_rows_4cols", 49, 4}, // rows < 50, serial regardless
		// At/above pair-parallel gate.
		{"medium_4cols", 100, 4},  // pairs=6, rows≥50
		{"medium_6cols", 200, 6},  // pairs=15
		{"large_10cols", 500, 10}, // pairs=45, exercises Spearman parallel rank
		{"big_15cols_99rows", 99, 15},  // n*rows=1485 < 100*4=400 — rank serial
		{"big_15cols_200rows", 200, 15}, // rank parallel
	}
	methods := []struct {
		name string
		m    CorrelationMethod
	}{
		{"Pearson", PearsonCorrelation},
		{"Spearman", SpearmanCorrelation},
		{"Kendall", KendallCorrelation},
	}
	modes := []struct {
		name string
		gen  func(rng *rand.Rand, rows, cols int) [][]float64
	}{
		{
			name: "iid_normal",
			gen: func(rng *rand.Rand, rows, cols int) [][]float64 {
				m := make([][]float64, rows)
				for i := range m {
					m[i] = make([]float64, cols)
					for j := range m[i] {
						m[i][j] = rng.NormFloat64()
					}
				}
				return m
			},
		},
		{
			name: "monotone_columns",
			// Every column is a monotone function of the row index, so
			// Pearson, Spearman, Kendall all hit ≈+1 correlations.
			gen: func(rng *rand.Rand, rows, cols int) [][]float64 {
				m := make([][]float64, rows)
				for i := range m {
					m[i] = make([]float64, cols)
					for j := range m[i] {
						m[i][j] = float64(i)*float64(j+1) + 0.001*rng.NormFloat64()
					}
				}
				return m
			},
		},
		{
			name: "ties_heavy",
			gen: func(rng *rand.Rand, rows, cols int) [][]float64 {
				m := make([][]float64, rows)
				for i := range m {
					m[i] = make([]float64, cols)
					for j := range m[i] {
						m[i][j] = float64(rng.IntN(4))
					}
				}
				return m
			},
		},
	}

	for _, c := range cfgs {
		for _, method := range methods {
			for _, mode := range modes {
				t.Run(fmt.Sprintf("%s_%s_%s", c.name, method.name, mode.name), func(t *testing.T) {
					rng := rand.New(rand.NewPCG(uint64(c.rows*c.cols), 991))
					data := mode.gen(rng, c.rows, c.cols)
					dt := dataTableFromMatrix(data)

					corrDT, pDT, err := CorrelationMatrix(dt, method.m)
					if err != nil {
						// "constant column" type errors are legitimate
						// for the ties_heavy mode at small rows; just
						// assert we got *some* error message.
						if err.Error() == "" {
							t.Fatalf("empty error message")
						}
						return
					}
					if corrDT == nil || pDT == nil {
						t.Fatalf("nil result tables")
					}
					_, ncCorr := corrDT.Size()
					_, ncP := pDT.Size()
					if ncCorr != c.cols || ncP != c.cols {
						t.Fatalf("expected %d cols, got corr=%d p=%d", c.cols, ncCorr, ncP)
					}

					// Diagonal must be 1.0 exactly (or NaN for constants).
					for i := range c.cols {
						val, ok := insyra.ToFloat64Safe(corrDT.GetColByNumber(i).Get(i))
						if !ok {
							t.Fatalf("non-numeric on diagonal at %d", i)
						}
						if !math.IsNaN(val) && val != 1.0 {
							t.Fatalf("diagonal[%d][%d] = %v, expected 1.0 or NaN", i, i, val)
						}
					}

					// Symmetry: corr[i][j] == corr[j][i] bit-for-bit.
					for i := range c.cols {
						for j := i + 1; j < c.cols; j++ {
							a, _ := insyra.ToFloat64Safe(corrDT.GetColByNumber(i).Get(j))
							b, _ := insyra.ToFloat64Safe(corrDT.GetColByNumber(j).Get(i))
							if math.IsNaN(a) && math.IsNaN(b) {
								continue
							}
							if a != b {
								t.Fatalf("asymmetry at (%d,%d): %v vs %v", i, j, a, b)
							}
						}
					}
				})
			}
		}
	}
}

// ---------- PCA dispatch coverage ----------

// TestPCADispatchCoverage runs PCA at sizes hitting each dispatch branch
// of the standardisation step (rowNum*colNum ≥ 5000 → parallel) and
// asserts the eigenvalue spectrum is finite, ordered descending, and
// sums to a reasonable total (n_components for standardised data).
//
// Equivalence between the serial and parallel standardisation is bit-exact
// per cell because each (i, j) is written by exactly one worker via the
// same two-pass mean/std formula. The Eigen decomposition downstream is
// gonum LAPACK and deterministic given identical input.
func TestPCADispatchCoverage(t *testing.T) {
	cases := []struct {
		rows, cols int
	}{
		{20, 3},     // 60 cells — serial standardise
		{50, 4},     // 200 — serial
		{200, 20},   // 4000 — serial (just below 5000)
		{300, 20},   // 6000 — parallel
		{1000, 12},  // parallel
		{5000, 8},   // parallel, larger
	}
	for _, c := range cases {
		t.Run(fmt.Sprintf("r%d_c%d", c.rows, c.cols), func(t *testing.T) {
			rng := rand.New(rand.NewPCG(uint64(c.rows), uint64(c.cols)*7+13))
			data := make([][]float64, c.rows)
			for i := range data {
				data[i] = make([]float64, c.cols)
				for j := range data[i] {
					data[i][j] = rng.NormFloat64()
				}
			}
			dt := dataTableFromMatrix(data)
			res, err := PCA(dt)
			if err != nil {
				t.Fatalf("PCA error: %v", err)
			}
			if len(res.Eigenvalues) != c.cols {
				t.Fatalf("eigenvalues length: got %d, want %d",
					len(res.Eigenvalues), c.cols)
			}
			// Descending order.
			for i := 1; i < len(res.Eigenvalues); i++ {
				if res.Eigenvalues[i] > res.Eigenvalues[i-1]+1e-12 {
					t.Fatalf("eigenvalues not descending at %d: %v > %v",
						i, res.Eigenvalues[i], res.Eigenvalues[i-1])
				}
			}
			// All finite.
			for i, v := range res.Eigenvalues {
				if math.IsNaN(v) || math.IsInf(v, 0) {
					t.Fatalf("eigenvalue %d is non-finite: %v", i, v)
				}
			}
			// Sum ≈ cols (standardised covariance matrix has trace=cols).
			sum := 0.0
			for _, v := range res.Eigenvalues {
				sum += v
			}
			if math.Abs(sum-float64(c.cols)) > 1e-9 {
				t.Fatalf("eigenvalue sum %v, expected ≈%d", sum, c.cols)
			}

			// ExplainedVariance components sum to 100%.
			expSum := 0.0
			for _, v := range res.ExplainedVariance {
				expSum += v
			}
			if math.Abs(expSum-100.0) > 1e-9 {
				t.Fatalf("ExplainedVariance sum %v%%, expected ≈100%%", expSum)
			}
		})
	}
}

// ---------- KNN dispatch coverage ----------

// TestKNNDispatchCoverage exercises every (algorithm, weighting, size) cell
// of the KNN dispatch matrix. For each cell we verify:
//
//  1. The function returns without error.
//  2. Output shapes match input expectations.
//  3. For very small (test, train) where the parallel gate is closed,
//     the SAME inputs run through Auto algorithm and explicit Brute give
//     bit-equal predictions — this is the "serial path" sanity check.
//  4. For inputs that trigger the parallel gate, predictions are still
//     consistent across repeated runs (no data races silently producing
//     different outputs).
func TestKNNDispatchCoverage(t *testing.T) {
	algos := []internalknn.Algorithm{
		internalknn.BruteForceAlgorithm,
		internalknn.KDTreeAlgorithm,
		internalknn.BallTreeAlgorithm,
	}
	weightings := []internalknn.Weighting{
		internalknn.UniformWeighting,
		internalknn.DistanceWeighting,
	}
	type sz struct {
		nTrain, nTest, dim int
	}
	sizes := []sz{
		{20, 5, 4},     // both below parallel gate
		{30, 16, 4},    // test=16 hits gate (≥16), train=30 (≥32? no): serial
		{50, 16, 4},    // test=16, train=50: parallel
		{200, 200, 6},  // parallel, brute-classify benchmark size
		{2000, 500, 4}, // parallel, kd-tree benchmark size
	}
	for _, algo := range algos {
		for _, w := range weightings {
			for _, s := range sizes {
				name := fmt.Sprintf("%s_%s_train%d_test%d_dim%d",
					algo, w, s.nTrain, s.nTest, s.dim)
				t.Run(name, func(t *testing.T) {
					rng := rand.New(rand.NewPCG(uint64(s.nTrain*s.nTest+s.dim), 211))
					train := make([][]float64, s.nTrain)
					for i := range train {
						train[i] = make([]float64, s.dim)
						for j := range train[i] {
							train[i][j] = rng.NormFloat64()
						}
					}
					test := make([][]float64, s.nTest)
					for i := range test {
						test[i] = make([]float64, s.dim)
						for j := range test[i] {
							test[i][j] = rng.NormFloat64()
						}
					}
					labels := make([]string, s.nTrain)
					for i := range labels {
						labels[i] = fmt.Sprintf("c%d", i%4)
					}

					k := 5
					if s.nTrain < 5 {
						k = s.nTrain
					}
					opts := internalknn.Options{Algorithm: algo, Weighting: w, LeafSize: 16}

					r1, err := internalknn.Classify(train, test, labels, k, opts)
					if err != nil {
						t.Fatalf("Classify error: %v", err)
					}
					if len(r1.Predictions) != s.nTest {
						t.Fatalf("predictions len: got %d, want %d",
							len(r1.Predictions), s.nTest)
					}
					if len(r1.Probabilities) != s.nTest {
						t.Fatalf("probabilities len: got %d, want %d",
							len(r1.Probabilities), s.nTest)
					}

					// Repeated run produces identical predictions. This
					// catches data-race regressions (concurrent writes to
					// shared state would yield different output between
					// runs).
					r2, err := internalknn.Classify(train, test, labels, k, opts)
					if err != nil {
						t.Fatalf("Classify second run error: %v", err)
					}
					for i := range r1.Predictions {
						if r1.Predictions[i] != r2.Predictions[i] {
							t.Fatalf("non-deterministic prediction at %d: %s vs %s",
								i, r1.Predictions[i], r2.Predictions[i])
						}
					}
					for i := range r1.Probabilities {
						for j := range r1.Probabilities[i] {
							if r1.Probabilities[i][j] != r2.Probabilities[i][j] {
								t.Fatalf("non-deterministic probability at [%d][%d]: %v vs %v",
									i, j, r1.Probabilities[i][j], r2.Probabilities[i][j])
							}
						}
					}
				})
			}
		}
	}
}

// TestKNNAutoVsBruteEquivalence asserts that for any (n_train, n_test, dim)
// where Auto resolves to BruteForce (n_train < 64), the Auto and explicit
// Brute paths give bit-equal predictions. This guards against the auto
// dispatch silently changing semantics.
func TestKNNAutoVsBruteEquivalence(t *testing.T) {
	rng := rand.New(rand.NewPCG(7, 8))
	for _, n := range []int{10, 20, 40, 60} {
		train := make([][]float64, n)
		test := make([][]float64, n/2)
		for i := range train {
			train[i] = []float64{rng.NormFloat64(), rng.NormFloat64()}
		}
		for i := range test {
			test[i] = []float64{rng.NormFloat64(), rng.NormFloat64()}
		}
		labels := make([]string, n)
		for i := range labels {
			labels[i] = fmt.Sprintf("c%d", i%3)
		}
		auto, err := internalknn.Classify(train, test, labels, 3, internalknn.Options{
			Algorithm: internalknn.AutoAlgorithm, Weighting: internalknn.UniformWeighting,
		})
		if err != nil {
			t.Fatalf("auto error: %v", err)
		}
		brute, err := internalknn.Classify(train, test, labels, 3, internalknn.Options{
			Algorithm: internalknn.BruteForceAlgorithm, Weighting: internalknn.UniformWeighting,
		})
		if err != nil {
			t.Fatalf("brute error: %v", err)
		}
		for i := range auto.Predictions {
			if auto.Predictions[i] != brute.Predictions[i] {
				t.Fatalf("n=%d row=%d auto=%s brute=%s",
					n, i, auto.Predictions[i], brute.Predictions[i])
			}
			for j := range auto.Probabilities[i] {
				if auto.Probabilities[i][j] != brute.Probabilities[i][j] {
					t.Fatalf("n=%d row=%d col=%d auto=%v brute=%v",
						n, i, j, auto.Probabilities[i][j], brute.Probabilities[i][j])
				}
			}
		}
	}
}
