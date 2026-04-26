package stats

import (
	"errors"
	"fmt"
	"math"
	"sort"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats/internal/linalg"
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
	var meanX, meanY float64
	var lenX int
	var lenY int
	var dataX, dataY []float64
	dlX.AtomicDo(func(dlx *insyra.DataList) {
		dlY.AtomicDo(func(dly *insyra.DataList) {
			meanX = dlx.Mean()
			meanY = dly.Mean()
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

	var sum float64
	for i := range lenX {
		x := dataX[i]
		y := dataY[i]
		sum += (x - meanX) * (y - meanY)
	}
	return sum / float64(lenX-1), nil
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

	det := linalg.DeterminantGauss(corrMatrix)
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

func kendallCorrelationWithStats(dlX, dlY insyra.IDataList) CorrelationResult {
	result := CorrelationResult{}
	x := dlX.ToF64Slice()
	y := dlY.ToF64Slice()
	tau := stat.Kendall(x, y, nil)
	result.Statistic = tau

	n := len(x)
	if n <= 7 {
		perms := generatePermutations(y)
		extreme := 0
		obs := math.Abs(tau)
		for _, perm := range perms {
			altTau := stat.Kendall(x, perm, nil)
			if math.Abs(altTau) >= obs {
				extreme++
			}
		}
		result.PValue = float64(extreme) / float64(len(perms))
	} else {
		z := 3 * tau * math.Sqrt(float64(n*(n-1))) / math.Sqrt(2*float64(2*n+5))
		result.PValue = 2 * (1 - zCDF(math.Abs(z)))
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
