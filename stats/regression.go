package stats

import (
	"errors"
	"fmt"
	"math"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats/internal/parutil"
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

	// CovType is the covariance the inference fields come from.
	CovType CovType
	// Lag is the number of autocovariances CovHAC used, after any automatic
	// choice; it is 0 for the other CovTypes.
	Lag int
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

func prepareRegressionPrediction(model string, typ PredictType, predictorCount int, newXs []insyra.IDataList) ([][]float64, int, error) {
	if typ == "" {
		typ = PredictResponse
	}
	if typ != PredictResponse {
		return nil, 0, fmt.Errorf("unsupported predict type %q for %s regression", typ, model)
	}
	if len(newXs) != predictorCount {
		return nil, 0, fmt.Errorf("expected %d predictors, got %d", predictorCount, len(newXs))
	}
	return gatherPredictorInputs(newXs)
}

// Predict returns response-scale point predictions for new observations.
func (r *LinearRegressionResult) Predict(typ PredictType, newXs ...insyra.IDataList) (*insyra.DataList, error) {
	if r == nil {
		return nil, errors.New("linear regression result is nil")
	}
	if len(r.Coefficients) == 0 {
		return nil, errors.New("no coefficients available")
	}
	xs, n, err := prepareRegressionPrediction("linear", typ, len(r.Coefficients)-1, newXs)
	if err != nil {
		return nil, err
	}
	out := make([]any, n)
	for i := range n {
		value := r.Coefficients[0]
		for j := range xs {
			value += r.Coefficients[j+1] * xs[j][i]
		}
		out[i] = value
	}
	return insyra.NewDataList(out), nil
}

// Predict returns response-scale point predictions for new observations.
func (r *PolynomialRegressionResult) Predict(typ PredictType, newXs ...insyra.IDataList) (*insyra.DataList, error) {
	if r == nil {
		return nil, errors.New("polynomial regression result is nil")
	}
	if len(r.Coefficients) == 0 {
		return nil, errors.New("no coefficients available")
	}
	xs, n, err := prepareRegressionPrediction("polynomial", typ, 1, newXs)
	if err != nil {
		return nil, err
	}
	out := make([]any, n)
	for i := range n {
		value := 0.0
		power := 1.0
		for _, coefficient := range r.Coefficients {
			value += coefficient * power
			power *= xs[0][i]
		}
		out[i] = value
	}
	return insyra.NewDataList(out), nil
}

// Predict returns response-scale point predictions for new observations.
func (r *ExponentialRegressionResult) Predict(typ PredictType, newXs ...insyra.IDataList) (*insyra.DataList, error) {
	if r == nil {
		return nil, errors.New("exponential regression result is nil")
	}
	xs, n, err := prepareRegressionPrediction("exponential", typ, 1, newXs)
	if err != nil {
		return nil, err
	}
	out := make([]any, n)
	for i := range n {
		out[i] = r.Intercept * math.Exp(r.Slope*xs[0][i])
	}
	return insyra.NewDataList(out), nil
}

// Predict returns response-scale point predictions for new observations.
func (r *LogarithmicRegressionResult) Predict(typ PredictType, newXs ...insyra.IDataList) (*insyra.DataList, error) {
	if r == nil {
		return nil, errors.New("logarithmic regression result is nil")
	}
	xs, n, err := prepareRegressionPrediction("logarithmic", typ, 1, newXs)
	if err != nil {
		return nil, err
	}
	out := make([]any, n)
	for i, x := range xs[0] {
		if x <= 0 {
			return nil, fmt.Errorf("x must be > 0 for ln prediction (row index %d)", i)
		}
		out[i] = r.Intercept + r.Slope*math.Log(x)
	}
	return insyra.NewDataList(out), nil
}

// CovType chooses the covariance an ordinary least-squares fit reports its
// standard errors, t values, p values and confidence intervals from.
type CovType int

const (
	// CovOLS is the classical covariance s²(XᵀX)⁻¹, which assumes
	// homoskedastic, uncorrelated errors. It is the zero value.
	CovOLS CovType = iota
	// CovHC0 is White's heteroskedasticity-consistent covariance.
	CovHC0
	// CovHC1 is CovHC0 scaled by n/(n-k).
	CovHC1
	// CovHC2 divides each squared residual by 1-h, h the observation's leverage.
	CovHC2
	// CovHC3 divides each squared residual by (1-h)².
	CovHC3
	// CovHAC is the Newey-West heteroskedasticity-and-autocorrelation-consistent
	// covariance with Bartlett weights 1 - j/(lag+1) and no small-sample factor.
	CovHAC
)

// LinearRegressionOptions are the settings of LinearRegressionWithOptions. The
// zero value is LinearRegression.
type LinearRegressionOptions struct {
	// CovType chooses the covariance. The default is CovOLS.
	CovType CovType
	// Lag is the number of autocovariances CovHAC includes. A negative lag asks
	// for NeweyWestLag(n); 0 makes CovHAC equal CovHC0. It must be smaller than
	// the number of observations. Other CovTypes ignore it.
	Lag int
}

// LinearRegression performs ordinary least-squares linear regression with the
// classical standard errors. It is LinearRegressionWithOptions with zero
// options.
func LinearRegression(dlY insyra.IDataList, dlXs ...insyra.IDataList) (*LinearRegressionResult, error) {
	return LinearRegressionWithOptions(LinearRegressionOptions{}, dlY, dlXs...)
}

// LinearRegressionWithOptions performs ordinary least-squares linear
// regression and reports inference from the covariance opts.CovType names.
// The coefficients, residuals and R² do not depend on it.
//
// CovHC0 to CovHC3 are the estimators of R's sandwich::vcovHC. CovHAC is
// (XᵀX)⁻¹ S (XᵀX)⁻¹ with S the Bartlett-weighted long-run covariance of the
// scores, as sandwich::NeweyWest(lag = L, prewhite = FALSE, adjust = FALSE)
// and statsmodels' cov_type="HAC" with use_correction=False compute it.
//
// Every t value, p value and confidence interval uses the t distribution with
// n-k degrees of freedom, as lmtest::coeftest reports them and as
// NeweyWestMeanTest does; statsmodels reads robust p-values from the normal
// distribution unless use_t=True.
func LinearRegressionWithOptions(opts LinearRegressionOptions, dlY insyra.IDataList, dlXs ...insyra.IDataList) (*LinearRegressionResult, error) {
	if opts.CovType < CovOLS || opts.CovType > CovHAC {
		return nil, fmt.Errorf("unsupported covariance type %v", opts.CovType)
	}
	p := len(dlXs)
	if p == 0 {
		return nil, errors.New("no independent variables provided")
	}

	// Pull y and every xⱼ in parallel — each list is a separate actor and
	// can be entered independently. The previous serial form paid one
	// actor handshake per predictor sequentially; for p large this was
	// the wall-clock bottleneck.
	ys, xSlices, _, n, err := gatherRegressionInputs(dlY, dlXs)
	if err != nil {
		return nil, err
	}

	if n <= p+1 {
		return nil, errors.New("need at least p+2 observations for p independent variables to compute statistics")
	}

	// Design matrix X (n × (p+1)) with leading intercept column.
	//
	// Direct buffer write via RawMatrix().Data is row-major contiguous,
	// so for each row we touch (p+1) consecutive float64 slots — friendly
	// to the cache. mat.Dense.Set goes through a method-call + bounds-
	// check path that's significantly slower in tight loops; for large
	// (n, p) the design-matrix fill was a measurable fraction of the
	// regression call.
	//
	// Row writes are independent so we parallelise over rows when
	// n·(p+1) is large enough to amortise the goroutine launch.
	X := buildDesignMatrix(xSlices, n)
	xRaw := X.RawMatrix()
	stride := xRaw.Stride

	// Solve β = (XᵀX)⁻¹ Xᵀy and obtain (XᵀX)⁻¹ via gonum's LU-based solver
	// (replaces the previous hand-rolled Gauss-Jordan in stats/internal/linalg).
	coeffs, XTXInv := solveOLS(X, ys)
	if coeffs == nil {
		return nil, errors.New("matrix is singular, cannot solve")
	}

	df := float64(n - p - 1)
	// computeGoodnessOfFit calls predict(i) once per row. Per-row work is
	// (p+1) X.At lookups; switching to direct row-major buffer indexing
	// avoids that many method calls + bounds checks. xRaw / stride were
	// captured above for the parallel design-matrix fill.
	residuals, rSquared, adjRSquared, sse, ok := computeGoodnessOfFit(ys, func(i int) float64 {
		yHat := 0.0
		base := i * stride
		for j := 0; j <= p; j++ {
			yHat += coeffs[j] * xRaw.Data[base+j]
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
	lag := 0
	var standardErrors, tValues, pValues []float64
	switch opts.CovType {
	case CovOLS:
		standardErrors, tValues, pValues = computeCoeffInference(coeffs, XTXInv, mse, df)
	default:
		leveragePower, meatScale := 0.0, 1.0
		switch opts.CovType {
		case CovHC1:
			meatScale = float64(n) / df
		case CovHC2:
			leveragePower = 0.5
		case CovHC3:
			leveragePower = 1
		case CovHAC:
			lag = opts.Lag
			if lag < 0 {
				lag = NeweyWestLag(n)
			}
			if lag >= n {
				return nil, fmt.Errorf("lag %d must be smaller than the number of observations %d", lag, n)
			}
		}
		variances := olsSandwichVariances(X, residuals, XTXInv, leveragePower, meatScale, lag)
		standardErrors, tValues, pValues = coeffInferenceFromVariances(coeffs, variances, df)
	}

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
		CovType:                 opts.CovType,
		Lag:                     lag,
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
	var xsRaw, ysRaw []any
	isFailed := false
	dlx := asDataList(dlX)
	dly := asDataList(dlY)
	insyra.AtomicDoAll(func() {
		if dlx.Len() != dly.Len() || dlx.Len() == 0 {
			isFailed = true
			return
		}
		if dlx.Len() <= 2 {
			isFailed = true
			return
		}
		xsRaw = dlx.Data()
		ysRaw = dly.Data()
	}, dlx, dly)
	if isFailed {
		return nil, errors.New("input lengths mismatch/zero, or need at least 3 observations")
	}
	var convErr error
	if xs, convErr = numericValues(xsRaw, "x"); convErr != nil {
		return nil, convErr
	}
	if ys, convErr = numericValues(ysRaw, "y"); convErr != nil {
		return nil, convErr
	}

	logYs := make([]float64, len(ys))
	for i := range ys {
		if ys[i] <= 0 {
			return nil, fmt.Errorf("y must be > 0 for log computation (row index %d)", i)
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
	var xsRaw, ysRaw []any
	isFailed := false
	dlx := asDataList(dlX)
	dly := asDataList(dlY)
	insyra.AtomicDoAll(func() {
		if dlx.Len() != dly.Len() || dlx.Len() == 0 {
			isFailed = true
			return
		}
		if dlx.Len() <= 2 {
			isFailed = true
			return
		}
		xsRaw = dlx.Data()
		ysRaw = dly.Data()
	}, dlx, dly)
	if isFailed {
		return nil, errors.New("input lengths mismatch/zero, or need at least 3 observations")
	}
	var convErr error
	if xs, convErr = numericValues(xsRaw, "x"); convErr != nil {
		return nil, convErr
	}
	if ys, convErr = numericValues(ysRaw, "y"); convErr != nil {
		return nil, convErr
	}

	logXs := make([]float64, len(xs))
	for i := range xs {
		if xs[i] <= 0 {
			return nil, fmt.Errorf("x must be > 0 for ln computation (row index %d)", i)
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
	var xsRaw, ysRaw []any
	isFailed := false
	dlx := asDataList(dlX)
	dly := asDataList(dlY)
	insyra.AtomicDoAll(func() {
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

		xsRaw = dlx.Data()
		ysRaw = dly.Data()
	}, dlx, dly)
	if isFailed {
		return nil, errors.New("invalid input lengths, degree, or insufficient observations")
	}
	var convErr error
	if xs, convErr = numericValues(xsRaw, "x"); convErr != nil {
		return nil, convErr
	}
	if ys, convErr = numericValues(ysRaw, "y"); convErr != nil {
		return nil, convErr
	}

	n := len(xs)

	// Design matrix [1, x, x², …, x^degree]. Same direct-buffer +
	// row-parallel pattern as LinearRegression. Each x^j is computed by
	// repeated multiplication, so we walk row-major and never read across
	// rows.
	X := mat.NewDense(n, degree+1, nil)
	xRaw := X.RawMatrix()
	stride := xRaw.Stride
	parutil.Run(n, n*(degree+1) >= 50_000, func(i int) {
		base := i * stride
		xRaw.Data[base] = 1.0
		xi := xs[i]
		for j := 1; j <= degree; j++ {
			xRaw.Data[base+j] = xRaw.Data[base+j-1] * xi
		}
	})

	coeffs, XTXInv := solveOLS(X, ys)
	if coeffs == nil {
		return nil, errors.New("matrix is singular, cannot solve")
	}

	df := float64(n - degree - 1)
	// Direct buffer access — see LinearRegression for the rationale.
	residuals, r2, adjR2, sse, fitOK := computeGoodnessOfFit(ys, func(i int) float64 {
		yHat := 0.0
		base := i * stride
		for j := 0; j <= degree; j++ {
			yHat += coeffs[j] * xRaw.Data[base+j]
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
