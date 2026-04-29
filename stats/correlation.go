package stats

import (
	"errors"
	"fmt"
	"math"
	"sort"

	"github.com/HazelnutParadise/insyra"
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
func CorrelationMatrix(dataTable insyra.IDataTable, method CorrelationMethod) (corrMatrix *insyra.DataTable, pMatrix *insyra.DataTable, err error) {
	dt := insyra.NewDataTable()
	pdt := insyra.NewDataTable()
	var pairErr error
	dataTable.AtomicDo(func(table *insyra.DataTable) {
		_, n := table.Size()
		if n < 2 {
			err = errors.New("need at least two columns for correlation")
			return
		}

		matrix := make([][]float64, n)
		pmatrix := make([][]float64, n)
		for i := range matrix {
			matrix[i] = make([]float64, n)
			pmatrix[i] = make([]float64, n)
		}

		for i := range n {
			for j := i; j < n; j++ {
				if i == j {
					matrix[i][j] = 1.0
					pmatrix[i][j] = 0.0
					continue
				}

				corrResult, corrErr := Correlation(table.GetColByNumber(i), table.GetColByNumber(j), method)
				if corrErr == nil && corrResult != nil && !math.IsNaN(corrResult.Statistic) {
					matrix[i][j] = corrResult.Statistic
					matrix[j][i] = corrResult.Statistic
					pmatrix[i][j] = corrResult.PValue
					pmatrix[j][i] = corrResult.PValue
				} else {
					if pairErr == nil {
						if corrErr != nil {
							pairErr = fmt.Errorf("unable to calculate correlation for columns %d and %d: %w", i, j, corrErr)
						} else {
							pairErr = fmt.Errorf("unable to calculate correlation for columns %d and %d", i, j)
						}
					}
					matrix[i][j] = math.NaN()
					matrix[j][i] = math.NaN()
					pmatrix[i][j] = math.NaN()
					pmatrix[j][i] = math.NaN()
				}
			}
		}

		dtColNames := insyra.NewDataList()
		for i := range matrix {
			rowName := table.GetColByNumber(i).GetName()
			row := insyra.NewDataList().SetName(rowName)
			dtColNames.Append(rowName)
			for j := range matrix[i] {
				row.Append(matrix[i][j])
			}
			dt.AppendRowsFromDataList(row)
		}
		dt.AppendRowsFromDataList(dtColNames)
		dt.SetRowToColNames(-1)

		pdtColNames := insyra.NewDataList()
		for i := range pmatrix {
			rowName := table.GetColByNumber(i).GetName()
			row := insyra.NewDataList().SetName(rowName)
			pdtColNames.Append(rowName)
			for j := range pmatrix[i] {
				row.Append(pmatrix[i][j])
			}
			pdt.AppendRowsFromDataList(row)
		}
		pdt.AppendRowsFromDataList(pdtColNames)
		pdt.SetRowToColNames(-1)
	})
	if err != nil {
		return nil, nil, err
	}

	return dt, pdt, pairErr
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
// Algorithm: O(n²) pair iteration plus an O(n log n) sort per axis to extract
// tie group sizes for the tie-corrected asymptotic variance. Knight's
// O(n log n) merge-sort variant exists but adds significant complexity for
// little gain at the n's we exercise (the package's other tests cap around
// n ≈ 250); the naive form is correct by inspection.
//
// Returns:
//   tau  — Kendall's τ-b in [-1, 1] (NaN if either axis is all-tied)
//   sval — concordant minus discordant pair count (the "S" statistic)
//   varS — variance of S under H₀ with tie correction (Conover 1999, eq. 5.16)
func kendallTauBStats(x, y []float64) (tau, sval, varS float64) {
	n := len(x)
	if n < 2 {
		return math.NaN(), 0, math.NaN()
	}
	var nC, nD float64
	for i := range n {
		for j := i + 1; j < n; j++ {
			dx := x[i] - x[j]
			dy := y[i] - y[j]
			if dx == 0 || dy == 0 {
				continue // tied in at least one axis: contributes neither to nC nor nD
			}
			if (dx > 0) == (dy > 0) {
				nC++
			} else {
				nD++
			}
		}
	}

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
		// Exact two-sided permutation p-value: hold x fixed, permute y, count
		// how many permutations give |τ_b| at least as large as the observed
		// one. Tau-b (not gonum's tau-a) is used inside the loop too.
		perms := generatePermutations(y)
		extreme := 0
		obs := math.Abs(tau)
		for _, perm := range perms {
			altTau, _, _ := kendallTauBStats(x, perm)
			if math.Abs(altTau) >= obs {
				extreme++
			}
		}
		result.PValue = float64(extreme) / float64(len(perms))
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

func generatePermutations(arr []float64) [][]float64 {
	sort.Float64s(arr)
	res := [][]float64{}
	n := len(arr)
	used := make([]bool, n)
	perm := make([]float64, n)

	var dfs func(int)
	dfs = func(depth int) {
		if depth == n {
			copyPerm := make([]float64, n)
			copy(copyPerm, perm)
			res = append(res, copyPerm)
			return
		}
		for i := range n {
			if used[i] {
				continue
			}
			used[i] = true
			perm[depth] = arr[i]
			dfs(depth + 1)
			used[i] = false
		}
	}
	dfs(0)
	return res
}

func spearmanCorrelationWithStats(dlX, dlY insyra.IDataList) (CorrelationResult, error) {
	result := CorrelationResult{}
	var rankX, rankY insyra.IDataList
	var n float64
	dlX.AtomicDo(func(dlx *insyra.DataList) {
		dlY.AtomicDo(func(dly *insyra.DataList) {
			rankX = dlx.Rank()
			rankY = dly.Rank()
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

	populateCorrelationInference(&result, rho, n)

	return result, nil
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
