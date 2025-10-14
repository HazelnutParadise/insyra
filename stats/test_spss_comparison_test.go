package stats_test

import (
	"fmt"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

func TestDetailedSPSSComparison(t *testing.T) {
	dt := readFactorAnalysisSampleCSV(t)

	result := stats.FactorAnalysis(dt, stats.FactorAnalysisOptions{
		Preprocess: stats.FactorPreprocessOptions{Standardize: true},
		Count:      stats.FactorCountSpec{Method: stats.FactorCountFixed, FixedK: 3},
		Extraction: stats.FactorExtractionPAF,
		Rotation: stats.FactorRotationOptions{
			Method: stats.FactorRotationOblimin,
			Delta:  0,
		},
		MinErr:  1e-9,
		MaxIter: 1000,
	})

	if result == nil {
		t.Fatal("Factor analysis result is nil")
	}

	varNames := []string{"A1", "A2", "A3", "B1", "B2", "B3", "C1", "C2", "C3"}

	fmt.Println("\n=== 詳細比較: 當前實現 vs SPSS ===")
	fmt.Println()

	// 1. Pre-rotation (Factor Matrix) 比較
	fmt.Println("【1. Pre-rotation Factor Matrix (旋轉前因子矩陣)】")
	fmt.Println("變數 | SPSS F1    | 當前 F1    | 差異      | SPSS F2    | 當前 F2    | 差異      | SPSS F3    | 當前 F3    | 差異")
	fmt.Println("-----|-----------|-----------|----------|-----------|-----------|----------|-----------|-----------|----------")

	preRotation := extractMatrix(result.UnrotatedLoadings)
	maxDiffPre := 0.0
	for i := 0; i < 9; i++ {
		fmt.Printf("%-4s | %9.6f | %9.6f | %8.5f | %9.6f | %9.6f | %8.5f | %9.6f | %9.6f | %8.5f\n",
			varNames[i],
			spssPreRotation[i][0], preRotation[i][0], preRotation[i][0]-spssPreRotation[i][0],
			spssPreRotation[i][1], preRotation[i][1], preRotation[i][1]-spssPreRotation[i][1],
			spssPreRotation[i][2], preRotation[i][2], preRotation[i][2]-spssPreRotation[i][2])

		for j := 0; j < 3; j++ {
			diff := abs(preRotation[i][j] - spssPreRotation[i][j])
			if diff > maxDiffPre {
				maxDiffPre = diff
			}
		}
	}
	fmt.Printf("\n最大絕對差異: %.6f\n", maxDiffPre)

	// 2. Pattern Matrix 比較
	fmt.Println("\n【2. Pattern Matrix (型樣矩陣) - Oblimin 旋轉後】")
	fmt.Println("變數 | SPSS F1    | 當前 F1    | 差異      | SPSS F2    | 當前 F2    | 差異      | SPSS F3    | 當前 F3    | 差異")
	fmt.Println("-----|-----------|-----------|----------|-----------|-----------|----------|-----------|-----------|----------")

	pattern := extractMatrix(result.Loadings)
	maxDiffPattern := 0.0
	for i := 0; i < 9; i++ {
		fmt.Printf("%-4s | %9.6f | %9.6f | %8.5f | %9.6f | %9.6f | %8.5f | %9.6f | %9.6f | %8.5f\n",
			varNames[i],
			spssPattern[i][0], pattern[i][0], pattern[i][0]-spssPattern[i][0],
			spssPattern[i][1], pattern[i][1], pattern[i][1]-spssPattern[i][1],
			spssPattern[i][2], pattern[i][2], pattern[i][2]-spssPattern[i][2])

		for j := 0; j < 3; j++ {
			diff := abs(pattern[i][j] - spssPattern[i][j])
			if diff > maxDiffPattern {
				maxDiffPattern = diff
			}
		}
	}
	fmt.Printf("\n最大絕對差異: %.6f\n", maxDiffPattern)

	// 3. Phi Matrix (因子相關矩陣) 比較
	fmt.Println("\n【3. Phi Matrix (因子相關矩陣)】")
	fmt.Println("\nSPSS Phi Matrix:")
	fmt.Println("     F1      F2      F3")
	for i := 0; i < 3; i++ {
		fmt.Printf("F%d  %.3f   %.3f   %.3f\n", i+1, spssPhi[i][0], spssPhi[i][1], spssPhi[i][2])
	}

	phi := extractMatrix(result.Phi)
	fmt.Println("\n當前實現 Phi Matrix:")
	fmt.Println("     F1      F2      F3")
	for i := 0; i < 3; i++ {
		fmt.Printf("F%d  %.3f   %.3f   %.3f\n", i+1, phi[i][0], phi[i][1], phi[i][2])
	}

	fmt.Println("\n差異 (當前 - SPSS):")
	fmt.Println("     F1      F2      F3")
	maxDiffPhi := 0.0
	for i := 0; i < 3; i++ {
		fmt.Printf("F%d  %+.3f   %+.3f   %+.3f\n", i+1,
			phi[i][0]-spssPhi[i][0],
			phi[i][1]-spssPhi[i][1],
			phi[i][2]-spssPhi[i][2])

		for j := 0; j < 3; j++ {
			diff := abs(phi[i][j] - spssPhi[i][j])
			if diff > maxDiffPhi && i != j { // 排除對角線
				maxDiffPhi = diff
			}
		}
	}
	fmt.Printf("\n最大絕對差異(排除對角線): %.6f\n", maxDiffPhi)

	// 結論
	fmt.Println("\n=== 結論 ===")
	fmt.Printf("Pre-rotation:   最大差異 = %.6f  ✓ %s\n", maxDiffPre, status(maxDiffPre < 0.001))
	fmt.Printf("Pattern Matrix: 最大差異 = %.6f  ✗ %s\n", maxDiffPattern, status(maxDiffPattern < 0.05))
	fmt.Printf("Phi Matrix:     最大差異 = %.6f  ✗ %s\n", maxDiffPhi, status(maxDiffPhi < 0.05))

	fmt.Println("\n【問題分析】")
	fmt.Println("✓ Pre-rotation 與 SPSS 完全一致 (差異 < 0.001)")
	fmt.Println("✗ Pattern Matrix 差異很大 (差異 > 0.3)")
	fmt.Println("✗ Phi Matrix 顯示近正交結構 (~0.01) vs SPSS 的斜交結構 (~0.2-0.3)")
	fmt.Println("\n原因: GPFobliq 算法收斂到不同的局部最小值")
	fmt.Println("     - SPSS: 斜交解 (因子間有相關 0.2-0.3)")
	fmt.Println("     - 當前: 近正交解 (因子間幾乎無相關 ~0.01)")
}

func extractMatrix(dt insyra.IDataTable) [][]float64 {
	var result [][]float64
	dt.AtomicDo(func(table *insyra.DataTable) {
		rows, cols := table.Size()
		result = make([][]float64, rows)
		for i := 0; i < rows; i++ {
			result[i] = make([]float64, cols)
			row := table.GetRow(i)
			for j := 0; j < cols; j++ {
				if v, ok := row.Get(j).(float64); ok {
					result[i][j] = v
				}
			}
		}
	})
	return result
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func status(pass bool) string {
	if pass {
		return "接近一致"
	}
	return "差異顯著"
}
