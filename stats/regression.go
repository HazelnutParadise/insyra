package stats

import (
	"math"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats/internal/linalg"
)

// LinearRegressionResult holds the result of both simple and multiple linear regression.
// For simple regression: Coefficients[0] = intercept, Coefficients[1] = slope.
// For multiple regression: Coefficients[0] = intercept, Coefficients[1:] = slopes.
type LinearRegressionResult struct {
	Slope                  float64
	Intercept              float64
	StandardError          float64
	StandardErrorIntercept float64
	TValue                 float64
	TValueIntercept        float64
	PValue                 float64
	PValueIntercept        float64

	ConfidenceIntervalIntercept [2]float64
	ConfidenceIntervalSlope     [2]float64

	Coefficients        []float64
	StandardErrors      []float64
	TValues             []float64
	PValues             []float64
	ConfidenceIntervals [][2]float64

	Residuals        []float64
	RSquared         float64
	AdjustedRSquared float64
}

// PolynomialRegressionResult holds the result of polynomial regression.
type PolynomialRegressionResult struct {
	Coefficients        []float64
	Degree              int
	Residuals           []float64
	RSquared            float64
	AdjustedRSquared    float64
	StandardErrors      []float64
	TValues             []float64
	PValues             []float64
	ConfidenceIntervals [][2]float64
}

// ExponentialRegressionResult holds the result of exponential regression y = a*e^(b*x).
type ExponentialRegressionResult struct {
	Intercept              float64
	Slope                  float64
	Residuals              []float64
	RSquared               float64
	AdjustedRSquared       float64
	StandardErrorIntercept float64
	StandardErrorSlope     float64
	TValueIntercept        float64
	TValueSlope            float64
	PValueIntercept        float64
	PValueSlope            float64

	ConfidenceIntervalIntercept [2]float64
	ConfidenceIntervalSlope     [2]float64
}

// LogarithmicRegressionResult holds the result of logarithmic regression y = a + b*ln(x).
type LogarithmicRegressionResult struct {
	Intercept              float64
	Slope                  float64
	Residuals              []float64
	RSquared               float64
	AdjustedRSquared       float64
	StandardErrorIntercept float64
	StandardErrorSlope     float64
	TValueIntercept        float64
	TValueSlope            float64
	PValueIntercept        float64
	PValueSlope            float64

	ConfidenceIntervalIntercept [2]float64
	ConfidenceIntervalSlope     [2]float64
}

// LinearRegression performs ordinary least-squares linear regression.
func LinearRegression(dlY insyra.IDataList, dlXs ...insyra.IDataList) *LinearRegressionResult {
	var n int
	var ys []float64
	dlY.AtomicDo(func(dly *insyra.DataList) {
		n = dly.Len()
		ys = dly.ToF64Slice()
	})
	p := len(dlXs)

	if p == 0 {
		insyra.LogWarning("stats", "LinearRegression", "no independent variables provided")
		return nil
	}

	xSlices := make([][]float64, p)
	for j, dlX := range dlXs {
		isFailed := false
		dlX.AtomicDo(func(l *insyra.DataList) {
			if l.Len() != n {
				insyra.LogWarning("stats", "LinearRegression", "x and y must have the same length")
				isFailed = true
				return
			}
			xSlices[j] = l.ToF64Slice()
		})
		if isFailed {
			return nil
		}
	}

	if n <= p+1 {
		insyra.LogWarning("stats", "LinearRegression", "need at least p+2 observations for p independent variables to compute statistics")
		return nil
	}

	X := make([][]float64, n)
	for i := range n {
		X[i] = make([]float64, p+1)
		X[i][0] = 1.0
		for j := range p {
			X[i][j+1] = xSlices[j][i]
		}
	}

	XTX := make([][]float64, p+1)
	XTy := make([]float64, p+1)
	for i := 0; i <= p; i++ {
		XTX[i] = make([]float64, p+1)
		for j := 0; j <= p; j++ {
			for k := range n {
				XTX[i][j] += X[k][i] * X[k][j]
			}
		}
		for k := range n {
			XTy[i] += X[k][i] * ys[k]
		}
	}

	coeffs := linalg.GaussianElimination(XTX, XTy)
	if coeffs == nil {
		insyra.LogWarning("stats", "LinearRegression", "matrix is singular, cannot solve")
		return nil
	}

	df := float64(n - p - 1)
	residuals, rSquared, adjRSquared, sse, ok := computeGoodnessOfFit(ys, func(i int) float64 {
		yHat := 0.0
		for j := 0; j <= p; j++ {
			yHat += coeffs[j] * X[i][j]
		}
		return yHat
	}, df)
	if !ok {
		insyra.LogWarning("stats", "LinearRegression", "variance of y is zero; R² undefined")
		return nil
	}

	XTXInv := linalg.InvertMatrix(XTX)
	mse := math.NaN()
	if df > 0 {
		mse = sse / df
	}
	standardErrors, tValues, pValues := computeCoeffInference(coeffs, XTXInv, mse, df)

	result := &LinearRegressionResult{
		Residuals:        residuals,
		RSquared:         rSquared,
		AdjustedRSquared: adjRSquared,
		Coefficients:     coeffs,
		StandardErrors:   standardErrors,
		TValues:          tValues,
		PValues:          pValues,
	}

	if p == 1 {
		result.Intercept = coeffs[0]
		result.Slope = coeffs[1]
		result.StandardErrorIntercept = standardErrors[0]
		result.StandardError = standardErrors[1]
		result.TValueIntercept = tValues[0]
		result.TValue = tValues[1]
		result.PValueIntercept = pValues[0]
		result.PValue = pValues[1]
	}

	confIntervals := buildMultiCoeffCIs(coeffs, standardErrors, df)
	if df <= 0 {
		insyra.LogWarning("stats", "LinearRegression", "insufficient degrees of freedom for confidence intervals")
	}
	result.ConfidenceIntervals = confIntervals

	if p == 1 {
		result.ConfidenceIntervalIntercept = confIntervals[0]
		result.ConfidenceIntervalSlope = confIntervals[1]
	}

	return result
}

// ExponentialRegression performs y = a*e^(b*x) regression.
func ExponentialRegression(dlY, dlX insyra.IDataList) *ExponentialRegressionResult {
	var xs, ys []float64
	isFailed := false
	dlX.AtomicDo(func(dlx *insyra.DataList) {
		dlY.AtomicDo(func(dly *insyra.DataList) {
			if dlx.Len() != dly.Len() || dlx.Len() == 0 {
				insyra.LogWarning("stats", "ExponentialRegression", "input lengths mismatch or zero")
				isFailed = true
				return
			}
			if dlx.Len() <= 2 {
				insyra.LogWarning("stats", "ExponentialRegression", "need at least 3 observations for exponential regression")
				isFailed = true
				return
			}
			xs = dlx.ToF64Slice()
			ys = dly.ToF64Slice()
		})
	})
	if isFailed {
		return nil
	}

	logYs := make([]float64, len(ys))
	for i := range ys {
		if ys[i] <= 0 {
			insyra.LogWarning("stats", "ExponentialRegression", "y must be > 0 for log computation")
			return nil
		}
		logYs[i] = math.Log(ys[i])
	}

	lnA, b, ok := simpleOLSCoeffs(xs, logYs)
	if !ok {
		insyra.LogWarning("stats", "ExponentialRegression", "denominator zero, cannot compute coefficients")
		return nil
	}
	a := math.Exp(lnA)

	n := float64(len(xs))
	df := n - 2
	residuals, rSquared, adjRSquared, _, fitOK := computeGoodnessOfFit(ys, func(i int) float64 {
		return a * math.Exp(b*xs[i])
	}, df)
	if !fitOK {
		insyra.LogWarning("stats", "ExponentialRegression", "variance of y is zero; R² undefined")
		return nil
	}

	sumX := 0.0
	for _, x := range xs {
		sumX += x
	}
	meanX := sumX / n

	var sseLog float64
	for i := range xs {
		yHatLog := lnA + b*xs[i]
		residLog := logYs[i] - yHatLog
		sseLog += residLog * residLog
	}
	mseLog := sseLog / df

	var sumXMinusMeanXSquared float64
	for i := range xs {
		d := xs[i] - meanX
		sumXMinusMeanXSquared += d * d
	}

	seB := math.Sqrt(mseLog / sumXMinusMeanXSquared)
	seLnA := math.Sqrt(mseLog * (1.0/n + meanX*meanX/sumXMinusMeanXSquared))
	seA := a * seLnA

	tValA, tValB, pValA, pValB, ciIntercept, ciSlope := inferTwoCoeffStats(a, b, seA, seB, df)

	return &ExponentialRegressionResult{
		Intercept:                   a,
		Slope:                       b,
		Residuals:                   residuals,
		RSquared:                    rSquared,
		AdjustedRSquared:            adjRSquared,
		StandardErrorIntercept:      seA,
		StandardErrorSlope:          seB,
		TValueIntercept:             tValA,
		TValueSlope:                 tValB,
		PValueIntercept:             pValA,
		PValueSlope:                 pValB,
		ConfidenceIntervalIntercept: ciIntercept,
		ConfidenceIntervalSlope:     ciSlope,
	}
}

// LogarithmicRegression performs y = a + b*ln(x) regression.
func LogarithmicRegression(dlY, dlX insyra.IDataList) *LogarithmicRegressionResult {
	var xs, ys []float64
	isFailed := false
	dlX.AtomicDo(func(dlx *insyra.DataList) {
		dlY.AtomicDo(func(dly *insyra.DataList) {
			if dlx.Len() != dly.Len() || dlx.Len() == 0 {
				insyra.LogWarning("stats", "LogarithmicRegression", "input lengths mismatch or zero")
				isFailed = true
				return
			}
			if dlx.Len() <= 2 {
				insyra.LogWarning("stats", "LogarithmicRegression", "need at least 3 observations for logarithmic regression")
				isFailed = true
				return
			}
			xs = dlx.ToF64Slice()
			ys = dly.ToF64Slice()
		})
	})
	if isFailed {
		return nil
	}

	logXs := make([]float64, len(xs))
	for i := range xs {
		if xs[i] <= 0 {
			insyra.LogWarning("stats", "LogarithmicRegression", "x must be > 0 for ln computation")
			return nil
		}
		logXs[i] = math.Log(xs[i])
	}

	a, b, ok := simpleOLSCoeffs(logXs, ys)
	if !ok {
		insyra.LogWarning("stats", "LogarithmicRegression", "denominator zero, cannot compute coefficients")
		return nil
	}

	n := float64(len(xs))
	df := n - 2
	residuals, r2, adjR2, sse, fitOK := computeGoodnessOfFit(ys, func(i int) float64 {
		return a + b*logXs[i]
	}, df)
	if !fitOK {
		insyra.LogWarning("stats", "LogarithmicRegression", "variance of y is zero; R² undefined")
		return nil
	}

	sumLX := 0.0
	for _, lx := range logXs {
		sumLX += lx
	}
	meanLX := sumLX / n
	sxxLX := simpleOLSSxx(logXs)

	mse := sse / df
	seB := math.Sqrt(mse / sxxLX)
	seA := math.Sqrt(mse * (1.0/n + meanLX*meanLX/sxxLX))
	tValA, tValB, pValA, pValB, ciIntercept, ciSlope := inferTwoCoeffStats(a, b, seA, seB, df)

	return &LogarithmicRegressionResult{
		Intercept:                   a,
		Slope:                       b,
		Residuals:                   residuals,
		RSquared:                    r2,
		AdjustedRSquared:            adjR2,
		StandardErrorIntercept:      seA,
		StandardErrorSlope:          seB,
		TValueIntercept:             tValA,
		TValueSlope:                 tValB,
		PValueIntercept:             pValA,
		PValueSlope:                 pValB,
		ConfidenceIntervalIntercept: ciIntercept,
		ConfidenceIntervalSlope:     ciSlope,
	}
}

// PolynomialRegression performs polynomial regression of given degree.
func PolynomialRegression(dlY, dlX insyra.IDataList, degree int) *PolynomialRegressionResult {
	var xs, ys []float64
	isFailed := false
	dlX.AtomicDo(func(dlx *insyra.DataList) {
		dlY.AtomicDo(func(dly *insyra.DataList) {
			if dlx.Len() != dly.Len() || dlx.Len() == 0 {
				insyra.LogWarning("stats", "PolynomialRegression", "input lengths mismatch or zero")
				isFailed = true
				return
			}
			if degree < 1 || degree >= dlx.Len() {
				insyra.LogWarning("stats", "PolynomialRegression", "invalid degree")
				isFailed = true
				return
			}
			if dlx.Len() <= degree+1 {
				insyra.LogWarning("stats", "PolynomialRegression", "need at least degree+2 observations for polynomial regression")
				isFailed = true
				return
			}

			xs = dlx.ToF64Slice()
			ys = dly.ToF64Slice()
		})
	})
	if isFailed {
		return nil
	}

	n := len(xs)

	X := make([][]float64, n)
	for i := range X {
		X[i] = make([]float64, degree+1)
		X[i][0] = 1.0
		for j := 1; j <= degree; j++ {
			X[i][j] = X[i][j-1] * xs[i]
		}
	}

	XTX := make([][]float64, degree+1)
	XTy := make([]float64, degree+1)
	for i := range XTX {
		XTX[i] = make([]float64, degree+1)
		for j := range XTX[i] {
			for k := 0; k < n; k++ {
				XTX[i][j] += X[k][i] * X[k][j]
			}
		}
	}
	for i := 0; i <= degree; i++ {
		for j := range n {
			XTy[i] += X[j][i] * ys[j]
		}
	}

	coeffs := linalg.GaussianElimination(XTX, XTy)
	if coeffs == nil {
		insyra.LogWarning("stats", "PolynomialRegression", "matrix is singular, cannot solve")
		return nil
	}

	df := float64(n - degree - 1)
	residuals, r2, adjR2, sse, fitOK := computeGoodnessOfFit(ys, func(i int) float64 {
		yHat := 0.0
		for j := 0; j <= degree; j++ {
			yHat += coeffs[j] * X[i][j]
		}
		return yHat
	}, df)
	if !fitOK {
		insyra.LogWarning("stats", "PolynomialRegression", "variance of y is zero; R² undefined")
		return nil
	}

	XTXInv := linalg.InvertMatrix(XTX)
	mse := math.NaN()
	if df > 0 {
		mse = sse / df
	}
	standardErrors, tValues, pValues := computeCoeffInference(coeffs, XTXInv, mse, df)
	confIntervals := buildMultiCoeffCIs(coeffs, standardErrors, df)

	return &PolynomialRegressionResult{
		Coefficients:        coeffs,
		Degree:              degree,
		Residuals:           residuals,
		RSquared:            r2,
		AdjustedRSquared:    adjR2,
		StandardErrors:      standardErrors,
		TValues:             tValues,
		PValues:             pValues,
		ConfidenceIntervals: confIntervals,
	}
}
