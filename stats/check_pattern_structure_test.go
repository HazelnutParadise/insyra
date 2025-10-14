package stats_test

import (
	"fmt"
	"testing"

	"github.com/HazelnutParadise/insyra/stats"
	"gonum.org/v1/gonum/mat"
)

func TestCheckPatternStructureRelationship(t *testing.T) {
	dt := readFactorAnalysisSampleCSV(t)

	result := stats.FactorAnalysis(dt, stats.FactorAnalysisOptions{
		Preprocess: stats.FactorPreprocessOptions{Standardize: true},
		Count:      stats.FactorCountSpec{Method: stats.FactorCountFixed, FixedK: 3},
		Extraction: stats.FactorExtractionPAF,
		Rotation:   stats.FactorRotationOptions{Method: stats.FactorRotationOblimin, Kappa: 0},
		MinErr:     1e-9,
		MaxIter:    1000,
	})

	if result == nil {
		t.Fatal("Factor analysis result is nil")
	}

	loadingsMat := extractMatDense(result.Loadings)
	phiMat := extractMatDense(result.Phi)

	fmt.Println("\n========================================")
	fmt.Println("檢查 Pattern-Structure 關係")
	fmt.Println("========================================")

	// According to statistical theory:
	// Structure Matrix = Pattern Matrix × Phi
	var computedStructure mat.Dense
	computedStructure.Mul(loadingsMat, phiMat)

	fmt.Println("\n我們的 Loadings (應該是 Pattern Matrix):")
	for i := 0; i < 3; i++ {
		varName := []string{"A1", "A2", "A3"}[i]
		fmt.Printf("%s: [%.6f, %.6f, %.6f]\n",
			varName,
			loadingsMat.At(i, 0),
			loadingsMat.At(i, 1),
			loadingsMat.At(i, 2))
	}

	fmt.Println("\n我們的 Phi:")
	for i := 0; i < 3; i++ {
		fmt.Printf("F%d: [%.6f, %.6f, %.6f]\n",
			i+1,
			phiMat.At(i, 0),
			phiMat.At(i, 1),
			phiMat.At(i, 2))
	}

	fmt.Println("\n計算的 Structure = Pattern × Phi:")
	for i := 0; i < 3; i++ {
		varName := []string{"A1", "A2", "A3"}[i]
		fmt.Printf("%s: [%.6f, %.6f, %.6f]\n",
			varName,
			computedStructure.At(i, 0),
			computedStructure.At(i, 1),
			computedStructure.At(i, 2))
	}

	// SPSS Pattern Matrix
	spssPattern := [][]float64{
		{0.067, 0.849, -0.163},
		{-0.090, 0.724, 0.177},
		{0.127, 0.677, 0.085},
	}

	// SPSS Structure Matrix
	spssStructure := [][]float64{
		{0.224, 0.833, 0.027},
		{0.142, 0.738, 0.293},
		{0.319, 0.725, 0.259},
	}

	// SPSS Phi
	spssPhi := [][]float64{
		{1.0, 0.245, 0.311},
		{0.245, 1.0, 0.199},
		{0.311, 0.199, 1.0},
	}

	fmt.Println("\nSPSS Pattern Matrix:")
	for i := 0; i < 3; i++ {
		varName := []string{"A1", "A2", "A3"}[i]
		fmt.Printf("%s: [%.3f, %.3f, %.3f]\n",
			varName,
			spssPattern[i][0],
			spssPattern[i][1],
			spssPattern[i][2])
	}

	fmt.Println("\nSPSS Phi:")
	for i := 0; i < 3; i++ {
		fmt.Printf("F%d: [%.3f, %.3f, %.3f]\n",
			i+1,
			spssPhi[i][0],
			spssPhi[i][1],
			spssPhi[i][2])
	}

	fmt.Println("\nSPSS Structure Matrix:")
	for i := 0; i < 3; i++ {
		varName := []string{"A1", "A2", "A3"}[i]
		fmt.Printf("%s: [%.3f, %.3f, %.3f]\n",
			varName,
			spssStructure[i][0],
			spssStructure[i][1],
			spssStructure[i][2])
	}

	// Verify SPSS's relationship: Structure = Pattern × Phi
	spssPatternMat := mat.NewDense(3, 3, nil)
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			spssPatternMat.Set(i, j, spssPattern[i][j])
		}
	}

	spssPhiMat := mat.NewDense(3, 3, nil)
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			spssPhiMat.Set(i, j, spssPhi[i][j])
		}
	}

	var spssComputedStructure mat.Dense
	spssComputedStructure.Mul(spssPatternMat, spssPhiMat)

	fmt.Println("\nSPSS 計算檢驗 (Structure = Pattern × Phi):")
	for i := 0; i < 3; i++ {
		varName := []string{"A1", "A2", "A3"}[i]
		fmt.Printf("%s: [%.3f, %.3f, %.3f] vs SPSS [%.3f, %.3f, %.3f]\n",
			varName,
			spssComputedStructure.At(i, 0),
			spssComputedStructure.At(i, 1),
			spssComputedStructure.At(i, 2),
			spssStructure[i][0],
			spssStructure[i][1],
			spssStructure[i][2])
	}

	fmt.Println("\n========================================")
	fmt.Println("結論:")
	fmt.Println("========================================")
	fmt.Println("如果 SPSS 的 Pattern × Phi ≈ Structure,")
	fmt.Println("則統計學關係正確。")
	fmt.Println("我們需要確保輸出的是 Pattern Matrix!")
}
