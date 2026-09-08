package quant

import (
	"fmt"
	"math"

	"github.com/HazelnutParadise/insyra"
)

// CAPMResult holds a single-index regression of aligned per-period asset
// returns on aligned per-period market returns. Alpha is the per-period
// intercept after subtracting the per-period risk-free rate. For a constant
// excess asset return, RSquared is NaN because the total variance is zero;
// both standard errors are zero.
type CAPMResult struct {
	// Beta is the market exposure, or OLS slope.
	Beta float64
	// Alpha is Jensen's alpha per period, or the OLS intercept on excess returns.
	Alpha float64
	// RSquared is the coefficient of determination. It is NaN for a constant
	// excess asset return.
	RSquared float64
	// BetaStdErr is the OLS standard error of Beta.
	BetaStdErr float64
	// AlphaStdErr is the OLS standard error of Alpha.
	AlphaStdErr float64
	// N is the number of aligned return observations.
	N int
}

// CAPM fits the single-index market model by regressing asset excess returns
// on market excess returns. asset and market must be aligned per-period
// return series, not prices, and riskFreeRate is a per-period rate (pass 0
// for a raw-return regression). Alpha is therefore also per period.
//
// Returns an error if either input is nil, their lengths differ, fewer than
// three observations are supplied, the market has zero variance, or a cell
// is non-numeric or non-finite. Input values are read through numericSeries;
// no cell is dropped or coerced to zero.
func CAPM(asset, market insyra.IDataList, riskFreeRate float64) (*CAPMResult, error) {
	if asset == nil {
		return nil, fmt.Errorf("CAPM: asset is nil")
	}
	if market == nil {
		return nil, fmt.Errorf("CAPM: market is nil")
	}
	if asset.Len() != market.Len() {
		return nil, fmt.Errorf("CAPM: asset and market lengths differ (%d vs %d)", asset.Len(), market.Len())
	}

	assetValues, err := numericSeries(asset, "asset")
	if err != nil {
		return nil, err
	}
	marketValues, err := numericSeries(market, "market")
	if err != nil {
		return nil, err
	}
	return capmF64(assetValues, marketValues, riskFreeRate)
}

// Beta returns the market beta of aligned per-period asset returns against
// aligned per-period market returns: covariance(asset, market) divided by
// variance(market). The covariance and variance use the same sample
// denominator, so Beta is the one-predictor OLS slope. Inputs are returns,
// not prices; date alignment and price-to-return conversion are outside this
// package.
//
// Returns an error if either input is nil, their lengths differ, fewer than
// three observations are supplied, the market has zero variance, or a cell
// is non-numeric or non-finite. Input values are read through numericSeries;
// no cell is dropped or coerced to zero.
func Beta(asset, market insyra.IDataList) (float64, error) {
	if asset == nil {
		return math.NaN(), fmt.Errorf("Beta: asset is nil")
	}
	if market == nil {
		return math.NaN(), fmt.Errorf("Beta: market is nil")
	}
	if asset.Len() != market.Len() {
		return math.NaN(), fmt.Errorf("Beta: asset and market lengths differ (%d vs %d)", asset.Len(), market.Len())
	}

	assetValues, err := numericSeries(asset, "asset")
	if err != nil {
		return math.NaN(), err
	}
	marketValues, err := numericSeries(market, "market")
	if err != nil {
		return math.NaN(), err
	}
	return betaF64(assetValues, marketValues)
}

func capmF64(asset, market []float64, rf float64) (*CAPMResult, error) {
	meanX, meanY, Sxx, Sxy, err := capmMoments("CAPM", asset, market, rf)
	if err != nil {
		return nil, err
	}

	constantAsset := constantSeries(asset)
	beta := Sxy / Sxx
	if constantAsset {
		beta = 0
	}
	alpha := meanY - beta*meanX
	SSR := 0.0
	SST := 0.0
	if !constantAsset {
		for i := range asset {
			x := market[i] - rf
			y := asset[i] - rf
			residual := y - (alpha + beta*x)
			SSR += residual * residual
			centered := y - meanY
			SST += centered * centered
		}
	}

	n := len(asset)
	s2 := SSR / float64(n-2)
	betaStdErr := math.Sqrt(s2 / Sxx)
	alphaStdErr := math.Sqrt(s2 * (1/float64(n) + meanX*meanX/Sxx))
	rSquared := math.NaN()
	if SST != 0 {
		rSquared = 1 - SSR/SST
	}

	return &CAPMResult{
		Beta:        beta,
		Alpha:       alpha,
		RSquared:    rSquared,
		BetaStdErr:  betaStdErr,
		AlphaStdErr: alphaStdErr,
		N:           n,
	}, nil
}

func betaF64(asset, market []float64) (float64, error) {
	_, _, Sxx, Sxy, err := capmMoments("Beta", asset, market, 0)
	if err != nil {
		return math.NaN(), err
	}
	return Sxy / Sxx, nil
}

// capmMoments returns the excess-return means and the centred sums Sxx and
// Sxy. funcName prefixes error messages so Beta and CAPM each name themselves.
func capmMoments(funcName string, asset, market []float64, rf float64) (meanX, meanY, Sxx, Sxy float64, err error) {
	if len(asset) != len(market) {
		return 0, 0, 0, 0, fmt.Errorf("%s: asset and market lengths differ (%d vs %d)", funcName, len(asset), len(market))
	}
	n := len(asset)
	if n < 3 {
		return 0, 0, 0, 0, fmt.Errorf("%s: need at least 3 observations, got %d", funcName, n)
	}

	for i := range n {
		meanX += market[i] - rf
		meanY += asset[i] - rf
	}
	meanX /= float64(n)
	meanY /= float64(n)
	for i := range n {
		x := market[i] - rf
		y := asset[i] - rf
		dx := x - meanX
		dy := y - meanY
		Sxx += dx * dx
		Sxy += dx * dy
	}
	if constantSeries(market) || Sxx == 0 {
		return 0, 0, 0, 0, fmt.Errorf("%s: benchmark variance is zero", funcName)
	}
	if constantSeries(asset) {
		Sxy = 0
	}
	return meanX, meanY, Sxx, Sxy, nil
}

func constantSeries(values []float64) bool {
	if len(values) < 2 {
		return true
	}
	for _, value := range values[1:] {
		if value != values[0] {
			return false
		}
	}
	return true
}
