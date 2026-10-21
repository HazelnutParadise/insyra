package quant

import (
	"errors"
	"fmt"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

// FactorModelResult holds a multiple-factor regression of asset excess returns.
// FactorNames and the factor-valued slices use the same table-column order.
type FactorModelResult struct {
	// Alpha is the intercept after subtracting the per-period risk-free rate
	// from the asset return.
	Alpha float64
	// AlphaStdErr is the standard error of Alpha under CovType.
	AlphaStdErr float64
	// AlphaTValue is the t statistic for Alpha under CovType.
	AlphaTValue float64
	// AlphaPValue is the two-tailed p value for Alpha under CovType.
	AlphaPValue float64

	// FactorNames contains factor column names in table order.
	FactorNames []string
	// Exposures contains the OLS coefficient for each factor.
	Exposures []float64
	// StdErrs contains the standard error of each factor exposure under
	// CovType.
	StdErrs []float64
	// TValues contains the t statistic of each factor exposure under CovType.
	TValues []float64
	// PValues contains the two-tailed p value of each factor exposure under
	// CovType.
	PValues []float64

	// RSquared is the coefficient of determination.
	RSquared float64
	// AdjustedRSquared is R-squared adjusted for the number of factors.
	AdjustedRSquared float64
	// N is the number of aligned observations.
	N int
	// Residuals contains one regression residual per observation.
	Residuals []float64

	// CovType is the covariance the standard errors come from; stats.CovOLS,
	// the ordinary OLS standard errors, unless FactorModelOptions chose another.
	CovType stats.CovType
	// Lag is the number of autocovariances stats.CovHAC used, after any
	// automatic choice; it is 0 for the other covariance types.
	Lag int
}

// FactorModelOptions are the optional settings of FactorModel. The zero value
// gives the ordinary OLS standard errors.
type FactorModelOptions struct {
	// CovType chooses the covariance of the coefficients, passed to
	// stats.LinearRegressionWithOptions: stats.CovOLS (the default), the
	// heteroskedasticity-consistent stats.CovHC0 to stats.CovHC3, or the
	// Newey-West stats.CovHAC, which also allows for autocorrelated residuals
	// such as those of daily strategy returns.
	CovType stats.CovType
	// Lag is the number of autocovariances stats.CovHAC includes. 0 chooses
	// stats.NeweyWestLag(n), a positive value is used as it is and must be
	// smaller than n, and a negative value is an error: for no autocorrelation
	// terms use stats.CovHC0. ForwardOptions.NeweyWestLag follows the same three
	// rules. Other covariance types ignore it.
	Lag int
}

func newFloat64DataList(values []float64) *insyra.DataList {
	raw := make([]any, len(values))
	for i, value := range values {
		raw[i] = value
	}
	return insyra.NewDataList(raw...)
}

// Exposure returns the exposure for the named factor. It returns false when
// name is not present. Factor columns are taken as given and are not reduced
// by riskFreeRate; callers must subtract it from a raw market factor before
// passing that factor to FactorModel.
func (r *FactorModelResult) Exposure(name string) (float64, bool) {
	if r == nil {
		return 0, false
	}
	for i, factorName := range r.FactorNames {
		if factorName == name {
			return r.Exposures[i], true
		}
	}
	return 0, false
}

// FactorModel fits a multiple-factor model by regressing asset excess returns
// on every column of factors. riskFreeRate is subtracted from the asset only;
// factor columns are taken as given. For a raw market factor, callers must
// subtract riskFreeRate from that column before passing it here. All values
// are aligned per period and riskFreeRate is a per-period rate.
//
// Standard errors are ordinary OLS ones unless opts chooses a robust or HAC
// covariance; at most one FactorModelOptions may be given. Without opts the
// result is the same as before the options existed.
//
// Returns an error if either input is nil, factors has no columns, a factor
// length differs from the asset length, fewer than k+2 observations are
// supplied for k factors, a cell is non-numeric or non-finite, more than one
// FactorModelOptions is given, or the HAC lag is negative. Errors from
// stats.LinearRegressionWithOptions, including a singular factor matrix, an
// unknown covariance type or a lag not smaller than n, are returned unchanged.
func FactorModel(asset insyra.IDataList, factors insyra.IDataTable, riskFreeRate float64, opts ...FactorModelOptions) (*FactorModelResult, error) {
	if len(opts) > 1 {
		return nil, errors.New("FactorModel: accepts at most one FactorModelOptions")
	}
	var o FactorModelOptions
	if len(opts) == 1 {
		o = opts[0]
	}
	regressionOptions := stats.LinearRegressionOptions{CovType: o.CovType}
	if o.CovType == stats.CovHAC {
		if o.Lag < 0 {
			return nil, fmt.Errorf("FactorModel: HAC lag %d is negative; 0 chooses it automatically, and stats.CovHC0 has no autocorrelation terms", o.Lag)
		}
		regressionOptions.Lag = o.Lag
		if o.Lag == 0 {
			regressionOptions.Lag = stats.AutoLag
		}
	}
	if asset == nil {
		return nil, fmt.Errorf("FactorModel: asset is nil")
	}
	if factors == nil {
		return nil, fmt.Errorf("FactorModel: factors is nil")
	}

	nFactors := factors.NumCols()
	if nFactors == 0 {
		return nil, fmt.Errorf("FactorModel: no factor columns provided")
	}
	factorNames := factors.ColNames()
	assetLength := asset.Len()
	factorColumns := make([]*insyra.DataList, nFactors)
	for i := range nFactors {
		factorColumn := factors.GetColByNumber(i)
		if factorColumn == nil {
			return nil, fmt.Errorf("FactorModel: factor column %q is nil", factorNames[i])
		}
		if factorColumn.Len() != assetLength {
			return nil, fmt.Errorf("FactorModel: factor %q and asset lengths differ (%d vs %d)", factorNames[i], factorColumn.Len(), assetLength)
		}
		factorColumns[i] = factorColumn
	}

	if assetLength < nFactors+2 {
		return nil, fmt.Errorf("FactorModel: need at least %d observations for %d factors, got %d", nFactors+2, nFactors, assetLength)
	}

	assetValues, err := numericSeries(asset, "asset")
	if err != nil {
		return nil, err
	}
	assetExcess := make([]float64, len(assetValues))
	for i, value := range assetValues {
		assetExcess[i] = value - riskFreeRate
	}

	factorValues := make([]insyra.IDataList, nFactors)
	for i, factorColumn := range factorColumns {
		values, err := numericSeries(factorColumn, factorNames[i])
		if err != nil {
			return nil, err
		}
		factorValues[i] = newFloat64DataList(values)
	}

	regression, err := stats.LinearRegressionWithOptions(regressionOptions, newFloat64DataList(assetExcess), factorValues...)
	if err != nil {
		return nil, err
	}

	return &FactorModelResult{
		Alpha:            regression.Coefficients[0],
		AlphaStdErr:      regression.StandardErrors[0],
		AlphaTValue:      regression.TValues[0],
		AlphaPValue:      regression.PValues[0],
		FactorNames:      append([]string(nil), factorNames...),
		Exposures:        append([]float64(nil), regression.Coefficients[1:]...),
		StdErrs:          append([]float64(nil), regression.StandardErrors[1:]...),
		TValues:          append([]float64(nil), regression.TValues[1:]...),
		PValues:          append([]float64(nil), regression.PValues[1:]...),
		RSquared:         regression.RSquared,
		AdjustedRSquared: regression.AdjustedRSquared,
		N:                assetLength,
		Residuals:        append([]float64(nil), regression.Residuals...),
		CovType:          regression.CovType,
		Lag:              regression.Lag,
	}, nil
}
