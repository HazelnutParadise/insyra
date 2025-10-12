package stats

import (
	"fmt"
	"sort"
	"strings"

	"github.com/HazelnutParadise/Go-Utils/conv"
	"github.com/HazelnutParadise/insyra"
	"gonum.org/v1/gonum/stat/distuv"
)

type ChiSquareTestResult struct {
	testResultBase

	// a DataTable representing the contingency table([2]float64{observed, expected})
	ContingencyTable *insyra.DataTable
}

func (r *ChiSquareTestResult) Show() {
	fmt.Printf("Chi-Square Test Statistic: %v\n", r.Statistic)
	fmt.Printf("Chi-Square Test P-Value: %v\n", r.PValue)
	fmt.Printf("Chi-Square Test Degrees of Freedom: %v\n", *r.DF)
	insyra.Show("Contingency Table([2]float64{observed, expected})", r.ContingencyTable)
}

// calculateChiSquare calculates the chi-square statistic and related results.
// Returns nil and an error message if any problems occur.
func calculateChiSquare(observed, expected []float64, df int) (*ChiSquareTestResult, string) {
	chiSquare := 0.0
	for i := range observed {
		if expected[i] == 0 {
			return nil, "Expected values must not be zero"
		}
		chiSquare += (observed[i] - expected[i]) * (observed[i] - expected[i]) / expected[i]
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

	// 創建 ContingencyTable 作為單列表格
	contingencyTable := insyra.NewDataTable()
	col := insyra.NewDataList()
	for i := range observed {
		col.Append([2]float64{observed[i], expected[i]})
	}
	contingencyTable.AppendCols(col)
	contingencyTable.SetColNameByNumber(0, "Observed_Expected")

	// 設置行名稱為類別索引
	for i := range observed {
		contingencyTable.SetRowNameByIndex(i, fmt.Sprintf("Category_%d", i))
	}

	result.ContingencyTable = contingencyTable
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
		s := strings.TrimSpace(conv.ToString(v))
		if s != "" && s != "#N/A" && s != "NA" && s != "N/A" {
			rowSet[s] = struct{}{}
		}
	}
	for _, v := range colVals {
		s := strings.TrimSpace(conv.ToString(v))
		if s != "" && s != "#N/A" && s != "NA" && s != "N/A" {
			colSet[s] = struct{}{}
		}
	}

	// 排序分類鍵值，確保順序一致
	rowKeys := make([]string, 0, len(rowSet))
	colKeys := make([]string, 0, len(colSet))
	for k := range rowSet {
		rowKeys = append(rowKeys, strings.TrimSpace(k))
	}
	for k := range colSet {
		colKeys = append(colKeys, strings.TrimSpace(k))
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
		rStr := strings.TrimSpace(conv.ToString(rowVals[i]))
		cStr := strings.TrimSpace(conv.ToString(colVals[i]))
		if rStr != "" && rStr != "#N/A" && rStr != "NA" && rStr != "N/A" &&
			cStr != "" && cStr != "#N/A" && cStr != "NA" && cStr != "N/A" {
			r := rStr
			c := cStr
			if _, exists := rowIndices[r]; exists {
				if _, exists := colIndices[c]; exists {
					observed[rowIndices[r]*cols+colIndices[c]]++
				}
			}
		}
	}

	// 計算期望值
	expected := make([]float64, rows*cols)
	rowSums := make([]float64, rows)
	colSums := make([]float64, cols)
	totalSum := 0.0

	for i := range rows {
		for j := range cols {
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

	// 創建 ContingencyTable
	contingencyTable := insyra.NewDataTable()
	for j := range cols {

		col := insyra.NewDataList()
		for i := range rows {
			obs := observed[i*cols+j]
			exp := expected[i*cols+j]
			col.Append([2]float64{obs, exp})
		}
		contingencyTable.AppendCols(col)
	}

	// 設置列名稱
	for j, colKey := range colKeys {
		contingencyTable.SetColNameByNumber(j, colKey)
	}

	// 設置行名稱
	for i, rowKey := range rowKeys {
		contingencyTable.SetRowNameByIndex(i, rowKey)
	}

	result.ContingencyTable = contingencyTable
	return result
}
