// linear_regression.go - 線性回歸分析(尚未完成)

package stats

import (
	"math"
	"math/big"

	"github.com/HazelnutParadise/insyra"
	"gonum.org/v1/gonum/stat/distuv"
)

// LinearRegressionResult holds the result of a linear regression, including slope, intercept, and other statistical details.
type LinearRegressionResult struct {
	Slope            float64   // 斜率
	Intercept        float64   // 截距
	Residuals        []float64 // 殘差
	Rsquared         float64   // R-squared
	AdjustedRsquared float64   // 調整後的 R-squared
	StandardError    float64   // 標準誤差
	TValue           float64   // t 值
	Pvalue           float64   // p 值
}

// LinearRegression performs simple linear regression on two datasets (X and Y).
// It returns the slope, intercept, residuals, R-squared, and other statistical details.
func LinearRegression(dlX, dlY insyra.IDataList) *LinearRegressionResult {
	if dlX.Len() != dlY.Len() || dlX.Len() == 0 {
		insyra.LogWarning("stats.LinearRegression: data lists must have the same non-zero length.")
		return nil
	}

	// 計算 X 和 Y 的均值
	meanX := dlX.Mean(true).(*big.Rat)
	meanY := dlY.Mean(true).(*big.Rat)

	// 初始化變量
	numerator := new(big.Rat)
	denominator := new(big.Rat)
	var residuals []float64
	var sumSquaredResiduals, sumTotalSquares float64

	// 計算斜率的分子和分母
	for i := 0; i < dlX.Len(); i++ {
		x := new(big.Rat).SetFloat64(dlX.Data()[i].(float64))
		y := new(big.Rat).SetFloat64(dlY.Data()[i].(float64))

		// (x_i - meanX) 和 (y_i - meanY)
		diffX := new(big.Rat).Sub(x, meanX)
		diffY := new(big.Rat).Sub(y, meanY)

		// 分子: sum((x_i - meanX) * (y_i - meanY))
		numerator.Add(numerator, new(big.Rat).Mul(diffX, diffY))

		// 分母: sum((x_i - meanX)^2)
		denominator.Add(denominator, new(big.Rat).Mul(diffX, diffX))
	}

	// 防止除以 0
	if denominator.Sign() == 0 {
		insyra.LogWarning("stats.LinearRegression: denominator is zero, unable to calculate slope.")
		return nil
	}

	// 計算斜率 beta_1
	slopeRat := new(big.Rat).Quo(numerator, denominator)
	slopeFloat, _ := slopeRat.Float64()

	// 計算截距 beta_0 = meanY - slope * meanX
	interceptRat := new(big.Rat).Sub(meanY, new(big.Rat).Mul(slopeRat, meanX))
	interceptFloat, _ := interceptRat.Float64()

	// 計算 y 的均值 (修正 R-squared 計算)
	meanYFloat, _ := meanY.Float64()

	// 計算殘差和平方和
	for i := 0; i < dlX.Len(); i++ {
		x := dlX.Data()[i].(float64)
		y := dlY.Data()[i].(float64)

		// 預測值: y_pred = beta_0 + beta_1 * x_i
		yPred := interceptFloat + slopeFloat*x

		// 殘差: residual = y_i - y_pred
		residual := y - yPred
		residuals = append(residuals, residual)

		// 計算殘差平方和
		sumSquaredResiduals += residual * residual

		// 計算總平方和 (y_i - meanY)^2
		sumTotalSquares += (y - meanYFloat) * (y - meanYFloat)
	}

	// 計算 R-squared 和 Adjusted R-squared
	rSquared := 1 - (sumSquaredResiduals / sumTotalSquares)
	adjustedRsquared := 1 - (1-rSquared)*float64(dlX.Len()-1)/float64(dlX.Len()-2)

	// 計算標準誤差
	standardError := math.Sqrt(sumSquaredResiduals / float64(dlX.Len()-2))

	// 計算 t 值
	tValue := slopeFloat / standardError

	// 使用 t 分布來計算 P 值
	pValue := calculatePValue(tValue, dlX.Len()-2)

	return &LinearRegressionResult{
		Slope:            slopeFloat,
		Intercept:        interceptFloat,
		Residuals:        residuals,
		Rsquared:         rSquared,
		AdjustedRsquared: adjustedRsquared,
		StandardError:    standardError,
		TValue:           tValue,
		Pvalue:           pValue,
	}
}

// calculatePValue 基於 t 值和自由度計算 P 值
func calculatePValue(tValue float64, df int) float64 {
	if df <= 0 {
		return 1.0 // 當自由度無效時，P-value 為 1
	}

	// 使用 t 分布來計算雙尾 P-value
	tDist := distuv.StudentsT{
		Mu:    0,           // 平均值
		Sigma: 1,           // 標準差
		Nu:    float64(df), // 自由度
	}

	// 計算 t 值的絕對值，然後進行雙尾檢驗
	tValueAbs := math.Abs(tValue)
	pValue := 2 * (1 - tDist.CDF(tValueAbs))

	return pValue
}
