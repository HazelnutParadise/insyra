package stats

import (
	"fmt"

	"github.com/HazelnutParadise/insyra"
	"gonum.org/v1/gonum/mat"
	"gonum.org/v1/gonum/stat"
)

// PCAResult 用來儲存 PCA 分析的結果
type PCAResult struct {
	Components        insyra.IDataTable // 主成分存為 DataTable
	Eigenvalues       []float64         // 對應的特徵值
	ExplainedVariance []float64         // 每個主成分解釋的變異百分比
}

// PCADataTable 執行主成分分析，接受 IDataTable 形式的資料
// dataTable 是輸入的數據表，nComponents 是主成分數量
func PCADataTable(dataTable insyra.IDataTable, nComponents int) *PCAResult {
	rowNum, colNum := dataTable.Size()

	// 檢查 nComponents 是否合理
	if nComponents > colNum {
		nComponents = colNum // 返回全部主成分
	}

	// 將 DataTable 轉換為矩陣，將每行視為一個樣本
	data := mat.NewDense(rowNum, colNum, nil)
	for i := 0; i < rowNum; i++ {
		row := dataTable.GetRow(i)
		for j := 0; j < colNum; j++ {
			value, ok := row.Get(j).(float64)
			if ok {
				data.Set(i, j, value)
			}
		}
	}

	// 進行數據標準化（Z-Score 標準化）
	for j := 0; j < colNum; j++ {
		col := mat.Col(nil, j, data)
		mean, std := stat.MeanStdDev(col, nil)
		if std == 0 { // 防止標準差為 0
			std = 1
		}
		for i := 0; i < rowNum; i++ {
			data.Set(i, j, (data.At(i, j)-mean)/std)
		}
	}

	// 計算數據的協方差矩陣
	covMatrix := mat.NewSymDense(colNum, nil)
	stat.CovarianceMatrix(covMatrix, data, nil)

	// 特徵值分解協方差矩陣
	var eig mat.EigenSym
	if !eig.Factorize(covMatrix, true) {
		fmt.Println("Eigenvalue decomposition failed")
		return nil
	}

	// 取得特徵值和特徵向量
	eigenvalues := eig.Values(nil)
	var eigenvectors mat.Dense
	eig.VectorsTo(&eigenvectors)

	// 確認特徵向量矩陣的尺寸
	fmt.Printf("Eigenvectors matrix size: %dx%d\n", eigenvectors.RawMatrix().Rows, eigenvectors.RawMatrix().Cols)

	// 確保行列次序正確
	if nComponents > eigenvectors.RawMatrix().Cols {
		nComponents = eigenvectors.RawMatrix().Cols
	}

	// 生成 DataTable 並存儲主成分
	componentTable := insyra.NewDataTable()

	// 將主成分轉換為 DataTable，這次按列填入主成分
	for j := 0; j < nComponents; j++ {
		column := insyra.NewDataList()
		for i := 0; i < eigenvectors.RawMatrix().Rows; i++ { // 按照行來取主成分數據
			column.Append(eigenvectors.At(i, j))
		}
		componentTable.AppendColumns(column.SetName(fmt.Sprintf("PC%d", j+1)))
	}

	// 計算解釋變異百分比，使用協方差矩陣的特徵值
	totalVariance := 0.0
	for _, v := range eigenvalues {
		totalVariance += v // 使用特徵值來計算總變異
	}
	explainedVariance := make([]float64, nComponents)
	for i := 0; i < nComponents; i++ {
		explainedVariance[i] = (eigenvalues[i] / totalVariance) * 100
	}

	return &PCAResult{
		Components:        componentTable,
		Eigenvalues:       eigenvalues[:nComponents],
		ExplainedVariance: explainedVariance,
	}
}
