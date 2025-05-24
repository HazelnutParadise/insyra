package stats

import (
	"math"

	"github.com/HazelnutParadise/insyra"
)

// SkewnessMethod defines available skewness calculation methods.
type SkewnessMethod int

const (
	SkewnessG1           SkewnessMethod = iota + 1 // Type 1: G1 (default)
	SkewnessAdjusted                               // Type 2: Adjusted Fisher-Pearson
	SkewnessBiasAdjusted                           // Type 3: Bias-adjusted
)

// Skewness calculates the skewness of a sample using the specified method.
//
// method default: SkewnessG1（type 1）。
func Skewness(sample any, method ...SkewnessMethod) float64 {
	// 數據預處理和錯誤檢查
	d, dLen := insyra.ProcessData(sample)
	if dLen == 0 {
		insyra.LogWarning("stats.Skewness: empty data")
		return math.NaN()
	}

	d64 := insyra.SliceToF64(d)
	dl := insyra.NewDataList(d64)
	n := float64(dl.Len())

	// 方法選擇
	usemethod := SkewnessG1
	if len(method) > 0 {
		usemethod = method[0]
	}
	if len(method) > 1 {
		insyra.LogWarning("stats.Skewness: too many methods specified, returning NaN")
		return math.NaN()
	}

	// 特定方法的數據長度檢查
	if usemethod == SkewnessAdjusted && n < 3 {
		insyra.LogWarning("stats.Skewness: insufficient data for adjusted method (n < 3)")
		return math.NaN()
	}

	// 計算中心矩
	m2 := CalculateMoment(dl, 2, true)
	m3 := CalculateMoment(dl, 3, true)

	// 零方差檢查
	if m2 == 0 {
		return 0 // 如果方差為零，偏度為零
	}

	// 計算基本偏度
	g1 := m3 / math.Pow(m2, 1.5)

	// 根據方法應用調整
	switch usemethod {
	case SkewnessG1:
		return g1
	case SkewnessAdjusted:
		return g1 * math.Sqrt(n*(n-1)) / (n - 2)
	case SkewnessBiasAdjusted:
		return g1 * math.Pow((n-1)/n, 1.5)
	default:
		insyra.LogWarning("stats.Skewness: unknown method")
		return math.NaN()
	}
}
