package stats

import (
	"fmt"
	"math"
	"testing"

	"github.com/HazelnutParadise/insyra/internal/fa"
	"gonum.org/v1/gonum/mat"
)

func TestVarimaxWithSPSSInput(t *testing.T) {
	// SPSS ML 未旋轉載荷 (完全使用SPSS的值)
	spssUnrotated := [][]float64{
		{0.426, 0.754, -0.008},  // A1
		{0.432, 0.539, 0.273},   // A2
		{0.532, 0.494, 0.108},   // A3
		{0.738, 0.056, -0.418},  // B1
		{0.727, -0.362, -0.349}, // B2
		{0.690, -0.069, -0.240}, // B3
		{0.460, -0.163, 0.671},  // C1
		{0.627, -0.367, 0.378},  // C2
		{0.605, -0.195, 0.399},  // C3
	}

	// SPSS Varimax旋轉後載荷
	spssRotated := [][]float64{
		{0.137, -0.077, 0.852}, // A1
		{0.020, 0.225, 0.707},  // A2
		{0.208, 0.166, 0.684},  // A3
		{0.795, 0.035, 0.298},  // B1
		{0.848, 0.243, -0.068}, // B2
		{0.680, 0.195, 0.195},  // B3
		{-0.033, 0.814, 0.158}, // C1
		{0.324, 0.752, 0.001},  // C2
		{0.252, 0.691, 0.148},  // C3
	}

	// 轉換為 mat.Dense
	loadings := mat.NewDense(9, 3, nil)
	for i := 0; i < 9; i++ {
		for j := 0; j < 3; j++ {
			loadings.Set(i, j, spssUnrotated[i][j])
		}
	}

	// 調用我們的 Varimax 旋轉
	fmt.Println("用SPSS的未旋轉載荷調用我們的Varimax...")
	result := fa.Varimax(loadings, true, 1e-05, 1000)

	if result.Converged {
		fmt.Printf("✓ 旋轉收斂於 %d 次迭代\n\n", result.Iterations)
	} else {
		fmt.Printf("✗ 旋轉未收斂，%d 次迭代\n\n", result.Iterations)
	}

	// 提取旋轉後載荷
	oursRotated := result.Loadings

	// 打印我們的旋轉結果
	fmt.Println("我們的旋轉後載荷:")
	for i := 0; i < 9; i++ {
		varName := ""
		if i < 3 {
			varName = fmt.Sprintf("A%d", i+1)
		} else if i < 6 {
			varName = fmt.Sprintf("B%d", i-2)
		} else {
			varName = fmt.Sprintf("C%d", i-5)
		}
		fmt.Printf("%s: %+.3f %+.3f %+.3f\n",
			varName,
			oursRotated.At(i, 0),
			oursRotated.At(i, 1),
			oursRotated.At(i, 2))
	}

	// 測試所有排列和符號
	permutations := [][3]int{
		{0, 1, 2}, {0, 2, 1}, {1, 0, 2}, {1, 2, 0}, {2, 0, 1}, {2, 1, 0},
	}

	bestRMSE := 999.0
	bestPerm := 0
	bestSigns := [3]int{1, 1, 1}

	for pi, perm := range permutations {
		for s1 := -1; s1 <= 1; s1 += 2 {
			for s2 := -1; s2 <= 1; s2 += 2 {
				for s3 := -1; s3 <= 1; s3 += 2 {
					signs := [3]int{s1, s2, s3}
					sumSqErr := 0.0
					for i := 0; i < 9; i++ {
						for j := 0; j < 3; j++ {
							oursVal := oursRotated.At(i, perm[j]) * float64(signs[j])
							diff := spssRotated[i][j] - oursVal
							sumSqErr += diff * diff
						}
					}
					rmse := math.Sqrt(sumSqErr / 27.0)
					if rmse < bestRMSE {
						bestRMSE = rmse
						bestPerm = pi
						bestSigns = signs
					}
				}
			}
		}
	}

	permNames := []string{"F1,F2,F3", "F1,F3,F2", "F2,F1,F3", "F2,F3,F1", "F3,F1,F2", "F3,F2,F1"}
	fmt.Printf("\n最佳匹配: %s, 符號(%+d,%+d,%+d), RMSE=%.6f\n",
		permNames[bestPerm], bestSigns[0], bestSigns[1], bestSigns[2], bestRMSE)

	if bestRMSE < 0.05 {
		fmt.Println("\n✓✓✓ 完美匹配！問題解決！")
		t.Logf("RMSE = %.6f < 0.05, 完美匹配", bestRMSE)
	} else {
		fmt.Println("\n✗ 仍有差異，Varimax實現可能不同")
		t.Errorf("RMSE = %.6f >= 0.05, 仍有差異", bestRMSE)
	}
}
