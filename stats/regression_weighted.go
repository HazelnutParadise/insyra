package stats

import (
	"errors"
	"fmt"
	"math"

	"github.com/HazelnutParadise/insyra"
	"gonum.org/v1/gonum/mat"
)

// Layer 4 — weighted least squares.
//
// This is the sample-weight family that has exact classical inference: the
// weighted normal equations, the coefficient covariance and the t statistics
// are all closed-form, which is why the result carries standard errors and p
// values where the penalized results deliberately do not.

// WeightedLinearRegressionResult is a fitted weighted least-squares model.
type WeightedLinearRegressionResult struct {
	// Coefficients holds the intercept at index 0 followed by one coefficient
	// per predictor, matching LinearRegressionResult's layout.
	Coefficients   []float64
	StandardErrors []float64
	TValues        []float64
	PValues        []float64

	Residuals        []float64
	RSquared         float64
	AdjustedRSquared float64
}

// WeightedLinearRegression fits y = β₀ + Σβⱼxⱼ minimising Σwᵢ·eᵢ², with
// inference computed under the weights — statsmodels' WLS exactly, which is
// what it is verified against.
//
// Weights must be strictly positive finite numbers. A zero weight is refused
// rather than treated as exclusion: references disagree on whether an
// excluded row still counts toward degrees of freedom, and guessing between
// their answers would produce standard errors that match nobody. Drop the row
// instead.
func WeightedLinearRegression(dlY insyra.IDataList, dlWeights insyra.IDataList, dlXs ...insyra.IDataList) (*WeightedLinearRegressionResult, error) {
	if dlWeights == nil {
		return nil, errors.New("weights data list is nil")
	}
	y, xs, extras, n, err := gatherRegressionInputs(dlY, dlXs, dlWeights)
	if err != nil {
		return nil, err
	}
	weights := extras[0]
	for i, w := range weights {
		if w <= 0 {
			return nil, fmt.Errorf("weights must be strictly positive; row %d holds %v", i+1, w)
		}
	}
	p := len(xs)
	if n <= p+1 {
		return nil, fmt.Errorf("need more observations (%d) than coefficients (%d)", n, p+1)
	}

	// X'WX and X'Wy, assembled directly — W is diagonal so each is a weighted
	// accumulation over rows.
	X := buildDesignMatrix(xs, n)
	gram := mat.NewDense(p+1, p+1, nil)
	rhs := make([]float64, p+1)
	for i := 0; i < n; i++ {
		w := weights[i]
		for a := 0; a <= p; a++ {
			xa := X.At(i, a)
			rhs[a] += w * xa * y[i]
			for b := a; b <= p; b++ {
				gram.Set(a, b, gram.At(a, b)+w*xa*X.At(i, b))
			}
		}
	}
	for a := 0; a <= p; a++ {
		for b := 0; b < a; b++ {
			gram.Set(a, b, gram.At(b, a))
		}
	}

	var beta mat.VecDense
	if err := beta.SolveVec(gram, mat.NewVecDense(p+1, rhs)); err != nil {
		return nil, fmt.Errorf("weighted normal equations are singular: %w", err)
	}
	coefficients := make([]float64, p+1)
	for j := range coefficients {
		coefficients[j] = beta.AtVec(j)
	}

	// Residuals, weighted SSE, and the weighted R² statsmodels reports:
	// 1 − SSRw/CTSSw with the weighted mean as the centre.
	fitted := func(i int) float64 {
		out := coefficients[0]
		for j := 0; j < p; j++ {
			out += coefficients[j+1] * xs[j][i]
		}
		return out
	}
	residuals := make([]float64, n)
	weightTotal := 0.0
	weightedMean := 0.0
	for i := 0; i < n; i++ {
		residuals[i] = y[i] - fitted(i)
		weightTotal += weights[i]
		weightedMean += weights[i] * y[i]
	}
	weightedMean /= weightTotal
	ssr := 0.0
	ctss := 0.0
	for i := 0; i < n; i++ {
		ssr += weights[i] * residuals[i] * residuals[i]
		centred := y[i] - weightedMean
		ctss += weights[i] * centred * centred
	}
	df := float64(n - p - 1)
	rSquared := math.NaN()
	adjusted := math.NaN()
	if ctss != 0 {
		rSquared = 1 - ssr/ctss
		adjusted = 1 - (1-rSquared)*float64(n-1)/df
	}

	// cov(β) = σ̂²·(X'WX)⁻¹ with σ̂² = SSRw/df — the exact classical form.
	sigma2 := ssr / df
	var inverse mat.Dense
	if err := inverse.Inverse(gram); err != nil {
		return nil, fmt.Errorf("weighted normal equations are singular: %w", err)
	}
	standardErrors := make([]float64, p+1)
	tValues := make([]float64, p+1)
	pValues := make([]float64, p+1)
	for j := 0; j <= p; j++ {
		standardErrors[j] = math.Sqrt(sigma2 * inverse.At(j, j))
		tValues[j] = coefficients[j] / standardErrors[j]
		pValues[j] = tTwoTailedPValue(tValues[j], df)
	}

	return &WeightedLinearRegressionResult{
		Coefficients:     coefficients,
		StandardErrors:   standardErrors,
		TValues:          tValues,
		PValues:          pValues,
		Residuals:        residuals,
		RSquared:         rSquared,
		AdjustedRSquared: adjusted,
	}, nil
}

// Predict returns response-scale point predictions for new observations.
func (r *WeightedLinearRegressionResult) Predict(typ PredictType, newXs ...insyra.IDataList) (*insyra.DataList, error) {
	if r == nil {
		return nil, errors.New("weighted linear regression result is nil")
	}
	return predictLinearFromCoefficients("weighted linear", r.Coefficients, typ, newXs)
}
