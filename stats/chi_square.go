package stats

import (
	"math"
	"sort"

	"github.com/HazelnutParadise/Go-Utils/conv"
	"github.com/HazelnutParadise/insyra"
	"gonum.org/v1/gonum/stat/distuv"
)

type ChiSquareTestResult struct {
	testResultBase
}

// calculateChiSquare calculates the chi-square statistic and related results.
// Returns nil and an error message if any problems occur.
func calculateChiSquare(observed, expected []float64, df int) (*ChiSquareTestResult, string) {
	chiSquare := 0.0
	for i := range observed {
		if expected[i] == 0 {
			return nil, "Expected values must not be zero"
		}
		chiSquare += math.Pow(observed[i]-expected[i], 2) / expected[i]
	}

	chiDist := distuv.ChiSquared{K: float64(df)}
	pValue := 1 - chiDist.CDF(chiSquare)

	float64DF := float64(df)
	return &ChiSquareTestResult{
		testResultBase: testResultBase{
			Statistic: chiSquare,
			PValue:    pValue,
			DF:        &float64DF,
		},
	}, ""
}

// ChiSquareGoodnessOfFit performs a one-dimensional chi-square goodness of fit test.
func ChiSquareGoodnessOfFit(input insyra.IDataList, p []float64, rescaleP bool) *ChiSquareTestResult {
	observed := input.ToF64Slice()
	var expected []float64
	var df int

	if len(p) == 0 {
		p = make([]float64, len(observed))
		for i := range p {
			p[i] = 1.0 / float64(len(observed))
		}
	} else if len(p) != len(observed) {
		insyra.LogWarning("stats", "ChiSquareGoodnessOfFit", "Length of p does not match observed data length")
		return nil
	}

	if rescaleP {
		sumP := 0.0
		for _, val := range p {
			sumP += val
		}
		for i := range p {
			p[i] /= sumP
		}
	}

	totalObserved := 0.0
	for _, val := range observed {
		totalObserved += val
	}

	expected = make([]float64, len(observed))
	for i := range observed {
		expected[i] = totalObserved * p[i]
	}

	df = len(observed) - 1
	result, errMsg := calculateChiSquare(observed, expected, df)
	if errMsg != "" {
		insyra.LogWarning("stats", "ChiSquareGoodnessOfFit", "%s", errMsg)
		return nil
	}
	return result
}

// ChiSquareIndependenceTest performs a chi-square test of independence.
func ChiSquareIndependenceTest(rowData, colData insyra.IDataList) *ChiSquareTestResult {
	rowVals := rowData.Data()
	colVals := colData.Data()

	if len(rowVals) == 0 || len(colVals) == 0 {
		insyra.LogWarning("stats", "ChiSquareIndependenceTest", "Input DataLists cannot be empty")
		return nil
	}
	if len(rowVals) != len(colVals) {
		insyra.LogWarning("stats", "ChiSquareIndependenceTest", "Both DataLists must have the same length")
		return nil
	}

	// 建立分類
	rowSet := make(map[string]struct{})
	colSet := make(map[string]struct{})
	for _, v := range rowVals {
		rowSet[conv.ToString(v)] = struct{}{}
	}
	for _, v := range colVals {
		colSet[conv.ToString(v)] = struct{}{}
	}

	// 排序分類鍵值，確保順序一致
	rowKeys := make([]string, 0, len(rowSet))
	colKeys := make([]string, 0, len(colSet))
	for k := range rowSet {
		rowKeys = append(rowKeys, k)
	}
	for k := range colSet {
		colKeys = append(colKeys, k)
	}
	sort.Strings(rowKeys)
	sort.Strings(colKeys)

	// 建立分類到索引的映射
	rowIndices := make(map[string]int)
	colIndices := make(map[string]int)
	for i, k := range rowKeys {
		rowIndices[k] = i
	}
	for i, k := range colKeys {
		colIndices[k] = i
	}

	rows := len(rowKeys)
	cols := len(colKeys)
	observed := make([]float64, rows*cols)

	// 填入觀察值
	for i := range rowVals {
		r := conv.ToString(rowVals[i])
		c := conv.ToString(colVals[i])
		observed[rowIndices[r]*cols+colIndices[c]]++
	}

	// 計算期望值
	expected := make([]float64, rows*cols)
	rowSums := make([]float64, rows)
	colSums := make([]float64, cols)
	totalSum := 0.0

	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			val := observed[i*cols+j]
			rowSums[i] += val
			colSums[j] += val
			totalSum += val
		}
	}

	for i := range rows {
		for j := range cols {
			expected[i*cols+j] = (rowSums[i] * colSums[j]) / totalSum
		}
	}

	df := (rows - 1) * (cols - 1)
	result, errMsg := calculateChiSquare(observed, expected, df)
	if errMsg != "" {
		insyra.LogWarning("stats", "ChiSquareIndependenceTest", "%s", errMsg)
		return nil
	}
	return result
}
