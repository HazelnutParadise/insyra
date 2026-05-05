package stats

import (
	"cmp"
	"errors"
	"fmt"
	"math"
	"sort"
	"sync"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/internal/algorithms"
	"github.com/HazelnutParadise/insyra/stats/internal/parutil"
	"gonum.org/v1/gonum/mat"
	"gonum.org/v1/gonum/stat"
)

// CorrelationMethod specifies which correlation coefficient to compute.
type CorrelationMethod int

const (
	PearsonCorrelation CorrelationMethod = iota
	KendallCorrelation
	SpearmanCorrelation
)

// CorrelationAnalysis calculates correlation matrix, p-value matrix and Bartlett test.
func CorrelationAnalysis(dataTable insyra.IDataTable, method CorrelationMethod) (corrMatrix *insyra.DataTable, pMatrix *insyra.DataTable, chiSquare float64, pValue float64, df int, err error) {
	dataTable.AtomicDo(func(dt *insyra.DataTable) {
		corrMatrix, pMatrix, err = CorrelationMatrix(dt, method)
		if err != nil {
			return
		}

		if method == PearsonCorrelation {
			chiSquare, pValue, df, err = BartlettSphericity(dt)
		} else {
			chiSquare, pValue, df = math.NaN(), math.NaN(), 0
		}
	})

	return corrMatrix, pMatrix, chiSquare, pValue, df, err
}

// CorrelationMatrix calculates the correlation coefficient matrix and p-value matrix.
//
// Performance notes: extracts every column's float64 slice exactly once
// (dropping the previous O(C²) ToF64Slice + AtomicDo work), pre-computes
// per-column ranks for Spearman, then computes the C(C-1)/2 unique pairs in
// parallel on those cached slices. Result is bit-identical to the previous
// serial path: the underlying primitives are the same, only the order /
// granularity of work differs.
func CorrelationMatrix(dataTable insyra.IDataTable, method CorrelationMethod) (corrMatrix *insyra.DataTable, pMatrix *insyra.DataTable, err error) {
	dt := insyra.NewDataTable()
	pdt := insyra.NewDataTable()

	var n int
	var colSlices [][]float64
	var colNames []string
	dataTable.AtomicDo(func(table *insyra.DataTable) {
		_, n = table.Size()
		if n < 2 {
			err = errors.New("need at least two columns for correlation")
			return
		}
		colSlices = make([][]float64, n)
		colNames = make([]string, n)
		for i := range n {
			c := table.GetColByNumber(i)
			c.AtomicDo(func(dl *insyra.DataList) {
				colSlices[i] = dl.ToF64Slice()
				colNames[i] = dl.GetName()
			})
		}
	})
	if err != nil {
		return nil, nil, err
	}

	matrix := make([][]float64, n)
	pmatrix := make([][]float64, n)
	for i := range matrix {
		matrix[i] = make([]float64, n)
		pmatrix[i] = make([]float64, n)
		matrix[i][i] = 1.0
	}

	// For Spearman the rank vectors are reused across every pair the column
	// participates in, so we precompute them once. The ranking algorithm
	// matches DataList.Rank exactly (stable sort + average-rank ties).
	rows := 0
	if n > 0 {
		rows = len(colSlices[0])
	}
	var ranks [][]float64
	if method == SpearmanCorrelation {
		ranks = make([][]float64, n)
		// Per-column rank cost is O(rows·log rows). Parallel pays off once
		// rows·n exceeds a few thousand "work units" — gate generously since
		// the rank step is typically a small fraction of the pairs phase.
		parutil.Run(n, n >= 4 && rows >= 100, func(i int) {
			ranks[i] = rankSliceAverage(colSlices[i])
		})
	}

	pairs := n * (n - 1) / 2
	pairErrs := make([]error, pairs)
	pairAt := func(p int) (int, int) {
		// Map linear pair index p ∈ [0, n*(n-1)/2) to (i, j) with i < j.
		i := 0
		for {
			rowSize := n - i - 1
			if p < rowSize {
				return i, i + 1 + p
			}
			p -= rowSize
			i++
		}
	}

	// Per-pair cost: Pearson/Spearman are O(rows) (~5µs at rows=500). Kendall
	// is O(rows²) so it always wants parallel as soon as we have ≥2 pairs.
	// Pearson/Spearman want pairs ≥ 4 to amortise goroutine launch (~18µs at
	// 24 workers). rows ≥ 50 ensures per-pair has actual work.
	goPar := pairs >= 2 && rows >= 50
	if method != KendallCorrelation {
		goPar = pairs >= 4 && rows >= 50
	}
	parutil.Run(pairs, goPar, func(p int) {
		i, j := pairAt(p)
		var statVal, pval float64
		var perr error
		switch method {
		case PearsonCorrelation:
			statVal, pval, perr = pearsonOnSlices(colSlices[i], colSlices[j])
		case KendallCorrelation:
			statVal, pval, perr = kendallOnSlices(colSlices[i], colSlices[j])
		case SpearmanCorrelation:
			statVal, pval, perr = spearmanOnSlices(colSlices[i], colSlices[j], ranks[i], ranks[j])
		default:
			perr = errors.New("unsupported method")
		}
		if perr != nil || math.IsNaN(statVal) {
			matrix[i][j] = math.NaN()
			matrix[j][i] = math.NaN()
			pmatrix[i][j] = math.NaN()
			pmatrix[j][i] = math.NaN()
			if perr != nil {
				pairErrs[p] = fmt.Errorf("unable to calculate correlation for columns %d and %d: %w", i, j, perr)
			} else {
				pairErrs[p] = fmt.Errorf("unable to calculate correlation for columns %d and %d", i, j)
			}
			return
		}
		matrix[i][j] = statVal
		matrix[j][i] = statVal
		pmatrix[i][j] = pval
		pmatrix[j][i] = pval
	})

	var pairErr error
	for _, e := range pairErrs {
		if e != nil {
			pairErr = e
			break
		}
	}

	dtColNames := insyra.NewDataList()
	for i := range matrix {
		row := insyra.NewDataList().SetName(colNames[i])
		dtColNames.Append(colNames[i])
		for j := range matrix[i] {
			row.Append(matrix[i][j])
		}
		dt.AppendRowsFromDataList(row)
	}
	dt.AppendRowsFromDataList(dtColNames)
	dt.SetRowToColNames(-1)

	pdtColNames := insyra.NewDataList()
	for i := range pmatrix {
		row := insyra.NewDataList().SetName(colNames[i])
		pdtColNames.Append(colNames[i])
		for j := range pmatrix[i] {
			row.Append(pmatrix[i][j])
		}
		pdt.AppendRowsFromDataList(row)
	}
	pdt.AppendRowsFromDataList(pdtColNames)
	pdt.SetRowToColNames(-1)

	return dt, pdt, pairErr
}

// rankSliceAverage replicates DataList.Rank's average-rank-on-ties behaviour
// on a raw []float64 — used by CorrelationMatrix's Spearman path so each
// column is ranked exactly once instead of once per pair.
func rankSliceAverage(data []float64) []float64 {
	n := len(data)
	if n == 0 {
		return nil
	}
	indexes := make([]int, n)
	for i := range indexes {
		indexes[i] = i
	}
	// algorithms.ParallelSortStableFunc is generic (no reflection) and
	// auto-falls-back to slices.SortStableFunc below n=4910. So this is
	// strictly faster than sort.SliceStable at every size: small n loses
	// the reflection overhead, large n picks up parallel sort for free.
	algorithms.ParallelSortStableFunc(indexes, func(a, b int) int {
		return cmp.Compare(data[a], data[b])
	})
	ranked := make([]float64, n)
	for i := 0; i < n; {
		sumRank := 0.0
		count := 0
		val := data[indexes[i]]
		for j := i; j < n && data[indexes[j]] == val; j++ {
			sumRank += float64(j + 1)
			count++
		}
		avgRank := sumRank / float64(count)
		for k := range count {
			ranked[indexes[i+k]] = avgRank
		}
		i += count
	}
	return ranked
}

func pearsonOnSlices(x, y []float64) (statVal, pValue float64, err error) {
	n := len(x)
	if n != len(y) || n < 2 {
		return math.NaN(), math.NaN(), errors.New("invalid input length or insufficient data")
	}
	// gonum's Variance is two-pass with n-1 divisor — matches DataList.Var.
	varX := stat.Variance(x, nil)
	varY := stat.Variance(y, nil)
	if varX == 0 || varY == 0 {
		return math.NaN(), math.NaN(), errors.New("cannot calculate correlation due to zero variance")
	}
	cov := stat.Covariance(x, y, nil)
	r := cov / math.Sqrt(varX*varY)
	res := CorrelationResult{}
	res.Statistic = r
	populateCorrelationInference(&res, r, float64(n))
	return res.Statistic, res.PValue, nil
}

func kendallOnSlices(x, y []float64) (statVal, pValue float64, err error) {
	n := len(x)
	if n != len(y) || n < 2 {
		return math.NaN(), math.NaN(), errors.New("invalid input length or insufficient data")
	}
	// Mirror kendallCorrelationWithStats but on slices directly. The stdev
	// guard from the IDataList path is preserved: a constant axis means no
	// usable rank ordering, so τ-b is NaN.
	if !hasVarianceFloats(x) || !hasVarianceFloats(y) {
		return math.NaN(), math.NaN(), errors.New("cannot calculate correlation due to zero variance")
	}
	tau, sval, varS := kendallTauBStats(x, y)
	if math.IsNaN(tau) {
		return math.NaN(), math.NaN(), errors.New("cannot calculate correlation")
	}
	if n <= 7 {
		// Exact two-sided permutation p-value via Heap's algorithm:
		// iterate every permutation of y in place (no per-leaf allocation)
		// and count how many give |τ_b| ≥ observed.
		yCopy := append([]float64(nil), y...)
		sort.Float64s(yCopy)
		extreme := 0
		obs := math.Abs(tau)
		forEachPermutation(yCopy, func(perm []float64) {
			altTau, _, _ := kendallTauBStats(x, perm)
			if math.Abs(altTau) >= obs {
				extreme++
			}
		})
		return tau, float64(extreme) / float64(factorial(n)), nil
	}
	if varS <= 0 || math.IsNaN(varS) {
		return tau, math.NaN(), nil
	}
	z := sval / math.Sqrt(varS)
	return tau, 2 * (1 - zCDF(math.Abs(z))), nil
}

func spearmanOnSlices(rawX, rawY, rankX, rankY []float64) (statVal, pValue float64, err error) {
	n := len(rawX)
	if n != len(rawY) || n < 2 {
		return math.NaN(), math.NaN(), errors.New("invalid input length or insufficient data")
	}
	if !hasVarianceFloats(rawX) || !hasVarianceFloats(rawY) {
		return math.NaN(), math.NaN(), errors.New("cannot calculate correlation due to zero variance")
	}
	varRX := stat.Variance(rankX, nil)
	varRY := stat.Variance(rankY, nil)
	if varRX == 0 || varRY == 0 {
		return math.NaN(), math.NaN(), errors.New("cannot calculate correlation due to zero variance")
	}
	covR := stat.Covariance(rankX, rankY, nil)
	rho := covR / math.Sqrt(varRX*varRY)
	if math.IsNaN(rho) {
		return math.NaN(), math.NaN(), errors.New("cannot calculate correlation")
	}
	nF := float64(n)
	df := nF - 2
	if df <= 0 {
		return rho, math.NaN(), nil
	}
	if hasTiesFloats(rawX) || hasTiesFloats(rawY) || nF > 1290 {
		return rho, tTwoTailedPValue(correlationToT(rho, nF), df), nil
	}
	return rho, spearmanPValueAS89(rho, n), nil
}

// hasVarianceFloats reports whether the slice has at least two distinct
// values (i.e. non-zero sample variance). Cheap O(n) replacement for
// computing Stdev() just to compare to zero.
func hasVarianceFloats(v []float64) bool {
	if len(v) < 2 {
		return false
	}
	first := v[0]
	for _, x := range v[1:] {
		if x != first {
			return true
		}
	}
	return false
}

func Covariance(dlX, dlY insyra.IDataList) (float64, error) {
	var lenX, lenY int
	var dataX, dataY []float64
	dlX.AtomicDo(func(dlx *insyra.DataList) {
		dlY.AtomicDo(func(dly *insyra.DataList) {
			lenX = dlx.Len()
			lenY = dly.Len()
			dataX = dlx.ToF64Slice()
			dataY = dly.ToF64Slice()
		})
	})
	if lenX != lenY {
		return math.NaN(), errors.New("input lengths must match")
	}
	if lenX < 2 {
		return math.NaN(), errors.New("at least two observations are required")
	}
	// gonum/stat.Covariance returns the sample covariance with n-1 divisor when
	// weights == nil — same definition as the previous hand-rolled formula.
	return stat.Covariance(dataX, dataY, nil), nil
}

type CorrelationResult struct {
	testResultBase
}

// Correlation calculates correlation between two IDataLists.
func Correlation(dlX, dlY insyra.IDataList, method CorrelationMethod) (*CorrelationResult, error) {
	var result CorrelationResult
	var err error
	dlX.AtomicDo(func(dlx *insyra.DataList) {
		dlY.AtomicDo(func(dly *insyra.DataList) {
			lenX := dlx.Len()
			lenY := dly.Len()
			stdevX := dlx.Stdev()
			stdevY := dly.Stdev()

			if lenX != lenY || lenX < 2 {
				err = errors.New("invalid input length or insufficient data")
				return
			}
			if stdevX == 0 || stdevY == 0 {
				err = errors.New("cannot calculate correlation due to zero variance")
				return
			}

			switch method {
			case PearsonCorrelation:
				result, err = pearsonCorrelationWithStats(dlx, dly)
			case KendallCorrelation:
				result = kendallCorrelationWithStats(dlx, dly)
			case SpearmanCorrelation:
				result, err = spearmanCorrelationWithStats(dlx, dly)
			default:
				err = errors.New("unsupported method")
				return
			}
		})
	})
	if err != nil {
		return nil, err
	}

	if math.IsNaN(result.Statistic) {
		return nil, errors.New("cannot calculate correlation")
	}

	return &result, nil
}

// BartlettSphericity performs Bartlett's test of sphericity.
func BartlettSphericity(dataTable insyra.IDataTable) (chiSquare float64, pValue float64, df int, err error) {
	var rows, cols int
	var corrMatrix [][]float64
	dataTable.AtomicDo(func(dt *insyra.DataTable) {
		rows, cols = dt.Size()
		if cols < 2 {
			err = errors.New("need at least two columns for sphericity test")
			return
		}

		corrMatrix = make([][]float64, cols)
		for i := range corrMatrix {
			corrMatrix[i] = make([]float64, cols)
			for j := range corrMatrix[i] {
				if i == j {
					corrMatrix[i][j] = 1.0
				} else {
					col1 := dt.GetColByNumber(i)
					col2 := dt.GetColByNumber(j)
					result, corrErr := Correlation(col1, col2, PearsonCorrelation)
					if corrErr == nil && result != nil && !math.IsNaN(result.Statistic) {
						corrMatrix[i][j] = result.Statistic
					} else {
						if corrErr != nil {
							err = fmt.Errorf("failed to calculate Pearson correlation for columns %d and %d: %w", i, j, corrErr)
						} else {
							err = fmt.Errorf("failed to calculate Pearson correlation for columns %d and %d", i, j)
						}
						return
					}
				}
			}
		}
	})
	if err != nil {
		return math.NaN(), math.NaN(), 0, err
	}

	// Flatten corrMatrix into row-major storage for gonum/mat (LU-based det,
	// numerically more stable than the previous hand-rolled Gauss elimination
	// in stats/internal/linalg).
	cm := mat.NewDense(cols, cols, nil)
	for i := range corrMatrix {
		for j := range corrMatrix[i] {
			cm.Set(i, j, corrMatrix[i][j])
		}
	}
	det := mat.Det(cm)
	if det <= 0 {
		return math.NaN(), math.NaN(), 0, errors.New("correlation matrix is singular or not positive definite")
	}

	n := float64(rows)
	p := float64(cols)
	chisq := -((n - 1) - (2*p+5)/6) * math.Log(det)
	degreesOfFreedom := int((p * (p - 1)) / 2)
	pval := chiSquaredPValue(chisq, float64(degreesOfFreedom))

	return chisq, pval, degreesOfFreedom, nil
}

func pearsonCorrelation(dlX, dlY insyra.IDataList) (float64, error) {
	var stdX, stdY float64
	var cov float64
	var err error
	dlX.AtomicDo(func(dlx *insyra.DataList) {
		dlY.AtomicDo(func(dly *insyra.DataList) {
			cov, err = Covariance(dlx, dly)
			stdX = dlx.Stdev()
			stdY = dly.Stdev()
		})
	})
	if err != nil {
		return math.NaN(), err
	}
	if stdX == 0 || stdY == 0 {
		return math.NaN(), errors.New("cannot calculate correlation due to zero variance")
	}
	return cov / (stdX * stdY), nil
}

func pearsonCorrelationWithStats(dlX, dlY insyra.IDataList) (CorrelationResult, error) {
	result := CorrelationResult{}
	var corr, n float64
	var err error
	dlX.AtomicDo(func(dlx *insyra.DataList) {
		dlY.AtomicDo(func(dly *insyra.DataList) {
			corr, err = pearsonCorrelation(dlx, dly)
			result.Statistic = corr
			n = float64(dlx.Len())
		})
	})
	if err != nil {
		return result, err
	}
	if math.IsNaN(corr) {
		result.PValue = math.NaN()
		result.CI = nanCIPtr()
	} else {
		populateCorrelationInference(&result, corr, n)
	}
	return result, nil
}

// kendallTauBStats computes Kendall's τ-b along with the S-statistic and the
// asymptotic variance with full tie correction.
//
// Why we don't use gonum/stat.Kendall: gonum returns τ-a, defined as
// (concordant − discordant) / (n choose 2). That divisor doesn't react to
// ties in either variable, so |τ_a| can fall short of 1 even for a perfectly
// monotonic mapping when the data has ties — and the value disagrees with
// R's cor(method="kendall"), SciPy's kendalltau, and SPSS, which all default
// to τ-b. We implement τ-b directly here.
//
// Two pair-counting strategies, dispatched by n based on calibration data
// (BenchmarkCalib_KendallStrategies on 24 threads):
//   - n < 128 : serial brute O(n²). Tightest inner loop, smallest constant
//               — at n≤96 this beats every alternative including Knight.
//   - n ≥ 128 : Knight's O(n log n) algorithm via mergesort inversion
//               counting. Wins by 5% at n=128, 38% at n=192, 3.5× at n=1024,
//               12× at n=8192.
//
// Note: a parallel brute O(n²) variant exists in this file
// (kendallPairCountBruteParallel) but is NOT on the dispatch path.
// Calibration shows Knight's serial implementation beats parallel brute on
// 24 threads at every n where parallel brute would otherwise be picked
// (e.g. n=512: knight 42µs, brute-par 110µs; n=2048: knight 268µs,
// brute-par 1.16ms). Spending 24 cores on n² work that Knight does in
// O(n log n) on one core is just bad triage. The parallel brute remains
// callable for the kendall_knight_test.go equivalence check.
//
// All branches return identical (nC, nD) — they're integer pair counts
// of the same combinatorial definition, just enumerated differently — so
// τ, sval and varS are bit-identical regardless of which branch fires.
//
// Returns:
//
//	tau  — Kendall's τ-b in [-1, 1] (NaN if either axis is all-tied)
//	sval — concordant minus discordant pair count (the "S" statistic)
//	varS — variance of S under H₀ with tie correction (Conover 1999, eq. 5.16)
func kendallTauBStats(x, y []float64) (tau, sval, varS float64) {
	n := len(x)
	if n < 2 {
		return math.NaN(), 0, math.NaN()
	}
	var nC, nD float64
	if n >= 128 {
		nC, nD = kendallPairCountKnight(x, y)
	} else {
		nC, nD = kendallPairCountBruteSerial(x, y)
	}
	return kendallTauBFinish(x, y, n, nC, nD)
}

// kendallPairCountBruteSerial is the textbook O(n²) double loop. Used as the
// reference and as the production path for small n where the loop body's
// constant factor (≈5ns/pair) beats Knight's mergesort overhead.
func kendallPairCountBruteSerial(x, y []float64) (nC, nD float64) {
	n := len(x)
	for i := range n {
		xi, yi := x[i], y[i]
		for j := i + 1; j < n; j++ {
			dx := xi - x[j]
			dy := yi - y[j]
			if dx == 0 || dy == 0 {
				continue
			}
			if (dx > 0) == (dy > 0) {
				nC++
			} else {
				nD++
			}
		}
	}
	return
}

// kendallPairCountBruteParallel mirrors the serial brute count but spreads
// the outer i-loop across goroutines with worker-local int accumulators.
// Sums are integer-valued so the reduction order doesn't perturb the result.
func kendallPairCountBruteParallel(x, y []float64) (nC, nD float64) {
	n := len(x)
	workers := parutil.MaxWorkers(n)
	if workers <= 1 {
		return kendallPairCountBruteSerial(x, y)
	}
	cArr := make([]float64, workers)
	dArr := make([]float64, workers)
	chunk := (n + workers - 1) / workers
	var wg sync.WaitGroup
	for w := range workers {
		start := w * chunk
		if start >= n {
			break
		}
		end := min(start+chunk, n)
		wg.Add(1)
		go func(w, start, end int) {
			defer wg.Done()
			var lc, ld float64
			for i := start; i < end; i++ {
				xi, yi := x[i], y[i]
				for j := i + 1; j < n; j++ {
					dx := xi - x[j]
					dy := yi - y[j]
					if dx == 0 || dy == 0 {
						continue
					}
					if (dx > 0) == (dy > 0) {
						lc++
					} else {
						ld++
					}
				}
			}
			cArr[w] = lc
			dArr[w] = ld
		}(w, start, end)
	}
	wg.Wait()
	for i := range workers {
		nC += cArr[i]
		nD += dArr[i]
	}
	return
}

// kendallPairCountKnight implements Knight (1966)'s O(n log n) algorithm.
// Reference: "A computer method for calculating Kendall's tau with ungrouped
// data", JASA 61(314): 436–439.
//
// Method:
//  1. Lex-sort pairs by (x, y).
//  2. Walk the sorted array to count tie groups: n1 = pairs tied in x,
//     n_xy = pairs tied in both axes (the lex sort makes (x,y)-tie groups
//     contiguous within x-tie groups).
//  3. Extract the y-vector in x-sorted order, count its inversions via
//     mergesort. Because the secondary sort orders y-ascending within
//     x-ties, every inversion (i<j, y_i>y_j) has x_i<x_j — i.e. each is
//     exactly one discordant pair. So discordant = inversions.
//  4. After the mergesort the y-vector is sorted; walk it to count
//     n2 = pairs tied in y.
//  5. nC = n0 − n1 − n2 + n_xy − discordant   (derived in the comment
//     block below).
//
// Derivation. After lex-sort, every pair (i<j) falls into:
//
//	A: x_i<x_j ∧ y_i<y_j  (concordant)
//	B: x_i<x_j ∧ y_i>y_j  (discordant) = inversions
//	C: x_i<x_j ∧ y_i=y_j  (tied in y only)
//	D: x_i=x_j ∧ y_i<y_j  (tied in x only — secondary sort puts these here)
//	E: x_i=x_j ∧ y_i=y_j  (tied in both)
//
// Then n1 = D+E, n2 = C+E, n_xy = E, and A+B+C+D+E = n0, so
// A = n0 − B − C − D − E = n0 − inversions − (n2−E) − (n1−E) − E
//   = n0 − n1 − n2 + E − inversions.
func kendallPairCountKnight(x, y []float64) (nC, nD float64) {
	n := len(x)
	if n < 2 {
		return 0, 0
	}
	type pair struct{ x, y float64 }
	pairs := make([]pair, n)
	for i := range x {
		pairs[i] = pair{x[i], y[i]}
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].x != pairs[j].x {
			return pairs[i].x < pairs[j].x
		}
		return pairs[i].y < pairs[j].y
	})

	// Walk lex-sorted pairs, counting n1 (x-ties) and n_xy (xy-ties).
	var n1, eXY float64
	i := 0
	for i < n {
		j := i
		for j < n && pairs[j].x == pairs[i].x {
			j++
		}
		if t := j - i; t > 1 {
			tt := float64(t)
			n1 += tt * (tt - 1) / 2
			// Within this x-tie group, count y-tie subgroups (secondary
			// sort means equal y's are contiguous here).
			k := i
			for k < j {
				l := k
				for l < j && pairs[l].y == pairs[k].y {
					l++
				}
				if tt := float64(l - k); tt > 1 {
					eXY += tt * (tt - 1) / 2
				}
				k = l
			}
		}
		i = j
	}

	// Project y into x-sorted order, count inversions (= discordant), and
	// pick up the y-sort as a side effect of mergesort.
	ys := make([]float64, n)
	for i := range pairs {
		ys[i] = pairs[i].y
	}
	aux := make([]float64, n)
	inversions := mergeCountInversions(ys, aux, 0, n-1)

	// ys is now sorted; walk it for n2 (y-ties).
	var n2 float64
	i = 0
	for i < n {
		j := i
		for j < n && ys[j] == ys[i] {
			j++
		}
		if t := j - i; t > 1 {
			tt := float64(t)
			n2 += tt * (tt - 1) / 2
		}
		i = j
	}

	n0 := float64(n*(n-1)) / 2
	nC = n0 - n1 - n2 + eXY - inversions
	nD = inversions
	return
}

// mergeCountInversions sorts a[lo:hi+1] in place via mergesort and returns
// the inversion count. aux must have len ≥ hi+1.
func mergeCountInversions(a, aux []float64, lo, hi int) float64 {
	if lo >= hi {
		return 0
	}
	mid := (lo + hi) / 2
	inv := mergeCountInversions(a, aux, lo, mid)
	inv += mergeCountInversions(a, aux, mid+1, hi)
	inv += mergeCountStep(a, aux, lo, mid, hi)
	return inv
}

func mergeCountStep(a, aux []float64, lo, mid, hi int) float64 {
	for k := lo; k <= hi; k++ {
		aux[k] = a[k]
	}
	var inv float64
	i, j := lo, mid+1
	for k := lo; k <= hi; k++ {
		switch {
		case i > mid:
			a[k] = aux[j]
			j++
		case j > hi:
			a[k] = aux[i]
			i++
		case aux[i] <= aux[j]:
			a[k] = aux[i]
			i++
		default:
			// aux[i] > aux[j]: every remaining element in left half [i..mid]
			// forms an inversion with aux[j].
			inv += float64(mid - i + 1)
			a[k] = aux[j]
			j++
		}
	}
	return inv
}

func kendallTauBFinish(x, y []float64, n int, nC, nD float64) (tau, sval, varS float64) {

	// Tie group sizes per axis. n1 = Σ t(t-1)/2 (pairs tied in x); same for y.
	// T1 = Σ t(t-1)(2t+5) appears in the variance formula's primary term.
	tieGroupSizes := func(z []float64) []int {
		s := append([]float64(nil), z...)
		sort.Float64s(s)
		var out []int
		for i := 0; i < len(s); {
			j := i + 1
			for j < len(s) && s[j] == s[i] {
				j++
			}
			if j-i > 1 {
				out = append(out, j-i)
			}
			i = j
		}
		return out
	}
	tx := tieGroupSizes(x)
	ty := tieGroupSizes(y)
	var n1, n2 float64                  // Σ t(t-1)/2  — for τ-b denominator
	var T1, T2 float64                  // Σ t(t-1)(2t+5)
	var T1b, T2b float64                // Σ t(t-1)(t-2)
	var T1c, T2c float64                // Σ t(t-1)
	for _, t := range tx {
		tt := float64(t)
		t1 := tt - 1
		n1 += tt * t1 / 2
		T1 += tt * t1 * (2*tt + 5)
		T1b += tt * t1 * (tt - 2)
		T1c += tt * t1
	}
	for _, t := range ty {
		tt := float64(t)
		t1 := tt - 1
		n2 += tt * t1 / 2
		T2 += tt * t1 * (2*tt + 5)
		T2b += tt * t1 * (tt - 2)
		T2c += tt * t1
	}

	n0 := float64(n*(n-1)) / 2
	denom := math.Sqrt((n0 - n1) * (n0 - n2))
	if denom == 0 {
		tau = math.NaN()
	} else {
		tau = (nC - nD) / denom
	}

	sval = nC - nD
	nF := float64(n)
	// Full Kendall (1948) asymptotic variance under H₀ with both first-order
	// (T1, T2) and second-order (T1b·T2b, T1c·T2c) tie corrections. Matches
	// R's cor.test(method="kendall") exactly. SciPy's kendalltau historically
	// dropped the second-order terms (matching only T1/T2); we include them
	// so the p-value is correct for tied data with non-trivial tie structure.
	// Reduces to n(n−1)(2n+5)/18 — Kendall's classical no-ties formula —
	// when both axes are tie-free.
	base := nF*(nF-1)*(2*nF+5) - T1 - T2
	varS = base / 18
	if n >= 3 {
		varS += (T1b * T2b) / (9 * nF * (nF - 1) * (nF - 2))
	}
	if n >= 2 {
		varS += (T1c * T2c) / (2 * nF * (nF - 1))
	}
	return tau, sval, varS
}

func kendallCorrelationWithStats(dlX, dlY insyra.IDataList) CorrelationResult {
	result := CorrelationResult{}
	x := dlX.ToF64Slice()
	y := dlY.ToF64Slice()
	tau, sval, varS := kendallTauBStats(x, y)
	result.Statistic = tau

	n := len(x)
	if n <= 7 {
		// Exact two-sided permutation p-value via Heap's algorithm:
		// hold x fixed, iterate every permutation of y in place, count
		// how many give |τ_b| ≥ observed. Tau-b (not gonum's tau-a) is
		// used inside the loop too. The previous form copied each leaf
		// permutation; Heap's mutates y in place across n! visits with
		// zero per-leaf allocation.
		yCopy := append([]float64(nil), y...)
		sort.Float64s(yCopy)
		extreme := 0
		obs := math.Abs(tau)
		forEachPermutation(yCopy, func(perm []float64) {
			altTau, _, _ := kendallTauBStats(x, perm)
			if math.Abs(altTau) >= obs {
				extreme++
			}
		})
		result.PValue = float64(extreme) / float64(factorial(n))
	} else {
		// Asymptotic z = S / sqrt(var(S)). For no-ties data this is identical
		// to the previous Kendall formula 3·τ·sqrt(n(n−1)) / sqrt(2(2n+5)).
		// With ties, varS subtracts the tie-group contributions from both
		// axes — matching scipy.stats.kendalltau and R cor.test asymptotic.
		if varS <= 0 || math.IsNaN(varS) {
			result.PValue = math.NaN()
		} else {
			z := sval / math.Sqrt(varS)
			result.PValue = 2 * (1 - zCDF(math.Abs(z)))
		}
	}

	return result
}

// forEachPermutation iterates every permutation of arr in-place, calling
// fn with the current arrangement on each leaf. fn must NOT keep a reference
// to the slice — we mutate it back to a new arrangement on the next step.
//
// Heap's algorithm (1963): generates n! permutations using only single
// element swaps, no allocations per leaf. The previous DFS form copied the
// whole slice at every leaf (n! × n floats); for n=7 that was 5040 × 7 =
// 35280 allocated floats plus 5040 slice headers. Heap's allocates exactly
// zero per leaf — the caller's accumulator (e.g. counting how many leaves
// satisfy a predicate) is the only state.
func forEachPermutation(arr []float64, fn func([]float64)) {
	heapsPermute(len(arr), arr, fn)
}

func heapsPermute(k int, arr []float64, fn func([]float64)) {
	if k == 1 {
		fn(arr)
		return
	}
	// Recurse first, then swap: this is the iterative-friendly "even/odd"
	// form of Heap's algorithm. Each invocation produces every permutation
	// of the prefix arr[:k] exactly once.
	for i := 0; i < k; i++ {
		heapsPermute(k-1, arr, fn)
		if k%2 == 0 {
			arr[i], arr[k-1] = arr[k-1], arr[i]
		} else {
			arr[0], arr[k-1] = arr[k-1], arr[0]
		}
	}
}

// factorial returns n! for n in [0, 20]. Used only to compute the
// permutation-p-value denominator for Kendall n ≤ 7.
func factorial(n int) int {
	r := 1
	for i := 2; i <= n; i++ {
		r *= i
	}
	return r
}

func spearmanCorrelationWithStats(dlX, dlY insyra.IDataList) (CorrelationResult, error) {
	result := CorrelationResult{}
	var rankX, rankY insyra.IDataList
	var rawX, rawY []float64
	var n float64
	dlX.AtomicDo(func(dlx *insyra.DataList) {
		dlY.AtomicDo(func(dly *insyra.DataList) {
			rankX = dlx.Rank()
			rankY = dly.Rank()
			rawX = dlx.ToF64Slice()
			rawY = dly.ToF64Slice()
			n = float64(dlx.Len())
		})
	})
	rho, err := pearsonCorrelation(rankX, rankY)
	if err != nil {
		return result, err
	}
	result.Statistic = rho

	if math.IsNaN(rho) {
		result.PValue = math.NaN()
		return result, nil
	}

	df := n - 2
	result.DF = &df
	if df <= 0 {
		result.PValue = math.NaN()
		result.CI = nanCIPtr()
		return result, nil
	}

	// p-value: matches R cor.test(method="spearman"). For data without ties
	// and n ≤ 1290 (no integer overflow), R uses prho() — exact enumeration
	// for n ≤ 9, AS-89 Edgeworth-series approximation for n ≥ 10. With ties
	// or very large n, R falls back to Fisher r-to-t (which is also what
	// insyra used universally before).
	if hasTiesFloats(rawX) || hasTiesFloats(rawY) || n > 1290 {
		tStat := correlationToT(rho, n)
		result.PValue = tTwoTailedPValue(tStat, df)
	} else {
		result.PValue = spearmanPValueAS89(rho, int(n))
	}
	// CI uses Fisher z-transform (same as Pearson — R does not return one for
	// Spearman by default; we keep our existing Fisher CI behaviour).
	result.CI = pearsonFisherCI(rho, n, defaultConfidenceLevel)
	return result, nil
}

// hasTiesFloats reports whether any value in v repeats. O(n log n).
func hasTiesFloats(v []float64) bool {
	sorted := append([]float64(nil), v...)
	sort.Float64s(sorted)
	for i := 1; i < len(sorted); i++ {
		if sorted[i] == sorted[i-1] {
			return true
		}
	}
	return false
}

// spearmanPValueAS89 returns the two-sided p-value for Spearman's rho via R's
// pspearman/prho path. Mirrors the wrapper in stats:::cor.test:
//
//	q       = (n^3 - n) * (1 - rho) / 6      (Hotelling-Pabst S statistic)
//	if q > (n^3 - n)/6   →   p = pSpearmanRho(round(q),     n, lower=false)
//	else                 →   p = pSpearmanRho(round(q)+2, n, lower=true)
//	return min(2p, 1)
//
// The +2 in the lower-tail branch accounts for S taking only even integer
// values under the no-ties null distribution (so Pr[S ≤ q] = Pr[S < q+2]).
func spearmanPValueAS89(rho float64, n int) float64 {
	q := float64(n*n*n-n) * (1 - rho) / 6.0
	mu := float64(n*n*n-n) / 6.0
	var p float64
	if q > mu {
		p = pSpearmanRho(math.Round(q), n, false)
	} else {
		p = pSpearmanRho(math.Round(q)+2, n, true)
	}
	if 2*p > 1 {
		return 1
	}
	return 2 * p
}

// pSpearmanRho is a faithful Go port of R's prho() (src/library/stats/src/
// prho.c, AS-89 by Best & Roberts, Appl. Statist. 1975 24:377). Returns
// Pr[S ≥ s] when lowerTail is false, Pr[S < s] when lowerTail is true,
// under the H₀ rank-permutation distribution (no ties, n ≥ 2).
//
// Two regimes:
//   - n ≤ 9: exact enumeration of all n! rank permutations (matches R
//     character-for-character; the rotation-based permutation generator and
//     the (i+1 - l[i]) running squared-difference accumulation are direct
//     translations of the original Fortran).
//   - n ≥ 10: Edgeworth-series approximation with the twelve AS-89
//     coefficients c1..c12.
func pSpearmanRho(s float64, n int, lowerTail bool) float64 {
	pv := 0.0
	if !lowerTail {
		pv = 1.0
	}
	if n <= 1 {
		return math.NaN()
	}
	if s <= 0 {
		return pv
	}
	nF := float64(n)
	n3 := nF * (nF*nF - 1) / 3.0 // = (n³ − n)/3
	if s > n3 {
		return 1 - pv
	}

	const nSmall = 9
	if n <= nSmall {
		nfac := 1
		for i := 1; i <= n; i++ {
			nfac *= i
		}
		l := make([]int, n)
		for i := range l {
			l[i] = i + 1
		}

		ifr := 0
		if s == n3 {
			ifr = 1
		} else {
			sInt := int(s)
			for m := 0; m < nfac; m++ {
				ise := 0
				for i := 0; i < n; i++ {
					d := i + 1 - l[i]
					ise += d * d
				}
				if sInt <= ise {
					ifr++
				}
				// Rotation-based permutation enumeration (matches R's
				// algorithm AS 89). The carry-style termination mirrors the
				// original Fortran do-while.
				n1 := n
				for {
					mt := l[0]
					for i := 1; i < n1; i++ {
						l[i-1] = l[i]
					}
					n1--
					l[n1] = mt
					if mt != n1+1 || n1 <= 1 {
						break
					}
				}
			}
		}
		if lowerTail {
			return float64(nfac-ifr) / float64(nfac)
		}
		return float64(ifr) / float64(nfac)
	}

	// AS-89 Edgeworth coefficients
	const (
		c1  = 0.2274
		c2  = 0.2531
		c3  = 0.1745
		c4  = 0.0758
		c5  = 0.1033
		c6  = 0.3932
		c7  = 0.0879
		c8  = 0.0151
		c9  = 0.0072
		c10 = 0.0831
		c11 = 0.0131
		c12 = 4.6e-4
	)
	y := nF
	b := 1 / y
	x := (6.0*(s-1)*b/(y*y-1) - 1) * math.Sqrt(y-1)
	yy := x * x
	u := x * b * (c1 + b*(c2+c3*b) +
		yy*(-c4+b*(c5+c6*b)-
			yy*b*(c7+c8*b-yy*(c9-c10*b+yy*b*(c11-c12*yy)))))
	corr := u / math.Exp(yy/2.0)
	var pn float64
	if lowerTail {
		pn = norm.CDF(x)
		pv = -corr + pn
	} else {
		pn = 1 - norm.CDF(x)
		pv = corr + pn
	}
	if pv < 0 {
		pv = 0
	}
	if pv > 1 {
		pv = 1
	}
	return pv
}

func populateCorrelationInference(result *CorrelationResult, corr, n float64) {
	df := n - 2
	result.DF = &df
	if df <= 0 || math.IsNaN(corr) {
		result.PValue = math.NaN()
		result.CI = nanCIPtr()
		return
	}
	tStat := correlationToT(corr, n)
	result.PValue = tTwoTailedPValue(tStat, df)
	result.CI = pearsonFisherCI(corr, n, defaultConfidenceLevel)
}
