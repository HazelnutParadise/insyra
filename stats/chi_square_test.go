package stats_test

import (
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

func TestChiSquareGoodnessOfFit(t *testing.T) {
	// 測試資料：觀察值 = [10, 20, 30]，期望分配 = [0.2, 0.3, 0.5]
	observed := insyra.NewDataList([]float64{10, 20, 30})
	expectedProbs := []float64{0.2, 0.3, 0.5}
	result := stats.ChiSquareGoodnessOfFit(observed, expectedProbs, true)

	if result == nil {
		t.Fatal("ChiSquareGoodnessOfFit returned nil")
	}

	// Python scipy.stats.chisquare 結果（使用 f_exp = [12, 18, 30]）：
	expectedStat := 0.5556
	expectedP := 0.7575

	if !floatEquals(result.Statistic, expectedStat, 0.001) {
		t.Errorf("Chi-square statistic mismatch, got %f, expected %f", result.Statistic, expectedStat)
	}
	if !floatEquals(result.PValue, expectedP, 0.001) {
		t.Errorf("Chi-square p-value mismatch, got %f, expected %f", result.PValue, expectedP)
	}
}

func TestChiSquareIndependenceTest(t *testing.T) {
	// rowData: A, A, B, B, B, C
	// colData: X, Y, X, Y, Y, Y
	// 列聯表將是：
	//        X   Y
	// A      1   1
	// B      1   2
	// C      0   1

	rowData := insyra.NewDataList([]string{"A", "A", "B", "B", "B", "C"})
	colData := insyra.NewDataList([]string{"X", "Y", "X", "Y", "Y", "Y"})

	result := stats.ChiSquareIndependenceTest(rowData, colData)
	if result == nil {
		t.Fatal("ChiSquareIndependenceTest returned nil")
	}

	expectedStat := 0.75
	expectedP := 0.6873

	if !floatEquals(result.Statistic, expectedStat, 0.001) {
		t.Errorf("Chi-square statistic mismatch, got %f, expected %f", result.Statistic, expectedStat)
	}
	if !floatEquals(result.PValue, expectedP, 0.001) {
		t.Errorf("Chi-square p-value mismatch, got %f, expected %f", result.PValue, expectedP)
	}
}
