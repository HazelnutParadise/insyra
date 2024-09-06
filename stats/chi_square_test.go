package stats

import (
	"errors"
	"math"

	"github.com/HazelnutParadise/insyra"
	"gonum.org/v1/gonum/stat/distuv"
)

type ChiSquareTestResult struct {
	ChiSquare float64
	PValue    float64
	Df        int
}

// ChiSquareTest supports both DataList (1D) and DataTable (2D) for chi-square tests.
func ChiSquareTest(input interface{}) (*ChiSquareTestResult, error) {
	var observed [][]float64

	// 檢查輸入類型
	switch v := input.(type) {
	case insyra.IDataList:
		// 如果是 DataList，轉換成 2x1 表格進行處理
		observed = [][]float64{
			v.ToF64Slice(), // 直接使用 DataList 的轉換函數
		}
	case insyra.IDataTable:
		rowNum, _ := v.Size()
		// 如果是 DataTable，提取 2D 數據
		for i := 0; i < rowNum; i++ {
			observed = append(observed, v.GetRow(i).ToF64Slice()) // 將每一行轉換為 float64 切片
		}
	default:
		return nil, errors.New("unsupported input type for ChiSquareTest")
	}

	rows := len(observed)
	if rows == 0 {
		return nil, errors.New("empty data input")
	}
	cols := len(observed[0])

	// 計算行和列總和
	rowSums := make([]float64, rows)
	colSums := make([]float64, cols)
	totalSum := 0.0
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			rowSums[i] += observed[i][j]
			colSums[j] += observed[i][j]
			totalSum += observed[i][j]
		}
	}

	// 計算期望值表格
	expected := make([][]float64, rows)
	for i := range expected {
		expected[i] = make([]float64, cols)
		for j := 0; j < cols; j++ {
			expected[i][j] = (rowSums[i] * colSums[j]) / totalSum
		}
	}

	// 計算卡方統計量
	chiSquare := 0.0
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			if expected[i][j] == 0 {
				return nil, errors.New("expected values must not be zero")
			}
			chiSquare += math.Pow(observed[i][j]-expected[i][j], 2) / expected[i][j]
		}
	}

	// 自由度 = (行數 - 1) * (列數 - 1)
	df := (rows - 1) * (cols - 1)

	// 基於卡方分布計算 P 值
	chiDist := distuv.ChiSquared{K: float64(df)}
	pValue := 1 - chiDist.CDF(chiSquare)

	return &ChiSquareTestResult{
		ChiSquare: chiSquare,
		PValue:    pValue,
		Df:        df,
	}, nil
}
