package quant

import (
	"errors"
	"fmt"
	"math"

	"github.com/HazelnutParadise/insyra"
)

// ErrSharpeUndefined is returned when a series has no variance, so its Sharpe
// ratio does not exist: a strategy that never entered the market, for example.
var ErrSharpeUndefined = errors.New("a series has zero standard deviation, so its Sharpe ratio is undefined")

// SharpeDifferenceResult is the result of SharpeDifferenceTest. The Sharpe
// ratios are per period, not annualized.
type SharpeDifferenceResult struct {
	SharpeA, SharpeB float64
	Difference       float64 // SharpeA - SharpeB
	Z                float64 // the Jobson-Korkie statistic with Memmel's correction
	PValue           float64 // two-tailed
	PValueGreater    float64 // one-tailed, H1: SharpeA > SharpeB
	PValueLess       float64 // one-tailed, H1: SharpeA < SharpeB
	N                int
}

// SharpeDifferenceTest tests whether two return series measured over the same
// periods have the same Sharpe ratio, with the Jobson-Korkie (1981) statistic
// as corrected by Memmel (2003). riskFree is the per-period risk-free rate
// subtracted from both series. The p-values come from the standard normal
// distribution.
//
// The test assumes returns that are roughly normal and not autocorrelated; with
// fat tails or serial correlation it is too eager to reject, and a HAC or
// bootstrap test (Ledoit and Wolf 2008) is the more reliable choice. It is a
// closed formula, so it is cheap enough to run on thousands of series.
//
// A series with zero standard deviation has no Sharpe ratio and returns
// ErrSharpeUndefined. The series must have the same length, at least 3 values
// and only finite numbers.
func SharpeDifferenceTest(a, b insyra.IDataList, riskFree float64) (*SharpeDifferenceResult, error) {
	if math.IsNaN(riskFree) || math.IsInf(riskFree, 0) {
		return nil, errors.New("riskFree must be a finite number")
	}
	x, err := numericSeries(a, "a")
	if err != nil {
		return nil, err
	}
	y, err := numericSeries(b, "b")
	if err != nil {
		return nil, err
	}
	if len(x) != len(y) {
		return nil, fmt.Errorf("a has %d rows and b has %d", len(x), len(y))
	}
	n := len(x)
	if n < 3 {
		return nil, errors.New("at least 3 observations are needed")
	}

	nf := float64(n)
	var muA, muB float64
	for i := range x {
		muA += x[i] - riskFree
		muB += y[i] - riskFree
	}
	muA, muB = muA/nf, muB/nf
	var vA, vB, cov float64
	for i := range x {
		da, db := x[i]-riskFree-muA, y[i]-riskFree-muB
		vA += da * da
		vB += db * db
		cov += da * db
	}
	vA, vB, cov = vA/nf, vB/nf, cov/nf
	if vA == 0 {
		return nil, fmt.Errorf("series a: %w", ErrSharpeUndefined)
	}
	if vB == 0 {
		return nil, fmt.Errorf("series b: %w", ErrSharpeUndefined)
	}
	sA, sB := math.Sqrt(vA), math.Sqrt(vB)

	theta := (2*vA*vB - 2*sA*sB*cov + 0.5*muA*muA*vB + 0.5*muB*muB*vA -
		(muA*muB/(sA*sB))*(cov*cov+vA*vB)) / nf
	result := &SharpeDifferenceResult{SharpeA: muA / sA, SharpeB: muB / sB, N: n}
	result.Difference = result.SharpeA - result.SharpeB
	if theta <= 0 {
		// Identical series give theta = 0 and a difference of exactly 0.
		if result.Difference == 0 || math.Abs(result.Difference) < 1e-12 {
			result.Z, result.PValue, result.PValueGreater, result.PValueLess = 0, 1, 0.5, 0.5
			return result, nil
		}
		return nil, errors.New("the variance of the Sharpe difference is not positive")
	}
	result.Z = (sB*muA - sA*muB) / math.Sqrt(theta)
	result.PValue = 2 * zCDFUpper(math.Abs(result.Z))
	result.PValueGreater = zCDFUpper(result.Z)
	result.PValueLess = zCDFUpper(-result.Z)
	return result, nil
}

// zCDFUpper is P(Z > z) for a standard normal Z.
func zCDFUpper(z float64) float64 {
	return 0.5 * math.Erfc(z/math.Sqrt2)
}
