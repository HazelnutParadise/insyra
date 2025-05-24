// linear_regression.go - 線性回歸分析(尚未完成)

package stats

import (
	"math"

	"github.com/HazelnutParadise/insyra"
)

// LinearRegressionResult holds the result of a linear regression, including slope, intercept, and other statistical details.
type LinearRegressionResult struct {
	Slope            float64   // 斜率
	Intercept        float64   // 截距
	Residuals        []float64 // 殘差
	RSquared         float64   // R-squared
	AdjustedRSquared float64   // 調整後的 R-squared
	StandardError    float64   // 標準誤差
	TValue           float64   // t 值
	PValue           float64   // p 值
}

// LinearRegression performs simple linear regression on two datasets (X and Y).
// It returns the slope, intercept, residuals, R-squared, and other statistical details.
func LinearRegression(dlX, dlY insyra.IDataList) *LinearRegressionResult {
	if dlX.Len() != dlY.Len() || dlX.Len() == 0 {
		insyra.LogWarning("stats.LinearRegression: data lists must have the same non-zero length.")
		return nil
	}

	xVals := dlX.ToF64Slice()
	yVals := dlY.ToF64Slice()
	n := float64(len(xVals))

	// 計算平均值
	var sumX, sumY float64
	for i := 0; i < len(xVals); i++ {
		sumX += xVals[i]
		sumY += yVals[i]
	}
	meanX := sumX / n
	meanY := sumY / n

	// 計算斜率
	var numerator, denominator float64
	for i := 0; i < len(xVals); i++ {
		diffX := xVals[i] - meanX
		diffY := yVals[i] - meanY
		numerator += diffX * diffY
		denominator += diffX * diffX
	}
	if denominator == 0 {
		insyra.LogWarning("stats.LinearRegression: denominator is zero, unable to calculate slope.")
		return nil
	}
	slope := numerator / denominator
	intercept := meanY - slope*meanX

	// 計算預測值與殘差
	residuals := make([]float64, len(xVals))
	var sumSquaredResiduals, sumTotalSquares float64
	for i := 0; i < len(xVals); i++ {
		yPred := intercept + slope*xVals[i]
		residual := yVals[i] - yPred
		residuals[i] = residual
		sumSquaredResiduals += residual * residual
		sumTotalSquares += (yVals[i] - meanY) * (yVals[i] - meanY)
	}

	// 計算統計指標
	rSquared := 1 - (sumSquaredResiduals / sumTotalSquares)
	adjustedRSquared := 1 - (1-rSquared)*(n-1)/(n-2)

	sumXSquared := 0.0
	for i := 0; i < len(xVals); i++ {
		sumXSquared += (xVals[i] - meanX) * (xVals[i] - meanX)
	}
	mse := sumSquaredResiduals / (n - 2)
	standardError := math.Sqrt(mse / sumXSquared)
	tValue := slope / standardError
	pValue := 2.0 * tCDF(-math.Abs(tValue), int(n-2))

	return &LinearRegressionResult{
		Slope:            slope,
		Intercept:        intercept,
		Residuals:        residuals,
		RSquared:         rSquared,
		AdjustedRSquared: adjustedRSquared,
		StandardError:    standardError,
		TValue:           tValue,
		PValue:           pValue,
	}
}

// tCDF returns the cumulative distribution function (CDF) of the Student's t-distribution
func tCDF(t float64, df int) float64 {
	x := float64(df) / (float64(df) + t*t)

	// 根據 t 值的正負決定如何計算 CDF
	if t <= 0 {
		// t ≤ 0 時，CDF(t) = 0.5 * Beta(df/2, 1/2, x)
		return 0.5 * betaInc(float64(df)/2.0, 0.5, x)
	}
	// t > 0 時，CDF(t) = 1 - 0.5 * Beta(df/2, 1/2, x)
	return 1.0 - 0.5*betaInc(float64(df)/2.0, 0.5, x)
}

// betaInc implements the regularized incomplete beta function I_x(a, b)
func betaInc(a, b, x float64) float64 {
	if x < 0.0 || x > 1.0 {
		return math.NaN()
	}

	if x == 0.0 || x == 1.0 {
		return x
	}

	// 正確使用 Lgamma 回傳值
	lg1, _ := math.Lgamma(a + b)
	lg2, _ := math.Lgamma(a)
	lg3, _ := math.Lgamma(b)

	bt := math.Exp(lg1 - lg2 - lg3 + a*math.Log(x) + b*math.Log(1.0-x))

	if x < (a+1.0)/(a+b+2.0) {
		return bt * betaCF(a, b, x) / a
	}

	return 1.0 - bt*betaCF(b, a, 1.0-x)/b
}

// betaCF implements the continued fraction approximation for the incomplete beta function
func betaCF(a, b, x float64) float64 {
	const maxIter = 100
	const eps = 1e-14
	const fpmin = 1e-30

	m2 := 0
	aa := 0.0
	c := 1.0
	d := 1.0 - (a+b)*x/(a+1.0)
	if math.Abs(d) < fpmin {
		d = fpmin
	}
	d = 1.0 / d
	h := d

	for m := 1; m <= maxIter; m++ {
		m2 = 2 * m

		// even step
		aa = float64(m) * (b - float64(m)) * x / ((a + float64(m2) - 1) * (a + float64(m2)))
		d = 1.0 + aa*d
		if math.Abs(d) < fpmin {
			d = fpmin
		}
		c = 1.0 + aa/c
		if math.Abs(c) < fpmin {
			c = fpmin
		}
		d = 1.0 / d
		h *= d * c

		// odd step
		aa = -(a + float64(m)) * (a + b + float64(m)) * x / ((a + float64(m2)) * (a + float64(m2) + 1))
		d = 1.0 + aa*d
		if math.Abs(d) < fpmin {
			d = fpmin
		}
		c = 1.0 + aa/c
		if math.Abs(c) < fpmin {
			c = fpmin
		}
		d = 1.0 / d
		delta := d * c
		h *= delta

		if math.Abs(delta-1.0) < eps {
			break
		}
	}

	return h
}
