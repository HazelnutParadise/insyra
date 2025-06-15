package stats

import (
	"math"

	"github.com/HazelnutParadise/insyra"
)

// LinearRegressionResult holds the result of both simple and multiple linear regression.
// For simple regression: Coefficients[0] = intercept, Coefficients[1] = slope
// For multiple regression: Coefficients[0] = intercept, Coefficients[1:] = slopes
// Comments are kept in English per project convention.
type LinearRegressionResult struct {
	// Legacy fields for simple regression compatibility
	Slope                  float64 // regression coefficient β₁ (for simple regression)
	Intercept              float64 // regression coefficient β₀ (for simple regression)
	StandardError          float64 // SE(β₁) - slope standard error (for simple regression)
	StandardErrorIntercept float64 // SE(β₀) - intercept standard error (for simple regression)
	TValue                 float64 // t statistic for β₁ (for simple regression)
	TValueIntercept        float64 // t statistic for β₀ (for simple regression)
	PValue                 float64 // two-tailed p-value for β₁ (for simple regression)
	PValueIntercept        float64 // two-tailed p-value for β₀ (for simple regression)

	// Extended fields for multiple regression
	Coefficients   []float64 // [β₀, β₁, ..., βₚ] (intercept + slopes)
	StandardErrors []float64 // standard errors for each coefficient
	TValues        []float64 // t statistics for each coefficient
	PValues        []float64 // two-tailed p-values for each coefficient

	// Common fields
	Residuals        []float64 // yᵢ − ŷᵢ
	RSquared         float64   // coefficient of determination
	AdjustedRSquared float64   // adjusted R²
}

// PolynomialRegressionResult holds the result of polynomial regression.
type PolynomialRegressionResult struct {
	Coefficients     []float64 // polynomial coefficients [a₀, a₁, a₂, ...]
	Degree           int       // degree of polynomial
	Residuals        []float64 // yᵢ − ŷᵢ
	RSquared         float64   // coefficient of determination
	AdjustedRSquared float64   // adjusted R²
	StandardErrors   []float64 // standard errors for each coefficient
	TValues          []float64 // t statistics for each coefficient
	PValues          []float64 // p-values for each coefficient
}

// ExponentialRegressionResult holds the result of exponential regression y = a·e^(b·x).
type ExponentialRegressionResult struct {
	Intercept              float64   // coefficient a in y = a·e^(b·x)
	Slope                  float64   // coefficient b in y = a·e^(b·x)
	Residuals              []float64 // yᵢ − ŷᵢ
	RSquared               float64   // coefficient of determination
	AdjustedRSquared       float64   // adjusted R²
	StandardErrorIntercept float64   // standard error of coefficient a
	StandardErrorSlope     float64   // standard error of coefficient b
	TValueIntercept        float64   // t statistic for coefficient a
	TValueSlope            float64   // t statistic for coefficient b
	PValueIntercept        float64   // p-value for coefficient a
	PValueSlope            float64   // p-value for coefficient b
}

// LogarithmicRegressionResult holds the result of logarithmic regression y = a + b·ln(x).
type LogarithmicRegressionResult struct {
	Intercept              float64   // intercept coefficient in y = a + b·ln(x)
	Slope                  float64   // slope coefficient in y = a + b·ln(x)
	Residuals              []float64 // yᵢ − ŷᵢ
	RSquared               float64   // coefficient of determination
	AdjustedRSquared       float64   // adjusted R²
	StandardErrorIntercept float64   // standard error of coefficient a
	StandardErrorSlope     float64   // standard error of coefficient b
	TValueIntercept        float64   // t statistic for coefficient a
	TValueSlope            float64   // t statistic for coefficient b
	PValueIntercept        float64   // p-value for coefficient a
	PValueSlope            float64   // p-value for coefficient b
}

// LinearRegression performs ordinary least-squares linear regression.
// Supports both simple (one X) and multiple (multiple X) linear regression.
// dlY is dependent variable, dlXs are independent variables (variadic).
//
// ** Verified using R **
func LinearRegression(dlY insyra.IDataList, dlXs ...insyra.IDataList) *LinearRegressionResult {
	// --- sanity checks ------------------------------------------------------
	n := dlY.Len()
	p := len(dlXs)

	if p == 0 {
		insyra.LogWarning("stats", "LinearRegression", "no independent variables provided")
		return nil
	}

	for _, dlX := range dlXs {
		if dlX.Len() != n {
			insyra.LogWarning("stats", "LinearRegression", "x and y must have the same length")
			return nil
		}
	}

	if n <= p {
		insyra.LogWarning("stats", "LinearRegression", "need more observations than variables")
		return nil
	}

	// --- build design matrix X (n×(p+1)) -----------------------------------
	// First column is 1s for intercept, remaining columns are the X variables
	X := make([][]float64, n)
	ys := dlY.ToF64Slice()

	for i := range n {
		X[i] = make([]float64, p+1)
		X[i][0] = 1.0 // intercept column
		for j, dlX := range dlXs {
			X[i][j+1] = dlX.ToF64Slice()[i]
		}
	}

	// --- compute X^T * X (normal matrix) -----------------------------------
	XTX := make([][]float64, p+1)
	for i := 0; i <= p; i++ {
		XTX[i] = make([]float64, p+1)
		for j := 0; j <= p; j++ {
			for k := 0; k < n; k++ {
				XTX[i][j] += X[k][i] * X[k][j]
			}
		}
	}

	// --- compute X^T * y ----------------------------------------------------
	XTy := make([]float64, p+1)
	for i := 0; i <= p; i++ {
		for k := range n {
			XTy[i] += X[k][i] * ys[k]
		}
	}

	// --- solve normal equations β = (X^T*X)^-1 * X^T*y ---------------------
	coeffs := gaussianElimination(XTX, XTy)
	if coeffs == nil {
		insyra.LogWarning("stats", "LinearRegression", "matrix is singular, cannot solve")
		return nil
	}

	// --- calculate predicted values and residuals --------------------------
	residuals := make([]float64, n)
	var sse, sst, sumY float64

	for _, y := range ys {
		sumY += y
	}
	meanY := sumY / float64(n)

	for i := 0; i < n; i++ {
		yHat := 0.0
		for j := 0; j <= p; j++ {
			yHat += coeffs[j] * X[i][j]
		}
		residuals[i] = ys[i] - yHat
		sse += residuals[i] * residuals[i]
		sst += (ys[i] - meanY) * (ys[i] - meanY)
	}

	if sst == 0 {
		insyra.LogWarning("stats", "LinearRegression", "variance of y is zero; R² undefined")
		return nil
	}

	// --- goodness-of-fit ----------------------------------------------------
	rSquared := 1.0 - sse/sst
	df := float64(n - p - 1)
	adjRSquared := 1.0 - (1.0-rSquared)*(float64(n-1))/df

	// --- statistical inference ---------------------------------------------
	XTXInv := invertMatrix(XTX)
	mse := sse / df
	standardErrors := make([]float64, p+1)
	tValues := make([]float64, p+1)
	pValues := make([]float64, p+1)

	for i := 0; i <= p; i++ {
		if XTXInv != nil && XTXInv[i][i] >= 0 {
			standardErrors[i] = math.Sqrt(mse * XTXInv[i][i])
			if standardErrors[i] > 0 {
				tValues[i] = coeffs[i] / standardErrors[i]
				pValues[i] = 2.0 * studentTCDF(-math.Abs(tValues[i]), int(df))
			}
		}
	}

	// --- prepare result with both legacy and extended fields ---------------
	result := &LinearRegressionResult{
		Residuals:        residuals,
		RSquared:         rSquared,
		AdjustedRSquared: adjRSquared,
		Coefficients:     coeffs,
		StandardErrors:   standardErrors,
		TValues:          tValues,
		PValues:          pValues,
	}

	// Fill legacy fields for simple regression compatibility
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

	return result
}

// --------------------------- Exponential ----------------------------------
//
// y = a·e^{b·x}
//
// ** Verified using R **
func ExponentialRegression(dlY, dlX insyra.IDataList) *ExponentialRegressionResult {
	if dlX.Len() != dlY.Len() || dlX.Len() == 0 {
		insyra.LogWarning("stats", "ExponentialRegression", "input lengths mismatch or zero")
		return nil
	}

	xs := dlX.ToF64Slice()
	ys := dlY.ToF64Slice()

	n := float64(len(xs))

	// 計算 log(y)
	logYs := make([]float64, len(ys))
	for i := range ys {
		if ys[i] <= 0 {
			insyra.LogWarning("stats", "ExponentialRegression", "y must be > 0 for log computation")
			return nil
		}
		logYs[i] = math.Log(ys[i])
	}

	// 線性迴歸 ln(y) = ln(a) + bx
	sumX := 0.0
	sumY := 0.0
	sumXY := 0.0
	sumX2 := 0.0
	for i := range xs {
		sumX += xs[i]
		sumY += logYs[i]
		sumXY += xs[i] * logYs[i]
		sumX2 += xs[i] * xs[i]
	}

	denom := n*sumX2 - sumX*sumX
	if denom == 0 {
		insyra.LogWarning("stats", "ExponentialRegression", "denominator zero, cannot compute coefficients")
		return nil
	}

	b := (n*sumXY - sumX*sumY) / denom
	lnA := (sumY - b*sumX) / n
	a := math.Exp(lnA)

	// 預測值和殘差
	residuals := make([]float64, len(xs))
	yHat := make([]float64, len(xs))
	var sse float64
	var sst float64
	meanY := 0.0
	for _, y := range ys {
		meanY += y
	}
	meanY /= n

	for i := range xs {
		yHat[i] = a * math.Exp(b*xs[i])
		residuals[i] = ys[i] - yHat[i]
		sse += residuals[i] * residuals[i]
		sst += (ys[i] - meanY) * (ys[i] - meanY)
	}

	if sst == 0 {
		insyra.LogWarning("stats", "ExponentialRegression", "variance of y is zero; R² undefined")
		return nil
	}

	rSquared := 1.0 - sse/sst
	adjRSquared := 1.0 - (1.0-rSquared)*(n-1)/(n-2)

	// 計算線性回歸的標準誤差 (在對數空間)
	meanX := sumX / n
	df := n - 2

	// 計算對數空間的 SSE
	var sseLog float64
	for i := range xs {
		yHatLog := lnA + b*xs[i]
		residLog := logYs[i] - yHatLog
		sseLog += residLog * residLog
	}
	mseLog := sseLog / df

	var sumXMinusMeanXSquared float64
	for i := range xs {
		sumXMinusMeanXSquared += (xs[i] - meanX) * (xs[i] - meanX)
	}

	seB := math.Sqrt(mseLog / sumXMinusMeanXSquared)
	seLnA := math.Sqrt(mseLog * (1.0/n + meanX*meanX/sumXMinusMeanXSquared))

	// 使用 delta method 計算 a 的標準誤差
	seA := a * seLnA

	tValB := b / seB
	tValA := a / seA // 修正：使用 a/seA 而不是 lnA/seLnA
	pValB := 2.0 * studentTCDF(-math.Abs(tValB), int(df))
	pValA := 2.0 * studentTCDF(-math.Abs(tValA), int(df))

	return &ExponentialRegressionResult{
		Intercept:              a,
		Slope:                  b,
		Residuals:              residuals,
		RSquared:               rSquared,
		AdjustedRSquared:       adjRSquared,
		StandardErrorIntercept: seA,
		StandardErrorSlope:     seB,
		TValueIntercept:        tValA,
		TValueSlope:            tValB,
		PValueIntercept:        pValA,
		PValueSlope:            pValB,
	}
}

// --------------------------- Logarithmic ----------------------------------
// y = a + b·ln(x)
//
// ** Verified using R **
func LogarithmicRegression(dlY, dlX insyra.IDataList) *LogarithmicRegressionResult {
	if dlX.Len() != dlY.Len() || dlX.Len() == 0 {
		insyra.LogWarning("stats", "LogarithmicRegression", "input lengths mismatch or zero")
		return nil
	}

	xs := dlX.ToF64Slice()
	ys := dlY.ToF64Slice()

	var sumLX, sumYLX, sumY, sumLX2 float64
	for i := range xs {
		if xs[i] <= 0 {
			insyra.LogWarning("stats", "LogarithmicRegression", "x must be > 0 for ln computation")
			return nil
		}
		lx := math.Log(xs[i])
		y := ys[i]
		sumLX += lx
		sumYLX += y * lx
		sumY += y
		sumLX2 += lx * lx
	}

	n := float64(dlX.Len())
	denom := n*sumLX2 - sumLX*sumLX
	if denom == 0 {
		insyra.LogWarning("stats", "LogarithmicRegression", "denominator zero, cannot compute coefficients")
		return nil
	}

	// 正確的係數計算：y = a + b·ln(x)
	// b 是 ln(x) 的係數（斜率）
	// a 是截距
	b := (n*sumYLX - sumY*sumLX) / denom // slope coefficient for ln(x)
	a := (sumY - b*sumLX) / n            // intercept

	// Calculate residuals properly
	residuals := make([]float64, len(xs))
	var sse, sst float64
	var meanY float64
	for _, v := range ys {
		meanY += v
	}
	meanY /= n

	for i := range xs {
		yHat := a + b*math.Log(xs[i])
		residuals[i] = ys[i] - yHat
		sse += residuals[i] * residuals[i]
		sst += (ys[i] - meanY) * (ys[i] - meanY)
	}

	if sst == 0 {
		insyra.LogWarning("stats", "LogarithmicRegression", "variance of y is zero; R² undefined")
		return nil
	}

	r2 := 1.0 - sse/sst
	df := n - 2
	adjR2 := 1.0 - (1.0-r2)*(n-1)/df

	mse := sse / df
	meanLX := sumLX / n

	// 計算標準誤差 - 修正公式
	var sxxLX float64 // Σ(ln(x) - mean(ln(x)))²
	for i := range xs {
		lx := math.Log(xs[i])
		diff := lx - meanLX
		sxxLX += diff * diff
	}

	seB := math.Sqrt(mse / sxxLX)
	seA := math.Sqrt(mse * (1.0/n + meanLX*meanLX/sxxLX))

	tValB := b / seB
	tValA := a / seA
	pValB := 2.0 * studentTCDF(-math.Abs(tValB), int(df))
	pValA := 2.0 * studentTCDF(-math.Abs(tValA), int(df))

	return &LogarithmicRegressionResult{
		Intercept:              a, // intercept in y = a + b·ln(x)
		Slope:                  b, // slope coefficient in ln(x) in y = a + b·ln(x)
		Residuals:              residuals,
		RSquared:               r2,
		AdjustedRSquared:       adjR2,
		StandardErrorIntercept: seA,
		StandardErrorSlope:     seB,
		TValueIntercept:        tValA,
		TValueSlope:            tValB,
		PValueIntercept:        pValA,
		PValueSlope:            pValB,
	}
}

// --------------------------- Polynomial -----------------------------------
// y = a₀ + a₁x + a₂x² + ... + aₙxⁿ
//
// ** Verified using R **
func PolynomialRegression(dlY, dlX insyra.IDataList, degree int) *PolynomialRegressionResult {
	if dlX.Len() != dlY.Len() || dlX.Len() == 0 {
		insyra.LogWarning("stats", "PolynomialRegression", "input lengths mismatch or zero")
		return nil
	}
	if degree < 1 || degree >= dlX.Len() {
		insyra.LogWarning("stats", "PolynomialRegression", "invalid degree")
		return nil
	}

	xs := dlX.ToF64Slice()
	ys := dlY.ToF64Slice()
	n := len(xs)

	// Create design matrix X (Vandermonde matrix)
	// X[i][j] = xs[i]^j for j = 0, 1, ..., degree
	X := make([][]float64, n)
	for i := range X {
		X[i] = make([]float64, degree+1)
		X[i][0] = 1.0 // x^0 = 1
		for j := 1; j <= degree; j++ {
			X[i][j] = X[i][j-1] * xs[i] // x^j
		}
	}

	// Compute X^T * X (normal matrix)
	XTX := make([][]float64, degree+1)
	for i := range XTX {
		XTX[i] = make([]float64, degree+1)
		for j := range XTX[i] {
			for k := 0; k < n; k++ {
				XTX[i][j] += X[k][i] * X[k][j]
			}
		}
	}

	// Compute X^T * y
	XTy := make([]float64, degree+1)
	for i := 0; i <= degree; i++ {
		for j := range n {
			XTy[i] += X[j][i] * ys[j]
		}
	}

	// Solve XTX * coeffs = XTy using Gaussian elimination
	coeffs := gaussianElimination(XTX, XTy)
	if coeffs == nil {
		insyra.LogWarning("stats", "PolynomialRegression", "matrix is singular, cannot solve")
		return nil
	}

	// Calculate predicted values and residuals
	residuals := make([]float64, n)
	var sse, sst float64
	var meanY float64
	for _, y := range ys {
		meanY += y
	}
	meanY /= float64(n)

	for i := range n {
		yHat := 0.0
		for j := 0; j <= degree; j++ {
			yHat += coeffs[j] * X[i][j]
		}
		residuals[i] = ys[i] - yHat
		sse += residuals[i] * residuals[i]
		sst += (ys[i] - meanY) * (ys[i] - meanY)
	}

	if sst == 0 {
		insyra.LogWarning("stats", "PolynomialRegression", "variance of y is zero; R² undefined")
		return nil
	}

	r2 := 1.0 - sse/sst
	df := float64(n - degree - 1)
	adjR2 := 1.0 - (1.0-r2)*(float64(n-1))/df

	// Calculate standard errors using diagonal of (X^T * X)^(-1)
	XTXInv := invertMatrix(XTX)
	mse := sse / df
	standardErrors := make([]float64, degree+1)
	tValues := make([]float64, degree+1)
	pValues := make([]float64, degree+1)

	for i := 0; i <= degree; i++ {
		if XTXInv != nil && XTXInv[i][i] >= 0 {
			standardErrors[i] = math.Sqrt(mse * XTXInv[i][i])
			if standardErrors[i] > 0 {
				tValues[i] = coeffs[i] / standardErrors[i]
				pValues[i] = 2.0 * studentTCDF(-math.Abs(tValues[i]), int(df))
			}
		}
	}

	return &PolynomialRegressionResult{
		Coefficients:     coeffs,
		Degree:           degree,
		Residuals:        residuals,
		RSquared:         r2,
		AdjustedRSquared: adjR2,
		StandardErrors:   standardErrors,
		TValues:          tValues,
		PValues:          pValues,
	}
}

// gaussianElimination solves Ax = b using Gaussian elimination with partial pivoting
func gaussianElimination(A [][]float64, b []float64) []float64 {
	n := len(A)
	if n == 0 || len(b) != n {
		return nil
	}

	// Create augmented matrix [A|b]
	aug := make([][]float64, n)
	for i := range aug {
		aug[i] = make([]float64, n+1)
		copy(aug[i][:n], A[i])
		aug[i][n] = b[i]
	}

	// Forward elimination with partial pivoting
	for i := range n {
		// Find pivot
		maxRow := i
		for k := i + 1; k < n; k++ {
			if math.Abs(aug[k][i]) > math.Abs(aug[maxRow][i]) {
				maxRow = k
			}
		}

		// Swap rows
		aug[i], aug[maxRow] = aug[maxRow], aug[i]

		// Check for singular matrix
		if math.Abs(aug[i][i]) < 1e-12 {
			return nil
		}

		// Eliminate column
		for k := i + 1; k < n; k++ {
			factor := aug[k][i] / aug[i][i]
			for j := i; j <= n; j++ {
				aug[k][j] -= factor * aug[i][j]
			}
		}
	}

	// Back substitution
	x := make([]float64, n)
	for i := n - 1; i >= 0; i-- {
		x[i] = aug[i][n]
		for j := i + 1; j < n; j++ {
			x[i] -= aug[i][j] * x[j]
		}
		x[i] /= aug[i][i]
	}

	return x
}

// invertMatrix computes the inverse of a matrix using Gauss-Jordan elimination
func invertMatrix(A [][]float64) [][]float64 {
	n := len(A)
	if n == 0 {
		return nil
	}

	// Create augmented matrix [A|I]
	aug := make([][]float64, n)
	for i := range aug {
		aug[i] = make([]float64, 2*n)
		copy(aug[i][:n], A[i])
		aug[i][n+i] = 1.0 // Identity matrix
	}

	// Gauss-Jordan elimination
	for i := range n {
		// Find pivot
		maxRow := i
		for k := i + 1; k < n; k++ {
			if math.Abs(aug[k][i]) > math.Abs(aug[maxRow][i]) {
				maxRow = k
			}
		}

		// Swap rows
		aug[i], aug[maxRow] = aug[maxRow], aug[i]

		// Check for singular matrix
		if math.Abs(aug[i][i]) < 1e-12 {
			return nil
		}

		// Scale pivot row
		pivot := aug[i][i]
		for j := 0; j < 2*n; j++ {
			aug[i][j] /= pivot
		}

		// Eliminate column
		for k := 0; k < n; k++ {
			if k != i {
				factor := aug[k][i]
				for j := 0; j < 2*n; j++ {
					aug[k][j] -= factor * aug[i][j]
				}
			}
		}
	}

	// Extract inverse matrix
	inv := make([][]float64, n)
	for i := range inv {
		inv[i] = make([]float64, n)
		copy(inv[i], aug[i][n:])
	}

	return inv
}

// ---------------------------------------------------------------------------
// Auxiliary maths: CDF of two-tailed Student's t using the regularised beta.
// ---------------------------------------------------------------------------

// studentTCDF returns P(T ≤ t) for Student's t-distribution with `df` degrees
// of freedom using the relation to the regularised incomplete beta function.
func studentTCDF(t float64, df int) float64 {
	if df <= 0 {
		return math.NaN()
	}
	x := float64(df) / (float64(df) + t*t)

	ib := regIncBeta(float64(df)/2.0, 0.5, x)
	if t <= 0 {
		return 0.5 * ib
	}
	return 1.0 - 0.5*ib
}

// regIncBeta computes the regularised incomplete beta Iₓ(a,b) via a continued
// fraction expansion (modified Lentz algorithm).
func regIncBeta(a, b, x float64) float64 {
	if x < 0.0 || x > 1.0 {
		return math.NaN()
	}
	if x == 0.0 || x == 1.0 {
		return x
	}

	lgab, _ := math.Lgamma(a + b)
	lga, _ := math.Lgamma(a)
	lgb, _ := math.Lgamma(b)

	// Prefactor for I_z(alpha, beta) is Exp(Lgamma(alpha+beta)-Lgamma(alpha)-Lgamma(beta)+alpha*Log(z)+beta*Log(1-z)) / alpha
	// Common term for B(a,b)^-1 or B(b,a)^-1
	betaCoeff := math.Exp(lgab - lga - lgb)

	var cf float64
	if x < (a+1.0)/(a+b+2.0) {
		// Calculate I_x(a,b) directly
		prefactor := betaCoeff * math.Pow(x, a) * math.Pow(1.0-x, b) / a
		cf = betaCF(a, b, x)
		return prefactor * cf
	}
	// Calculate I_x(a,b) as 1 - I_{1-x}(b,a)
	// Need prefactor for I_{1-x}(b,a)
	// Here, alpha=b, beta=a, z=1-x
	prefactorForSwappedTerm := betaCoeff * math.Pow(1.0-x, b) * math.Pow(x, a) / b
	cf = betaCF(b, a, 1.0-x) // Continued fraction for I_{1-x}(b,a)
	return 1.0 - prefactorForSwappedTerm*cf
}

// betaCF evaluates the continued-fraction form for the incomplete beta.
func betaCF(a, b, x float64) float64 {
	const (
		maxIter = 300
		eps     = 1e-14
		tiny    = 1e-30
	)

	c, d := 1.0, 1.0-((a+b)*x)/(a+1.0)
	if math.Abs(d) < tiny {
		d = tiny
	}
	d = 1.0 / d
	h := d

	for m := 1; m <= maxIter; m++ {
		m2 := 2 * m

		aa := float64(m) * (b - float64(m)) * x / ((a + float64(m2) - 1) * (a + float64(m2)))
		d = 1.0 + aa*d
		if math.Abs(d) < tiny {
			d = tiny
		}
		c = 1.0 + aa/c
		if math.Abs(c) < tiny {
			c = tiny
		}
		d = 1.0 / d
		h *= d * c

		aa = -(a + float64(m)) * (a + b + float64(m)) * x / ((a + float64(m2)) * (a + float64(m2) + 1))
		d = 1.0 + aa*d
		if math.Abs(d) < tiny {
			d = tiny
		}
		c = 1.0 + aa/c
		if math.Abs(c) < tiny {
			c = tiny
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
