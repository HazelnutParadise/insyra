package stats_test

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra/stats"
)

// SPSS 測試 2: PCA + No Rotation
var spssPCA_NoRotation = [][]float64{
	{0.484, 0.746, 0.001},   // A1
	{0.549, 0.575, 0.323},   // A2
	{0.634, 0.536, 0.091},   // A3
	{0.698, 0.036, -0.556},  // B1
	{0.656, -0.383, -0.499}, // B2
	{0.711, -0.102, -0.428}, // B3
	{0.539, -0.294, 0.653},  // C1
	{0.666, -0.498, 0.280},  // C2
	{0.686, -0.332, 0.357},  // C3
}

// SPSS 測試 3: PCA + Varimax Rotation
var spssPCA_Varimax = [][]float64{
	{0.145, -0.095, 0.872}, // A1
	{-0.013, 0.237, 0.825}, // A2
	{0.221, 0.155, 0.790},  // A3
	{0.848, 0.020, 0.279},  // B1
	{0.872, 0.243, -0.082}, // B2
	{0.793, 0.180, 0.195},  // B3
	{-0.069, 0.880, 0.156}, // C1
	{0.331, 0.812, -0.022}, // C2
	{0.251, 0.791, 0.141},  // C3
}

func TestCompareAllMethods(t *testing.T) {
	dt := readFactorAnalysisSampleCSV(t)

	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("因素分析全面比較測試 - 所有方法與 SPSS 對比")
	fmt.Println(strings.Repeat("=", 80) + "\n")

	tests := []struct {
		name         string
		extraction   stats.FactorExtractionMethod
		rotation     stats.FactorRotationMethod
		delta        float64
		spssLoadings [][]float64
		spssPhi      [][]float64
		expectOrthog bool // 是否期望正交 (Phi 應該是單位矩陣)
	}{
		{
			name:         "PCA + No Rotation",
			extraction:   stats.FactorExtractionPCA,
			rotation:     stats.FactorRotationNone,
			spssLoadings: spssPCA_NoRotation,
			expectOrthog: true,
		},
		{
			name:         "PCA + Varimax",
			extraction:   stats.FactorExtractionPCA,
			rotation:     stats.FactorRotationVarimax,
			spssLoadings: spssPCA_Varimax,
			expectOrthog: true,
		},
		{
			name:         "PAF + Oblimin",
			extraction:   stats.FactorExtractionPAF,
			rotation:     stats.FactorRotationOblimin,
			delta:        0,
			spssLoadings: spssPattern,
			spssPhi:      spssPhi,
			expectOrthog: false,
		},
	}

	for i, tc := range tests {
		fmt.Printf("\n【測試 %d: %s】\n", i+1, tc.name)
		fmt.Println(strings.Repeat("-", 80))

		result := stats.FactorAnalysis(dt, stats.FactorAnalysisOptions{
			Preprocess: stats.FactorPreprocessOptions{Standardize: true},
			Count:      stats.FactorCountSpec{Method: stats.FactorCountFixed, FixedK: 3},
			Extraction: tc.extraction,
			Rotation: stats.FactorRotationOptions{
				Method: tc.rotation,
				Delta:  tc.delta,
			},
			MinErr:  1e-9,
			MaxIter: 1000,
		})

		if result == nil {
			t.Errorf("%s: 結果為 nil", tc.name)
			continue
		}

		// 提取 Pattern/Component Matrix
		loadings := extractMatrix(result.Loadings)
		spssLoadingsMat := convertToMat(tc.spssLoadings)

		// 對齊因子
		perm, signs, aligned := alignFactorsMatrix(spssLoadingsMat, loadings)
		maxAbs, rmse := compareMatricesSlice(aligned, tc.spssLoadings)

		// 輸出結果
		fmt.Printf("\n因子排列: %v\n", perm)
		fmt.Printf("符號調整: %v\n", signs)
		fmt.Printf("\nLoadings 比較:\n")
		fmt.Printf("  最大絕對差異: %.6f\n", maxAbs)
		fmt.Printf("  RMSE:         %.6f\n", rmse)

		// 評估結果
		var status string
		if rmse < 0.01 {
			status = "✓ 完美匹配"
		} else if rmse < 0.05 {
			status = "✓ 良好匹配"
		} else if rmse < 0.1 {
			status = "△ 可接受差異"
		} else {
			status = "✗ 差異顯著"
		}
		fmt.Printf("  評估:         %s\n", status)

		// 如果是斜交旋轉,檢查 Phi 矩陣
		if !tc.expectOrthog && tc.spssPhi != nil && result.Phi != nil {
			phi := extractMatrix(result.Phi)
			phiMaxAbs, phiRMSE := compareMatricesSlice(phi, tc.spssPhi)

			fmt.Printf("\nPhi Matrix 比較:\n")
			fmt.Printf("  最大絕對差異: %.6f\n", phiMaxAbs)
			fmt.Printf("  RMSE:         %.6f\n", phiRMSE)

			var phiStatus string
			if phiRMSE < 0.01 {
				phiStatus = "✓ 完美匹配"
			} else if phiRMSE < 0.05 {
				phiStatus = "✓ 良好匹配"
			} else if phiRMSE < 0.1 {
				phiStatus = "△ 可接受差異"
			} else {
				phiStatus = "✗ 差異顯著"
			}
			fmt.Printf("  評估:         %s\n", phiStatus)
		}

		// 如果是正交旋轉,確認 Phi 接近單位矩陣
		if tc.expectOrthog && result.Phi != nil {
			phi := extractMatrix(result.Phi)
			maxOffDiag := 0.0
			for i := 0; i < len(phi); i++ {
				for j := 0; j < len(phi[i]); j++ {
					if i != j {
						if math.Abs(phi[i][j]) > maxOffDiag {
							maxOffDiag = math.Abs(phi[i][j])
						}
					}
				}
			}
			fmt.Printf("\nPhi Matrix (正交性檢查):\n")
			fmt.Printf("  最大非對角線值: %.6f\n", maxOffDiag)
			if maxOffDiag < 0.01 {
				fmt.Printf("  評估:           ✓ 正交 (近似單位矩陣)\n")
			} else {
				fmt.Printf("  評估:           △ 不完全正交\n")
			}
		}
	}

	// 總結
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("總結")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("\n建議:")
	fmt.Println("1. PCA 方法通常更穩定,建議優先測試")
	fmt.Println("2. 正交旋轉(Varimax/None)比斜交旋轉(Oblimin)更容易收斂到一致結果")
	fmt.Println("3. Oblimin 旋轉對初始值敏感,可能收斂到不同局部最優")
}

func convertToMat(data [][]float64) [][]float64 {
	return data
}

func alignFactorsMatrix(ref, target [][]float64) ([]int, []int, [][]float64) {
	if len(ref) == 0 || len(target) == 0 || len(ref[0]) == 0 {
		nFactors := 0
		if len(ref) > 0 && len(ref[0]) > 0 {
			nFactors = len(ref[0])
		} else if len(target) > 0 && len(target[0]) > 0 {
			nFactors = len(target[0])
		}
		perm := make([]int, nFactors)
		for i := range perm {
			perm[i] = i
		}
		signs := make([]int, nFactors)
		for i := range signs {
			signs[i] = 1
		}
		return perm, signs, target
	}

	nVars := len(ref)
	nFactors := len(ref[0])

	bestRMSE := math.Inf(1)
	var bestPerm []int
	var bestSigns []int
	var bestAligned [][]float64

	// Generate all permutations
	perm := make([]int, nFactors)
	used := make([]bool, nFactors)

	var generatePermutations func(int)
	generatePermutations = func(pos int) {
		if pos == nFactors {
			// Try all sign combinations for this permutation
			for signMask := 0; signMask < (1 << nFactors); signMask++ {
				signs := make([]int, nFactors)
				for i := 0; i < nFactors; i++ {
					if (signMask>>i)&1 == 1 {
						signs[i] = -1
					} else {
						signs[i] = 1
					}
				}

				// Apply permutation and signs
				aligned := make([][]float64, nVars)
				for i := 0; i < nVars; i++ {
					aligned[i] = make([]float64, nFactors)
					for j := 0; j < nFactors; j++ {
						aligned[i][j] = target[i][perm[j]] * float64(signs[j])
					}
				}

				// Calculate RMSE
				_, rmse := compareMatricesSlice(aligned, ref)

				if rmse < bestRMSE {
					bestRMSE = rmse
					bestPerm = make([]int, nFactors)
					copy(bestPerm, perm)
					bestSigns = make([]int, nFactors)
					copy(bestSigns, signs)
					bestAligned = make([][]float64, nVars)
					for i := range aligned {
						bestAligned[i] = make([]float64, nFactors)
						copy(bestAligned[i], aligned[i])
					}
				}
			}
			return
		}

		for i := 0; i < nFactors; i++ {
			if !used[i] {
				used[i] = true
				perm[pos] = i
				generatePermutations(pos + 1)
				used[i] = false
			}
		}
	}

	generatePermutations(0)

	return bestPerm, bestSigns, bestAligned
}

func compareMatricesSlice(a, b [][]float64) (maxAbs float64, rmse float64) {
	var sumSq float64
	var count int

	for i := 0; i < len(a); i++ {
		for j := 0; j < len(a[i]); j++ {
			diff := a[i][j] - b[i][j]
			absDiff := math.Abs(diff)
			if absDiff > maxAbs {
				maxAbs = absDiff
			}
			sumSq += diff * diff
			count++
		}
	}

	if count > 0 {
		rmse = math.Sqrt(sumSq / float64(count))
	}

	return maxAbs, rmse
}
