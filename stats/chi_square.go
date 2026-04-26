package stats

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/HazelnutParadise/Go-Utils/conv"
	"github.com/HazelnutParadise/insyra"
)

type ChiSquareTestResult struct {
	testResultBase

	// a DataTable representing the contingency table([2]float64{observed, expected})
	ContingencyTable *insyra.DataTable
}

func (r *ChiSquareTestResult) Show() {
	if r == nil {
		fmt.Println("Chi-Square Test failed: cannot show results")
		return
	}
	fmt.Printf("Chi-Square Test Statistic: %v\n", r.Statistic)
	fmt.Printf("Chi-Square Test P-Value: %v\n", r.PValue)
	fmt.Printf("Chi-Square Test Degrees of Freedom: %v\n", *r.DF)
	insyra.Show("Contingency Table([2]float64{observed, expected})", r.ContingencyTable)
}

// calculateChiSquare calculates the chi-square statistic and related results.
// Returns nil and an error message if any problems occur.
func calculateChiSquare(observed, expected []float64, df int) (*ChiSquareTestResult, error) {
	if df <= 0 {
		return nil, errors.New("degrees of freedom must be positive")
	}
	chiSquare := 0.0
	for i := range observed {
		if expected[i] <= 0 {
			return nil, errors.New("expected values must be greater than zero")
		}
		chiSquare += (observed[i] - expected[i]) * (observed[i] - expected[i]) / expected[i]
	}

	pValue := chiSquaredPValue(chiSquare, float64(df))

	float64DF := float64(df)
	return &ChiSquareTestResult{
		testResultBase: testResultBase{
			Statistic: chiSquare,
			PValue:    pValue,
			DF:        &float64DF,
		},
	}, nil
}

// ChiSquareGoodnessOfFit performs a one-dimensional chi-square goodness of fit test.
//
// input: A DataList containing categorical data (e.g., ["A", "B", "A"]).
// p: Expected probabilities (e.g., []float64{0.5, 0.5}). If nil, assumes uniform distribution.
// rescaleP: Whether to rescale p to sum to 1.
func ChiSquareGoodnessOfFit(input insyra.IDataList, p []float64, rescaleP bool) (*ChiSquareTestResult, error) {
	// 計算類別頻率
	data := input.Data()
	if len(data) == 0 {
		return nil, errors.New("input DataList cannot be empty")
	}
	categoryFreq := make(map[string]float64)
	for _, v := range data {
		s := strings.TrimSpace(conv.ToString(v))
		categoryFreq[s]++
	}

	// 將頻率轉為 observed 切片
	categoryKeys := make([]string, 0, len(categoryFreq))
	for k := range categoryFreq {
		categoryKeys = append(categoryKeys, k)
	}
	sort.Strings(categoryKeys) // 確保順序一致
	observed := make([]float64, 0, len(categoryFreq))
	for _, k := range categoryKeys {
		observed = append(observed, categoryFreq[k])
	}

	var expected []float64
	var df int

	if len(p) == 0 {
		p = make([]float64, len(observed))
		for i := range p {
			p[i] = 1.0 / float64(len(observed))
		}
	} else if len(p) != len(observed) {
		return nil, errors.New("length of p does not match number of categories")
	}

	sumP := 0.0
	for _, val := range p {
		if val < 0 || math.IsNaN(val) || math.IsInf(val, 0) {
			return nil, errors.New("probabilities must be finite and non-negative")
		}
		sumP += val
	}
	if sumP <= 0 {
		return nil, errors.New("probabilities must sum to a positive value")
	}
	if rescaleP {
		for i := range p {
			p[i] /= sumP
		}
	} else if math.Abs(sumP-1) > 1e-12 {
		return nil, errors.New("probabilities must sum to 1 unless rescaleP is true")
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
	result, err := calculateChiSquare(observed, expected, df)
	if err != nil {
		return nil, err
	}

	// 創建 ContingencyTable 作為單列表格
	contingencyTable := insyra.NewDataTable()
	col := insyra.NewDataList()
	for i := range observed {
		col.Append([2]float64{observed[i], expected[i]})
	}
	contingencyTable.AppendCols(col)
	contingencyTable.SetColNameByNumber(0, "Observed_Expected")

	// 設置行名稱為類別
	for i, key := range categoryKeys {
		contingencyTable.SetRowNameByIndex(i, key)
	}

	result.ContingencyTable = contingencyTable.SetName("Contingency_Table")
	return result, nil
}

// ChiSquareIndependenceTest performs a chi-square test of independence.
func ChiSquareIndependenceTest(rowData, colData insyra.IDataList) (*ChiSquareTestResult, error) {
	rowVals := rowData.Data()
	colVals := colData.Data()

	if len(rowVals) == 0 || len(colVals) == 0 {
		return nil, errors.New("input DataLists cannot be empty")
	}
	if len(rowVals) != len(colVals) {
		return nil, errors.New("both DataLists must have the same length")
	}

	// 建立分類
	rowSet := make(map[string]struct{})
	colSet := make(map[string]struct{})
	for _, v := range rowVals {
		s := strings.TrimSpace(conv.ToString(v))
		rowSet[s] = struct{}{}
	}
	for _, v := range colVals {
		s := strings.TrimSpace(conv.ToString(v))
		colSet[s] = struct{}{}
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
	if rows < 2 || cols < 2 {
		return nil, errors.New("chi-square independence test requires at least two row and column categories")
	}
	observed := make([]float64, rows*cols)

	// 填入觀察值
	for i := range rowVals {
		rStr := strings.TrimSpace(conv.ToString(rowVals[i]))
		cStr := strings.TrimSpace(conv.ToString(colVals[i]))
		r := rStr
		c := cStr
		if _, exists := rowIndices[r]; exists {
			if _, exists := colIndices[c]; exists {
				observed[rowIndices[r]*cols+colIndices[c]]++
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
	result, err := calculateChiSquare(observed, expected, df)
	if err != nil {
		return nil, err
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

	result.ContingencyTable = contingencyTable.SetName("Contingency_Table")
	return result, nil
}
