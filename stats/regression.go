package stats

import (
	"errors"
	"fmt"
	"math"

	"github.com/HazelnutParadise/insyra"
	"gonum.org/v1/gonum/mat"
)

// solveOLS computes β = (XᵀX)⁻¹ Xᵀy and returns β plus (XᵀX)⁻¹ as a [][]float64
// (so the surrounding helpers — computeCoeffInference / buildMultiCoeffCIs —
// can stay slice-shaped and unchanged).
//
// X has shape (n, k); y has length n. Returns (nil, nil) if XᵀX is singular.
func solveOLS(X *mat.Dense, y []float64) (beta []float64, xtxInv [][]float64) {
	_, k := X.Dims()
	yVec := mat.NewVecDense(len(y), y)

	var xtx mat.Dense
	xtx.Mul(X.T(), X)

	var xty mat.VecDense
	xty.MulVec(X.T(), yVec)

	var betaVec mat.VecDense
	if err := betaVec.SolveVec(&xtx, &xty); err != nil {
		return nil, nil
	}

	var xtxInvDense mat.Dense
	if err := xtxInvDense.Inverse(&xtx); err != nil {
		return nil, nil
	}

	beta = make([]float64, k)
	for i := range k {
		beta[i] = betaVec.AtVec(i)
	}
	xtxInv = make([][]float64, k)
	for i := range k {
		xtxInv[i] = make([]float64, k)
		for j := range k {
			xtxInv[i][j] = xtxInvDense.At(i, j)
		}
	}
	return beta, xtxInv
}

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
func LinearRegression(dlY insyra.IDataList, dlXs ...insyra.IDataList) (*LinearRegressionResult, error) {
	var n int
	var ys []float64
	dlY.AtomicDo(func(dly *insyra.DataList) {
		n = dly.Len()
		ys = dly.ToF64Slice()
	})
	p := len(dlXs)

	if p == 0 {
		return nil, errors.New("no independent variables provided")
	}

	xSlices := make([][]float64, p)
	for j, dlX := range dlXs {
		isFailed := false
		dlX.AtomicDo(func(l *insyra.DataList) {
			if l.Len() != n {
				isFailed = true
				return
			}
			xSlices[j] = l.ToF64Slice()
		})
		if isFailed {
			return nil, fmt.Errorf("x and y must have the same length for predictor %d", j)
		}
	}

	if n <= p+1 {
		return nil, errors.New("need at least p+2 observations for p independent variables to compute statistics")
	}

	// Design matrix X (n × (p+1)) with leading intercept column.
	X := mat.NewDense(n, p+1, nil)
	for i := range n {
		X.Set(i, 0, 1.0)
		for j := range p {
			X.Set(i, j+1, xSlices[j][i])
		}
	}

	// Solve β = (XᵀX)⁻¹ Xᵀy and obtain (XᵀX)⁻¹ via gonum's LU-based solver
	// (replaces the previous hand-rolled Gauss-Jordan in stats/internal/linalg).
	coeffs, XTXInv := solveOLS(X, ys)
	if coeffs == nil {
		return nil, errors.New("matrix is singular, cannot solve")
	}

	df := float64(n - p - 1)
	residuals, rSquared, adjRSquared, sse, ok := computeGoodnessOfFit(ys, func(i int) float64 {
		yHat := 0.0
		for j := 0; j <= p; j++ {
			yHat += coeffs[j] * X.At(i, j)
		}
		return yHat
	}, df)
	if !ok {
		return nil, errors.New("variance of y is zero; R-squared undefined")
	}

	mse := math.NaN()
	if df > 0 {
		mse = sse / df
	}
	standardErrors, tValues, pValues := computeCoeffInference(coeffs, XTXInv, mse, df)

	result := &LinearRegressionResult{
		Slope:                   math.NaN(),
		Intercept:               coeffs[0],
		StandardError:           math.NaN(),
		StandardErrorIntercept:  standardErrors[0],
		TValue:                  math.NaN(),
		TValueIntercept:         tValues[0],
		PValue:                  math.NaN(),
		PValueIntercept:         pValues[0],
		ConfidenceIntervalSlope: nanCI(),
		Residuals:               residuals,
		RSquared:                rSquared,
		AdjustedRSquared:        adjRSquared,
		Coefficients:            coeffs,
		StandardErrors:          standardErrors,
		TValues:                 tValues,
		PValues:                 pValues,
	}

	if p == 1 {
		result.Slope = coeffs[1]
		result.StandardError = standardErrors[1]
		result.TValue = tValues[1]
		result.PValue = pValues[1]
	}

	confIntervals := buildMultiCoeffCIs(coeffs, standardErrors, df)
	result.ConfidenceIntervals = confIntervals

	result.ConfidenceIntervalIntercept = confIntervals[0]
	if p == 1 {
		result.ConfidenceIntervalSlope = confIntervals[1]
	}

	return result, nil
}

// ExponentialRegression performs y = a*e^(b*x) regression.
func ExponentialRegression(dlY, dlX insyra.IDataList) (*ExponentialRegressionResult, error) {
	var xs, ys []float64
	isFailed := false
	dlX.AtomicDo(func(dlx *insyra.DataList) {
		dlY.AtomicDo(func(dly *insyra.DataList) {
			if dlx.Len() != dly.Len() || dlx.Len() == 0 {
				isFailed = true
				return
			}
			if dlx.Len() <= 2 {
				isFailed = true
				return
			}
			xs = dlx.ToF64Slice()
			ys = dly.ToF64Slice()
		})
	})
	if isFailed {
		return nil, errors.New("input lengths mismatch/zero, or need at least 3 observations")
	}

	logYs := make([]float64, len(ys))
	for i := range ys {
		if ys[i] <= 0 {
			return nil, fmt.Errorf("y must be > 0 for log computation (index %d)", i)
		}
		logYs[i] = math.Log(ys[i])
	}

	lnA, b, ok := simpleOLSCoeffs(xs, logYs)
	if !ok {
		return nil, errors.New("denominator zero, cannot compute coefficients")
	}
	a := math.Exp(lnA)

	n := float64(len(xs))
	df := n - 2
	residuals, rSquared, adjRSquared, _, fitOK := computeGoodnessOfFit(ys, func(i int) float64 {
		return a * math.Exp(b*xs[i])
	}, df)
	if !fitOK {
		return nil, errors.New("variance of y is zero; R-squared undefined")
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
	}, nil
}

// LogarithmicRegression performs y = a + b*ln(x) regression.
func LogarithmicRegression(dlY, dlX insyra.IDataList) (*LogarithmicRegressionResult, error) {
	var xs, ys []float64
	isFailed := false
	dlX.AtomicDo(func(dlx *insyra.DataList) {
		dlY.AtomicDo(func(dly *insyra.DataList) {
			if dlx.Len() != dly.Len() || dlx.Len() == 0 {
				isFailed = true
				return
			}
			if dlx.Len() <= 2 {
				isFailed = true
				return
			}
			xs = dlx.ToF64Slice()
			ys = dly.ToF64Slice()
		})
	})
	if isFailed {
		return nil, errors.New("input lengths mismatch/zero, or need at least 3 observations")
	}

	logXs := make([]float64, len(xs))
	for i := range xs {
		if xs[i] <= 0 {
			return nil, fmt.Errorf("x must be > 0 for ln computation (index %d)", i)
		}
		logXs[i] = math.Log(xs[i])
	}

	a, b, ok := simpleOLSCoeffs(logXs, ys)
	if !ok {
		return nil, errors.New("denominator zero, cannot compute coefficients")
	}

	n := float64(len(xs))
	df := n - 2
	residuals, r2, adjR2, sse, fitOK := computeGoodnessOfFit(ys, func(i int) float64 {
		return a + b*logXs[i]
	}, df)
	if !fitOK {
		return nil, errors.New("variance of y is zero; R-squared undefined")
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
	}, nil
}

// PolynomialRegression performs polynomial regression of given degree.
func PolynomialRegression(dlY, dlX insyra.IDataList, degree int) (*PolynomialRegressionResult, error) {
	var xs, ys []float64
	isFailed := false
	dlX.AtomicDo(func(dlx *insyra.DataList) {
		dlY.AtomicDo(func(dly *insyra.DataList) {
			if dlx.Len() != dly.Len() || dlx.Len() == 0 {
				isFailed = true
				return
			}
			if degree < 1 || degree >= dlx.Len() {
				isFailed = true
				return
			}
			if dlx.Len() <= degree+1 {
				isFailed = true
				return
			}

			xs = dlx.ToF64Slice()
			ys = dly.ToF64Slice()
		})
	})
	if isFailed {
		return nil, errors.New("invalid input lengths, degree, or insufficient observations")
	}

	n := len(xs)

	// Design matrix [1, x, x², …, x^degree]
	X := mat.NewDense(n, degree+1, nil)
	for i := range n {
		X.Set(i, 0, 1.0)
		for j := 1; j <= degree; j++ {
			X.Set(i, j, X.At(i, j-1)*xs[i])
		}
	}

	coeffs, XTXInv := solveOLS(X, ys)
	if coeffs == nil {
		return nil, errors.New("matrix is singular, cannot solve")
	}

	df := float64(n - degree - 1)
	residuals, r2, adjR2, sse, fitOK := computeGoodnessOfFit(ys, func(i int) float64 {
		yHat := 0.0
		for j := 0; j <= degree; j++ {
			yHat += coeffs[j] * X.At(i, j)
		}
		return yHat
	}, df)
	if !fitOK {
		return nil, errors.New("variance of y is zero; R-squared undefined")
	}

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
	}, nil
}
