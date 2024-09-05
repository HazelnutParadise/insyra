package stats

import (
	"math/big"

	"github.com/HazelnutParadise/insyra"
)

// CorrelationMethod 定義了相關係數的計算方法
type CorrelationMethod int

const (
	// PearsonCorrelation 表示皮爾森相關係數的計算方法，用於測量線性相關性
	// PearsonCorrelation means Pearson correlation coefficient, used to measure linear correlation.
	PearsonCorrelation CorrelationMethod = iota
	// KendallCorrelation 表示肯德爾秩相關係數的計算方法，用於測量單調相關性
	// KendallCorrelation means Kendall rank correlation coefficient, used to measure monotonic correlation.
	KendallCorrelation
	// SpearmanCorrelation 表示斯皮爾曼秩相關係數的計算方法，基於排序後的數據。
	// SpearmanCorrelation means Spearman rank correlation coefficient, based on sorted data.
	SpearmanCorrelation
)

// Covariance calculates the covariance between two datasets.
// Always returns *big.Rat.
func Covariance(dlX, dlY *insyra.DataList) *big.Rat {
	meanX := dlX.Mean(true).(*big.Rat)
	meanY := dlY.Mean(true).(*big.Rat)

	cov := new(big.Rat)
	for i := 0; i < dlX.Len(); i++ {
		x := new(big.Rat).SetFloat64(dlX.Data()[i].(float64))
		y := new(big.Rat).SetFloat64(dlY.Data()[i].(float64))

		x.Sub(x, meanX) // (X_i - mean_X)
		y.Sub(y, meanY) // (Y_i - mean_Y)

		term := new(big.Rat).Mul(x, y) // (X_i - mean_X) * (Y_i - mean_Y)
		cov.Add(cov, term)             // 累加到協方差
	}

	// 取平均
	length := new(big.Rat).SetInt64(int64(dlX.Len()))
	cov.Quo(cov, length) // cov = cov / n

	return cov
}

// Correlation calculates the correlation coefficient between two datasets.
// Supports Pearson, Kendall, and Spearman methods.
// If highPrecision is set to true, it returns *big.Rat, otherwise float64.
func Correlation(dlX, dlY *insyra.DataList, method CorrelationMethod, highPrecision ...bool) interface{} {
	if len(highPrecision) > 1 {
		insyra.LogWarning("stats.Correlation: Too many arguments.")
		return nil
	}

	var result *big.Rat
	switch method {
	case PearsonCorrelation:
		result = pearsonCorrelation(dlX, dlY)
	case KendallCorrelation:
		result = kendallCorrelation(dlX, dlY)
	case SpearmanCorrelation:
		result = spearmanCorrelation(dlX, dlY)
	default:
		insyra.LogWarning("stats.Correlation: Unsupported method.")
		return nil // 無效的 method，返回 nil
	}

	if result == nil {
		insyra.LogWarning("stats.Correlation: Cannot calculate correlation.")
		return nil
	}

	// 根據 highPrecision 決定是否轉換為 float64
	if len(highPrecision) > 0 && !highPrecision[0] {
		f64Result, _ := result.Float64()
		return f64Result
	}

	return result
}

// ======================= calculation functions =======================

// pearsonCorrelation calculates Pearson correlation coefficient.
func pearsonCorrelation(dlX, dlY *insyra.DataList) *big.Rat {
	cov := Covariance(dlX, dlY)

	// 計算標準差
	stdX := dlX.Stdev(true).(*big.Rat)
	stdY := dlY.Stdev(true).(*big.Rat)

	// 防止除以0
	if stdX.Sign() == 0 || stdY.Sign() == 0 {
		return nil
	}

	// 計算相關係數
	corr := new(big.Rat).Quo(cov, new(big.Rat).Mul(stdX, stdY))

	return corr
}

// kendallCorrelation calculates Kendall rank correlation coefficient.
func kendallCorrelation(dlX, dlY *insyra.DataList) *big.Rat {
	n := dlX.Len()
	// 計算 Concordant 和 Discordant 配對
	var concordant, discordant int
	for i := 0; i < n-1; i++ {
		for j := i + 1; j < n; j++ {
			xi, yi := dlX.Data()[i].(float64), dlY.Data()[i].(float64)
			xj, yj := dlX.Data()[j].(float64), dlY.Data()[j].(float64)
			signX := xi - xj
			signY := yi - yj
			if signX*signY > 0 {
				concordant++
			} else if signX*signY < 0 {
				discordant++
			}
		}
	}

	// 計算 Kendall's Tau
	tau := new(big.Rat).SetFloat64(float64(concordant-discordant) / float64(n*(n-1)/2))
	return tau
}

// spearmanCorrelation calculates Spearman rank correlation coefficient.
func spearmanCorrelation(dlX, dlY *insyra.DataList) *big.Rat {
	// 對 X 和 Y 進行排序並分別計算排序
	dlX.Sort() // 使用 dlX 的內部排序方法
	dlY.Sort() // 使用 dlY 的內部排序方法

	// 計算 Pearson 相關係數，基於排序後的數據
	return pearsonCorrelation(dlX, dlY)
}
