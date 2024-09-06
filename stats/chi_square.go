package stats

import (
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
func ChiSquareTest(input interface{}, p []float64, rescaleP bool) *ChiSquareTestResult {
	var observed []float64
	var expected []float64
	var df int

	switch v := input.(type) {
	case insyra.IDataList:
		observed = v.ToF64Slice()

		// 確保 p 的長度與 observed 相同，如果 p 為空則默認均等分佈
		if len(p) == 0 {
			p = make([]float64, len(observed))
			for i := range p {
				p[i] = 1.0 / float64(len(observed))
			}
		} else if len(p) != len(observed) {
			insyra.LogWarning("ChiSquareTest(): Length of p does not match observed data length.")
			return nil
		}

		// 若指定了 rescaleP，則重新縮放 p 使其和為1
		if rescaleP {
			sumP := 0.0
			for _, val := range p {
				sumP += val
			}
			for i := range p {
				p[i] /= sumP
			}
		}

		// 計算期望值
		totalObserved := 0.0
		for _, val := range observed {
			totalObserved += val
		}
		expected = make([]float64, len(observed))
		for i := range observed {
			expected[i] = totalObserved * p[i]
		}

		// 自由度為 len(observed) - 1
		df = len(observed) - 1

	case insyra.IDataTable:
		// 二維數據處理，和之前的二維檢定邏輯一致
		rows, cols := v.Size()
		observed = make([]float64, rows*cols)
		expected = make([]float64, rows*cols)
		rowSums := make([]float64, rows)
		colSums := make([]float64, cols)
		totalSum := 0.0

		// 計算行列總和
		for i := 0; i < rows; i++ {
			row := v.GetRow(i).ToF64Slice()
			for j, val := range row {
				observed[i*cols+j] = val
				rowSums[i] += val
				colSums[j] += val
				totalSum += val
			}
		}

		// 計算期望值
		for i := 0; i < rows; i++ {
			for j := 0; j < cols; j++ {
				expected[i*cols+j] = (rowSums[i] * colSums[j]) / totalSum
			}
		}

		// 二維自由度
		df = (rows - 1) * (cols - 1)

	default:
		insyra.LogWarning("ChiSquareTest(): Unsupported input type, returning nil.")
		return nil
	}

	// 計算卡方統計量
	chiSquare := 0.0
	for i := 0; i < len(observed); i++ {
		if expected[i] == 0 {
			insyra.LogWarning("ChiSquareTest(): Expected values must not be zero, returning nil.")
			return nil
		}
		chiSquare += math.Pow(observed[i]-expected[i], 2) / expected[i]
	}

	// 基於卡方分布計算 P 值
	chiDist := distuv.ChiSquared{K: float64(df)}
	pValue := 1 - chiDist.CDF(chiSquare)

	return &ChiSquareTestResult{
		ChiSquare: chiSquare,
		PValue:    pValue,
		Df:        df,
	}
}
