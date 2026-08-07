package stats

import (
	"errors"
	"fmt"
	"math"

	"github.com/HazelnutParadise/insyra"
	"gonum.org/v1/gonum/mat"
)

// Layer 4 — penalized linear regression.
//
// The objectives are scikit-learn's, stated exactly because the other obvious
// reference disagrees with it. Ridge minimises ||y − Xβ||² + α·||β||² and
// lasso minimises (1/2n)·||y − Xβ||² + α·||β||₁; the intercept is never
// penalized and features are not standardised. R's glmnet standardises by
// default and scales its penalty by a different factor, so "matches glmnet"
// and "matches scikit-learn" are different coefficients from the same data.
// This library imitates scikit-learn's API, so its objectives are the ones a
// caller comparing results will expect, and the cross-language verification is
// against scikit-learn.
//
// Neither result type carries standard errors, t values or p values. Penalized
// estimates have no exact classical sampling distribution — the unpenalized
// formulas stop being valid the moment the penalty biases the estimator — so
// reporting those fields would be fabricating precision, not providing it.

// RidgeRegressionResult is a fitted L2-penalized linear model.
type RidgeRegressionResult struct {
	// Coefficients holds the intercept at index 0 followed by one coefficient
	// per predictor, matching LinearRegressionResult's layout.
	Coefficients []float64
	Alpha        float64

	Residuals        []float64
	RSquared         float64
	AdjustedRSquared float64
}

// LassoRegressionResult is a fitted L1-penalized linear model.
type LassoRegressionResult struct {
	// Coefficients holds the intercept at index 0 followed by one coefficient
	// per predictor. Predictors the penalty priced out hold exactly zero.
	Coefficients []float64
	Alpha        float64

	Residuals        []float64
	RSquared         float64
	AdjustedRSquared float64

	// Converged reports whether coordinate descent met its tolerance within
	// the iteration cap; Iterations is how many full passes ran. A result that
	// did not converge is still returned — it is the best estimate available,
	// and the flag is how the caller knows to raise the cap.
	Converged  bool
	Iterations int
}

// LassoOptions tunes coordinate descent. The zero value means scikit-learn's
// defaults: tolerance 1e-4, iteration cap 1000.
type LassoOptions struct {
	// Tolerance stops the descent when no coefficient moved more than this in
	// a full pass. Must be positive when set.
	Tolerance float64
	// MaxIterations caps the number of full passes. Must be positive when set.
	MaxIterations int
}

func validatePenalty(alpha float64) error {
	if math.IsNaN(alpha) || math.IsInf(alpha, 0) {
		return fmt.Errorf("penalty must be a finite number, got %v", alpha)
	}
	if alpha < 0 {
		return fmt.Errorf("penalty must not be negative, got %v", alpha)
	}
	return nil
}

// RidgeRegression fits y = β₀ + Σβⱼxⱼ minimising ||y − Xβ||² + α·||β₁..βₚ||².
// α = 0 reproduces ordinary least squares. The solve is the penalized normal
// equations, exact rather than iterative — the penalty makes them nonsingular
// even when the predictors are collinear, which is the ordinary reason to
// reach for ridge in the first place.
func RidgeRegression(dlY insyra.IDataList, alpha float64, dlXs ...insyra.IDataList) (*RidgeRegressionResult, error) {
	if err := validatePenalty(alpha); err != nil {
		return nil, err
	}
	y, xs, _, n, err := gatherRegressionInputs(dlY, dlXs)
	if err != nil {
		return nil, err
	}
	p := len(xs)
	if n <= p+1 {
		return nil, fmt.Errorf("need more observations (%d) than coefficients (%d)", n, p+1)
	}

	X := buildDesignMatrix(xs, n)

	// A = X'X + αD with D = diag(0, 1, …, 1): the zero exempts the intercept.
	var gram mat.Dense
	gram.Mul(X.T(), X)
	for j := 1; j <= p; j++ {
		gram.Set(j, j, gram.At(j, j)+alpha)
	}
	b := make([]float64, p+1)
	for j := 0; j <= p; j++ {
		total := 0.0
		for i := 0; i < n; i++ {
			total += X.At(i, j) * y[i]
		}
		b[j] = total
	}
	var beta mat.VecDense
	if err := beta.SolveVec(&gram, mat.NewVecDense(p+1, b)); err != nil {
		return nil, fmt.Errorf("penalized normal equations are singular: %w", err)
	}

	coefficients := make([]float64, p+1)
	for j := range coefficients {
		coefficients[j] = beta.AtVec(j)
	}
	result := &RidgeRegressionResult{Coefficients: coefficients, Alpha: alpha}
	result.Residuals, result.RSquared, result.AdjustedRSquared, _, _ = computeGoodnessOfFit(y, func(i int) float64 {
		fitted := coefficients[0]
		for j := 0; j < p; j++ {
			fitted += coefficients[j+1] * xs[j][i]
		}
		return fitted
	}, float64(n-p-1))
	return result, nil
}

// LassoRegression fits y = β₀ + Σβⱼxⱼ minimising (1/2n)·||y − Xβ||² +
// α·||β₁..βₚ||₁ by coordinate descent with soft thresholding, on centered
// data so the intercept stays unpenalized. A predictor whose contribution is
// worth less than the penalty gets a coefficient of exactly zero — that
// sparsity is lasso's point, and it is exact, not rounding.
func LassoRegression(dlY insyra.IDataList, alpha float64, dlXs []insyra.IDataList, options ...LassoOptions) (*LassoRegressionResult, error) {
	if err := validatePenalty(alpha); err != nil {
		return nil, err
	}
	if len(options) > 1 {
		return nil, errors.New("options accepts at most one value")
	}
	tolerance := 1e-4
	maxIterations := 1000
	if len(options) == 1 {
		if options[0].Tolerance != 0 {
			if options[0].Tolerance < 0 || math.IsNaN(options[0].Tolerance) {
				return nil, fmt.Errorf("tolerance must be positive, got %v", options[0].Tolerance)
			}
			tolerance = options[0].Tolerance
		}
		if options[0].MaxIterations != 0 {
			if options[0].MaxIterations < 0 {
				return nil, fmt.Errorf("iteration cap must be positive, got %d", options[0].MaxIterations)
			}
			maxIterations = options[0].MaxIterations
		}
	}
	y, xs, _, n, err := gatherRegressionInputs(dlY, dlXs)
	if err != nil {
		return nil, err
	}
	p := len(xs)
	if n < 2 {
		return nil, errors.New("need at least two observations")
	}

	// Center predictors and response; the intercept is recovered afterwards
	// from the means, which is what leaves it unpenalized.
	xMeans := make([]float64, p)
	centered := make([][]float64, p)
	for j := range xs {
		total := 0.0
		for _, value := range xs[j] {
			total += value
		}
		xMeans[j] = total / float64(n)
		centered[j] = make([]float64, n)
		for i, value := range xs[j] {
			centered[j][i] = value - xMeans[j]
		}
	}
	yMean := 0.0
	for _, value := range y {
		yMean += value
	}
	yMean /= float64(n)

	// columnNormSq[j] = Σᵢ xᵢⱼ², the coordinate update's denominator. A column
	// that is constant after centering has no signal to price; its coefficient
	// stays zero, matching scikit-learn.
	columnNormSq := make([]float64, p)
	for j := range centered {
		for _, value := range centered[j] {
			columnNormSq[j] += value * value
		}
	}

	weights := make([]float64, p)
	residual := make([]float64, n)
	for i := range residual {
		residual[i] = y[i] - yMean
	}
	threshold := alpha * float64(n)

	iterations := 0
	converged := false
	for iterations < maxIterations {
		iterations++
		maxChange := 0.0
		for j := 0; j < p; j++ {
			if columnNormSq[j] == 0 {
				continue
			}
			// ρ = Xⱼ'r + wⱼ·Xⱼ'Xⱼ is the coordinate's least-squares answer with
			// itself removed from the residual; the soft threshold then takes
			// the penalty's cut, and zero is what remains when the cut is the
			// whole of it.
			rho := weights[j] * columnNormSq[j]
			for i := 0; i < n; i++ {
				rho += centered[j][i] * residual[i]
			}
			updated := 0.0
			switch {
			case rho > threshold:
				updated = (rho - threshold) / columnNormSq[j]
			case rho < -threshold:
				updated = (rho + threshold) / columnNormSq[j]
			}
			if change := updated - weights[j]; change != 0 {
				for i := 0; i < n; i++ {
					residual[i] -= change * centered[j][i]
				}
				if math.Abs(change) > maxChange {
					maxChange = math.Abs(change)
				}
				weights[j] = updated
			}
		}
		if maxChange <= tolerance {
			converged = true
			break
		}
	}

	coefficients := make([]float64, p+1)
	coefficients[0] = yMean
	for j := 0; j < p; j++ {
		coefficients[j+1] = weights[j]
		coefficients[0] -= weights[j] * xMeans[j]
	}
	result := &LassoRegressionResult{
		Coefficients: coefficients,
		Alpha:        alpha,
		Converged:    converged,
		Iterations:   iterations,
	}
	result.Residuals, result.RSquared, result.AdjustedRSquared, _, _ = computeGoodnessOfFit(y, func(i int) float64 {
		fitted := coefficients[0]
		for j := 0; j < p; j++ {
			fitted += coefficients[j+1] * xs[j][i]
		}
		return fitted
	}, float64(n-p-1))
	return result, nil
}

// Predict returns response-scale point predictions for new observations.
func (r *RidgeRegressionResult) Predict(typ PredictType, newXs ...insyra.IDataList) (*insyra.DataList, error) {
	if r == nil {
		return nil, errors.New("ridge regression result is nil")
	}
	return predictLinearFromCoefficients("ridge", r.Coefficients, typ, newXs)
}

// Predict returns response-scale point predictions for new observations.
func (r *LassoRegressionResult) Predict(typ PredictType, newXs ...insyra.IDataList) (*insyra.DataList, error) {
	if r == nil {
		return nil, errors.New("lasso regression result is nil")
	}
	return predictLinearFromCoefficients("lasso", r.Coefficients, typ, newXs)
}

func predictLinearFromCoefficients(model string, coefficients []float64, typ PredictType, newXs []insyra.IDataList) (*insyra.DataList, error) {
	if len(coefficients) == 0 {
		return nil, errors.New("no coefficients available")
	}
	xs, n, err := prepareRegressionPrediction(model, typ, len(coefficients)-1, newXs)
	if err != nil {
		return nil, err
	}
	out := make([]any, n)
	for i := 0; i < n; i++ {
		fitted := coefficients[0]
		for j := range xs {
			fitted += coefficients[j+1] * xs[j][i]
		}
		out[i] = fitted
	}
	return insyra.NewDataList(out...), nil
}
