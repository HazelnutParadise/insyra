// linear_regression.go - 線性回歸分析(尚未完成)

package stats

import (
	"math"
	"math/big"

	"github.com/HazelnutParadise/Go-Utils/conv"
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

	// 計算 X 和 Y 值
	meanX := new(big.Rat).SetFloat64(dlX.Mean())
	meanY := new(big.Rat).SetFloat64(dlY.Mean())

	// 初始化變量
	numerator := new(big.Rat)
	denominator := new(big.Rat)
	var residuals []float64
	var sumSquaredResiduals, sumTotalSquares float64

	// 計算斜率的分子和分母
	for i := 0; i < dlX.Len(); i++ {
		x := new(big.Rat).SetFloat64(conv.ParseF64(dlX.Data()[i]))
		y := new(big.Rat).SetFloat64(conv.ParseF64(dlY.Data()[i]))

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
		x := conv.ParseF64(dlX.Data()[i])
		y := conv.ParseF64(dlY.Data()[i])

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

	// 計算 X 的平方和
	sumXSquared := 0.0
	meanXFloat, _ := meanX.Float64()
	for i := 0; i < dlX.Len(); i++ {
		x := conv.ParseF64(dlX.Data()[i])
		sumXSquared += (x - meanXFloat) * (x - meanXFloat)
	}

	// 修正標準誤差的計算
	n := float64(dlX.Len())
	mse := sumSquaredResiduals / (n - 2) // Mean Square Error
	standardError := math.Sqrt(mse / sumXSquared)

	// 修正 t 值的計算
	tValue := slopeFloat / standardError

	// 修改 p 值計算
	degreesOfFreedom := dlX.Len() - 2
	pValue := 2.0 * tCDF(-math.Abs(tValue), degreesOfFreedom)

	return &LinearRegressionResult{
		Slope:            slopeFloat,
		Intercept:        interceptFloat,
		Residuals:        residuals,
		RSquared:         rSquared,
		AdjustedRSquared: adjustedRsquared,
		StandardError:    standardError,
		TValue:           tValue,
		PValue:           pValue,
	}
}

// 新增 t 分布的累積分布函數
func tCDF(t float64, df int) float64 {
	x := float64(df) / (float64(df) + t*t)
	return betaInc(float64(df)/2.0, 0.5, x) / 2.0
}

// 新增不完全貝塔函數
func betaInc(a, b, x float64) float64 {
	if x < 0.0 || x > 1.0 {
		return 0.0
	}

	bt := math.Exp(lgamma(a+b) - lgamma(a) - lgamma(b) + a*math.Log(x) + b*math.Log(1.0-x))

	if x < (a+1.0)/(a+b+2.0) {
		return bt * betaCF(a, b, x) / a
	}

	return 1.0 - bt*betaCF(b, a, 1.0-x)/b
}

// 新增連分數展開
func betaCF(a, b, x float64) float64 {
	const MAXIT = 200
	const EPS = 3.0e-7
	const FPMIN = 1.0e-30

	qab := a + b
	qap := a + 1.0
	qam := a - 1.0
	c := 1.0
	d := 1.0 - qab*x/qap

	if math.Abs(d) < FPMIN {
		d = FPMIN
	}
	d = 1.0 / d
	h := d

	for m := 1; m <= MAXIT; m++ {
		m2 := 2 * m
		aa := float64(m) * (b - float64(m)) * x / ((qam + float64(m2)) * (a + float64(m2)))
		d = 1.0 + aa*d
		if math.Abs(d) < FPMIN {
			d = FPMIN
		}
		c = 1.0 + aa/c
		if math.Abs(c) < FPMIN {
			c = FPMIN
		}
		d = 1.0 / d
		h *= d * c
		aa = -(a + float64(m)) * (qab + float64(m)) * x / ((a + float64(m2)) * (qap + float64(m2)))
		d = 1.0 + aa*d
		if math.Abs(d) < FPMIN {
			d = FPMIN
		}
		c = 1.0 + aa/c
		if math.Abs(c) < FPMIN {
			c = FPMIN
		}
		d = 1.0 / d
		del := d * c
		h *= del
		if math.Abs(del-1.0) < EPS {
			break
		}
	}
	return h
}

// 新增對數伽瑪函數
func lgamma(x float64) float64 {
	return math.Log(math.Gamma(x))
}
