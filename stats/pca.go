package stats

import (
	"fmt"
	"sort"

	"github.com/HazelnutParadise/insyra"
	"gonum.org/v1/gonum/mat"
	"gonum.org/v1/gonum/stat"
)

// PCAResult contains the results of a Principal Component Analysis.
type PCAResult struct {
	Components        insyra.IDataTable // 主成分存為 DataTable
	Eigenvalues       []float64         // 對應的特徵值
	ExplainedVariance []float64         // 每個主成分解釋的變異百分比
}

// PCA calculates the Principal Component Analysis of a DataTable.
// The function returns a PCAResult struct containing the principal components,
// eigenvalues, and explained variance.
// The number of components to extract can be specified using the nComponents parameter.
// If nComponents is not specified or exceeds the number of columns, all components will be extracted.
func PCA(dataTable insyra.IDataTable, nComponents ...int) *PCAResult {
	rowNum, colNum := dataTable.Size()

	// 如果 nComponents 沒有指定，或者超過列數，則提取所有主成分
	numComponents := colNum
	if len(nComponents) == 1 && nComponents[0] > 0 && nComponents[0] <= colNum {
		numComponents = nComponents[0]
	} else if len(nComponents) > 1 {
		insyra.LogWarning("stats.PCA: Invalid number of components, extracting all components.")
	}

	// 將 DataTable 轉換為矩陣，將每行視為一個樣本
	data := mat.NewDense(rowNum, colNum, nil)
	for i := range rowNum {
		row := dataTable.GetRow(i)
		for j := range colNum {
			value, ok := row.Get(j).(float64)
			if ok {
				data.Set(i, j, value)
			}
		}
	}

	// 進行數據標準化（Z-Score 標準化）
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

	// 計算數據的協方差矩陣
	covMatrix := mat.NewSymDense(colNum, nil)
	stat.CovarianceMatrix(covMatrix, data, nil)

	// 特徵值分解協方差矩陣
	var eig mat.EigenSym
	if !eig.Factorize(covMatrix, true) {
		insyra.LogWarning("stats.PCA: Eigenvalue decomposition failed")
		return nil
	}

	// 取得特徵值和特徵向量
	eigenvalues := eig.Values(nil)
	var eigenvectors mat.Dense
	eig.VectorsTo(&eigenvectors)

	// 將特徵值與特徵向量根據特徵值大小進行排序
	indices := make([]int, len(eigenvalues))
	for i := range indices {
		indices[i] = i
	}
	sort.Slice(indices, func(i, j int) bool {
		return eigenvalues[indices[i]] > eigenvalues[indices[j]] // 根據特徵值由大到小排序
	})

	// 生成 DataTable 並存儲主成分
	componentTable := insyra.NewDataTable()

	// 手動調整每個主成分的正負號，使其與 R 的結果一致
	for compIndex := range numComponents {
		column := insyra.NewDataList()
		sign := 1.0 // 默認正負號

		// 根據某個基準來確定特徵向量的正負號，這裡你可以自行設置基準，例如 R 的結果
		if eigenvectors.At(0, indices[compIndex]) < 0 {
			sign = -1.0 // 若第一個值為負數，則反轉整列的正負號
		}

		for i := range eigenvectors.RawMatrix().Rows {
			column.Append(sign * eigenvectors.At(i, indices[compIndex])) // 根據排序後的索引提取數據並調整正負號
		}
		componentTable.AppendCols(column.SetName(fmt.Sprintf("PC%d", compIndex+1)))
	}

	// 計算解釋變異百分比，使用協方差矩陣的特徵值
	totalVariance := 0.0
	for _, v := range eigenvalues {
		totalVariance += v
	}
	explainedVariance := make([]float64, numComponents)
	for i := range numComponents {
		explainedVariance[i] = (eigenvalues[indices[i]] / totalVariance) * 100
	}

	// 按排序後的特徵值順序返回結果
	sortedEigenvalues := make([]float64, numComponents)
	for i := range numComponents {
		sortedEigenvalues[i] = eigenvalues[indices[i]]
	}

	return &PCAResult{
		Components:        componentTable,
		Eigenvalues:       sortedEigenvalues,
		ExplainedVariance: explainedVariance,
	}
}
