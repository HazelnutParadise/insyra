package stats_test

import (
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

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
