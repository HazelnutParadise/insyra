package stats

import (
	"errors"
	"fmt"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/internal/algorithms"
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
	dataTable.AtomicDo(func(dt *insyra.DataTable) {
		rowNum, colNum = dt.Size()

		numComponents = colNum
		if len(nComponents) == 1 {
			if nComponents[0] > 0 && nComponents[0] <= colNum {
				numComponents = nComponents[0]
			}
		}

		data = mat.NewDense(rowNum, colNum, nil)
		for i := range rowNum {
			row := dt.GetRow(i)
			for j := range colNum {
				value, ok := insyra.ToFloat64Safe(row.Get(j))
				if !ok {
					data = nil
					return
				}
				data.Set(i, j, value)
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

	for j := range colNum {
		col := mat.Col(nil, j, data)
		mean, std := stat.MeanStdDev(col, nil)
		if std == 0 {
			std = 1
		}
		for i := range rowNum {
			data.Set(i, j, (data.At(i, j)-mean)/std)
		}
	}

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
