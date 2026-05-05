package stats

import (
	"errors"
	"fmt"
	"math"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/internal/algorithms"
	"github.com/HazelnutParadise/insyra/stats/internal/parutil"
	"gonum.org/v1/gonum/mat"
	"gonum.org/v1/gonum/stat"
)

// PCAResult contains the results of a Principal Component Analysis.
type PCAResult struct {
	Components        insyra.IDataTable // component loadings matrix
	Eigenvalues       []float64
	ExplainedVariance []float64
}

// PCA calculates the Principal Component Analysis of a DataTable.
func PCA(dataTable insyra.IDataTable, nComponents ...int) (*PCAResult, error) {
	var rowNum, colNum, numComponents int
	var data *mat.Dense
	// Bulk-load per column via ToF64Slice. The previous nested-loop form
	// (`dt.GetRow(i).Get(j)` for every cell) profiled to 67% runtime.Stack —
	// each Get goes through the DataList actor, whose getGID call walks the
	// goroutine stack on every invocation. For an 800×12 table that is
	// 800 row-actor entries plus 9600 cell-actor entries, all paying the
	// stack-trace cost. Switching to one AtomicDo per column drops that
	// to colNum entries (= 12 here, ≈800× fewer actor handshakes).
	dataTable.AtomicDo(func(dt *insyra.DataTable) {
		rowNum, colNum = dt.Size()

		numComponents = colNum
		if len(nComponents) == 1 {
			numComponents = nComponents[0]
		}

		data = mat.NewDense(rowNum, colNum, nil)
		for j := range colNum {
			col := dt.GetColByNumber(j)
			col.AtomicDo(func(dl *insyra.DataList) {
				// One AtomicDo entry → one getGID stack walk. Inside we
				// pull the raw []any once (Data() re-enters the same
				// actor inline, no extra getGID) and iterate with
				// ToFloat64Safe so non-numeric cells surface as errors
				// (ToF64Slice would silently coerce them to 0).
				raw := dl.Data()
				if len(raw) != rowNum {
					data = nil
					return
				}
				for i, v := range raw {
					f, ok := insyra.ToFloat64Safe(v)
					if !ok {
						data = nil
						return
					}
					data.Set(i, j, f)
				}
			})
			if data == nil {
				return
			}
		}
	})
	if len(nComponents) > 1 {
		return nil, errors.New("nComponents accepts at most one value")
	}
	if data == nil {
		return nil, errors.New("input contains non-numeric values")
	}
	if rowNum < 2 || colNum < 1 {
		return nil, errors.New("insufficient data shape for PCA")
	}
	if numComponents <= 0 || numComponents > colNum {
		return nil, fmt.Errorf("nComponents must be between 1 and %d", colNum)
	}

	// Standardisation in two parallel phases:
	//   (1) per-column mean & sample std — each column j is independent.
	//       Uses the same two-pass formula as gonum/stat.MeanStdDev (Σx then
	//       Σ(x-mean)² with n-1 divisor) so the result is bit-identical to
	//       the previous mat.Col + MeanStdDev path, while skipping the
	//       per-column []float64 allocation.
	//   (2) row-parallel rewrite using the precomputed (mean, std). Each
	//       worker writes only to rows it owns, so no cache-line ping-pong.
	means := make([]float64, colNum)
	stds := make([]float64, colNum)
	stdErrs := make([]error, colNum)
	// Per-column work is 2·rowNum reads + 1·rowNum writes plus an Sqrt.
	// Strided column access through mat.Dense costs ~3ns per element on
	// modern x86, so per-column ≈ rowNum·9ns. Parallel pays off when
	// rowNum·colNum (total ops) ≳ 5000 — well below typical PCA sizes.
	parutil.Run(colNum, rowNum*colNum >= 5000, func(j int) {
		var sum float64
		for i := range rowNum {
			sum += data.At(i, j)
		}
		mean := sum / float64(rowNum)
		var ss float64
		for i := range rowNum {
			d := data.At(i, j) - mean
			ss += d * d
		}
		std := math.Sqrt(ss / float64(rowNum-1))
		if std == 0 {
			stdErrs[j] = fmt.Errorf("PCA undefined for zero-variance column %d", j)
			return
		}
		means[j] = mean
		stds[j] = std
	})
	for _, e := range stdErrs {
		if e != nil {
			return nil, e
		}
	}
	// Row-parallel rewrite: each worker writes contiguous rows, no false
	// sharing. Same total-ops gate as the column phase.
	parutil.Run(rowNum, rowNum*colNum >= 5000, func(i int) {
		for j := range colNum {
			data.Set(i, j, (data.At(i, j)-means[j])/stds[j])
		}
	})

	covMatrix := mat.NewSymDense(colNum, nil)
	stat.CovarianceMatrix(covMatrix, data, nil)

	var eig mat.EigenSym
	if !eig.Factorize(covMatrix, true) {
		return nil, errors.New("eigenvalue decomposition failed")
	}

	eigenvalues := eig.Values(nil)
	var eigenvectors mat.Dense
	eig.VectorsTo(&eigenvectors)

	indices := make([]int, len(eigenvalues))
	for i := range indices {
		indices[i] = i
	}
	algorithms.ParallelSortStableFunc(indices, func(a, b int) int {
		if eigenvalues[a] > eigenvalues[b] {
			return -1
		} else if eigenvalues[a] < eigenvalues[b] {
			return 1
		} else {
			return 0
		}
	})

	componentTable := insyra.NewDataTable()
	for compIndex := range numComponents {
		column := insyra.NewDataList()
		sign := 1.0
		if eigenvectors.At(0, indices[compIndex]) < 0 {
			sign = -1.0
		}
		for i := range eigenvectors.RawMatrix().Rows {
			column.Append(sign * eigenvectors.At(i, indices[compIndex]))
		}
		componentTable.AppendCols(column.SetName(fmt.Sprintf("PC%d", compIndex+1)))
	}

	totalVariance := 0.0
	for _, v := range eigenvalues {
		totalVariance += v
	}
	explainedVariance := make([]float64, numComponents)
	for i := range numComponents {
		explainedVariance[i] = (eigenvalues[indices[i]] / totalVariance) * 100
	}

	sortedEigenvalues := make([]float64, numComponents)
	for i := range numComponents {
		sortedEigenvalues[i] = eigenvalues[indices[i]]
	}

	return &PCAResult{
		Components:        componentTable,
		Eigenvalues:       sortedEigenvalues,
		ExplainedVariance: explainedVariance,
	}, nil
}
