package stats_test

import (
	"fmt"
	"math"
	"testing"

	"github.com/HazelnutParadise/insyra/stats"
)

func TestCheckPreRotationLoadings(t *testing.T) {
	dt := readFactorAnalysisSampleCSV(t)

	// Test with NO rotation first
	resultNoRot := stats.FactorAnalysis(dt, stats.FactorAnalysisOptions{
		Preprocess: stats.FactorPreprocessOptions{Standardize: true},
		Count:      stats.FactorCountSpec{Method: stats.FactorCountFixed, FixedK: 3},
		Extraction: stats.FactorExtractionPAF,
		Rotation:   stats.FactorRotationOptions{Method: stats.FactorRotationNone},
		MinErr:     1e-9,
		MaxIter:    50,
	})

	if resultNoRot == nil {
		t.Fatal("Factor analysis result is nil")
	}

	loadingsNoRot := extractMatDense(resultNoRot.Loadings)

	fmt.Println("\n========================================")
	fmt.Println("旋轉前的因子矩陣 (Factor Matrix)")
	fmt.Println("========================================")

	fmt.Println("\n我們的 Factor Matrix (未旋轉):")
	varNames := []string{"A1", "A2", "A3", "B1", "B2", "B3", "C1", "C2", "C3"}
	for i := 0; i < 9; i++ {
		fmt.Printf("%s: [%.6f, %.6f, %.6f]\n",
			varNames[i],
			loadingsNoRot.At(i, 0),
			loadingsNoRot.At(i, 1),
			loadingsNoRot.At(i, 2))
	}

	// SPSS Factor Matrix (pre-rotation)
	spssFactor := [][]float64{
		{0.465, 0.708, 0.006},
		{0.504, 0.497, 0.269},
		{0.581, 0.458, 0.083},
		{0.680, 0.044, -0.493},
		{0.656, -0.367, -0.467},
		{0.657, -0.076, -0.324},
		{0.516, -0.257, 0.596},
		{0.639, -0.441, 0.263},
		{0.635, -0.268, 0.302},
	}

	fmt.Println("\nSPSS Factor Matrix (未旋轉):")
	for i := 0; i < 9; i++ {
		fmt.Printf("%s: [%.3f, %.3f, %.3f]\n",
			varNames[i],
			spssFactor[i][0],
			spssFactor[i][1],
			spssFactor[i][2])
	}

	// Compare
	fmt.Println("\n比較 (我們 vs SPSS):")
	totalDiff := 0.0
	count := 0
	for i := 0; i < 9; i++ {
		diff1 := loadingsNoRot.At(i, 0) - spssFactor[i][0]
		diff2 := loadingsNoRot.At(i, 1) - spssFactor[i][1]
		diff3 := loadingsNoRot.At(i, 2) - spssFactor[i][2]

		fmt.Printf("%s: diff=[%.3f, %.3f, %.3f]\n",
			varNames[i], diff1, diff2, diff3)

		totalDiff += diff1*diff1 + diff2*diff2 + diff3*diff3
		count += 3
	}

	rmse := math.Sqrt(totalDiff / float64(count))
	fmt.Printf("\nRMSE vs SPSS Factor Matrix: %.6f\n", rmse)

	if rmse > 0.01 {
		t.Logf("⚠️  旋轉前的 Factor Matrix 就已經與 SPSS 不一致!")
		t.Logf("這可能是萃取方法 (PAF) 的問題")
	} else {
		t.Logf("✓ 旋轉前的 Factor Matrix 與 SPSS 一致")
	}
}
