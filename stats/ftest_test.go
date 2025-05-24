package stats_test

import (
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/isr"
	"github.com/HazelnutParadise/insyra/stats"
)

func TestFTestForVarianceEquality(t *testing.T) {
	data1 := insyra.NewDataList([]float64{10, 12, 9, 11})
	data2 := insyra.NewDataList([]float64{20, 19, 21, 22})

	result := stats.FTestForVarianceEquality(data1, data2)
	if result == nil {
		t.Fatal("FTestForVarianceEquality returned nil")
	}

	expectedF := 1.0
	expectedP := 1.0

	if !floatEquals(result.Statistic, expectedF, 1e-6) {
		t.Errorf("F-value mismatch: got %.6f, want %.6f", result.Statistic, expectedF)
	}
	if !floatEquals(result.PValue, expectedP, 1e-6) {
		t.Errorf("P-value mismatch: got %.6f, want %.6f", result.PValue, expectedP)
	}
}

func TestLeveneAndBartlett(t *testing.T) {
	// 測試資料（Python 驗證過）
	group1 := insyra.NewDataList([]float64{10, 12, 9, 11})
	group2 := insyra.NewDataList([]float64{20, 19, 21, 22})
	group3 := insyra.NewDataList([]float64{30, 29, 28, 32})

	groups := isr.DLs{group1, group2, group3}

	// Levene's Test
	leveneResult := stats.LeveneTest(groups)
	if leveneResult == nil {
		t.Fatal("LeveneTest returned nil")
	}
	expectedLeveneStat := 0.1579
	expectedLeveneP := 0.8562
	if !floatEquals(leveneResult.Statistic, expectedLeveneStat, 0.01) {
		t.Errorf("Levene Statistic mismatch: got %.4f, want %.4f", leveneResult.Statistic, expectedLeveneStat)
	}
	if !floatEquals(leveneResult.PValue, expectedLeveneP, 0.01) {
		t.Errorf("Levene P-value mismatch: got %.4f, want %.4f", leveneResult.PValue, expectedLeveneP)
	}

	// Bartlett's Test
	bartlettResult := stats.BartlettTest(groups)
	if bartlettResult == nil {
		t.Fatal("BartlettTest returned nil")
	}
	expectedBartlettStat := 0.2869
	expectedBartlettP := 0.8663
	if !floatEquals(bartlettResult.Statistic, expectedBartlettStat, 0.01) {
		t.Errorf("Bartlett Statistic mismatch: got %.4f, want %.4f", bartlettResult.Statistic, expectedBartlettStat)
	}
	if !floatEquals(bartlettResult.PValue, expectedBartlettP, 0.01) {
		t.Errorf("Bartlett P-value mismatch: got %.4f, want %.4f", bartlettResult.PValue, expectedBartlettP)
	}
}

func TestFTestForRegression(t *testing.T) {
	ssr := 500.0
	sse := 200.0
	df1 := 3
	df2 := 16

	result := stats.FTestForRegression(ssr, sse, df1, df2)
	if result == nil {
		t.Fatal("FTestForRegression returned nil")
	}

	expectedF := 13.3333
	expectedP := 0.000128

	if !floatEquals(result.Statistic, expectedF, 1e-3) {
		t.Errorf("F-value mismatch: got %.4f, want %.4f", result.Statistic, expectedF)
	}
	if !floatEquals(result.PValue, expectedP, 1e-5) {
		t.Errorf("P-value mismatch: got %.6f, want %.6f", result.PValue, expectedP)
	}
}

func TestFTestForNestedModels(t *testing.T) {
	rssReduced := 300.0
	rssFull := 200.0
	dfReduced := 18
	dfFull := 16

	result := stats.FTestForNestedModels(rssReduced, rssFull, dfReduced, dfFull)
	if result == nil {
		t.Fatal("FTestForNestedModels returned nil")
	}

	expectedF := 4.0
	expectedP := 0.0390

	if !floatEquals(result.Statistic, expectedF, 1e-3) {
		t.Errorf("F-value mismatch: got %.4f, want %.4f", result.Statistic, expectedF)
	}
	if !floatEquals(result.PValue, expectedP, 1e-4) {
		t.Errorf("P-value mismatch: got %.5f, want %.5f", result.PValue, expectedP)
	}
}
