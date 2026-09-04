package quant

import (
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
	// AlphaStdErr is the OLS standard error of Alpha.
	AlphaStdErr float64
	// AlphaTValue is the OLS t statistic for Alpha.
	AlphaTValue float64
	// AlphaPValue is the OLS p value for Alpha.
	AlphaPValue float64

	// FactorNames contains factor column names in table order.
	FactorNames []string
	// Exposures contains the OLS coefficient for each factor.
	Exposures []float64
	// StdErrs contains the OLS standard error for each factor exposure.
	StdErrs []float64
	// TValues contains the OLS t statistic for each factor exposure.
	TValues []float64
	// PValues contains the OLS p value for each factor exposure.
	PValues []float64

	// RSquared is the coefficient of determination.
	RSquared float64
	// AdjustedRSquared is R-squared adjusted for the number of factors.
	AdjustedRSquared float64
	// N is the number of aligned observations.
	N int
	// Residuals contains one regression residual per observation.
	Residuals []float64
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
// Returns an error if either input is nil, factors has no columns, a factor
// length differs from the asset length, fewer than k+2 observations are
// supplied for k factors, or a cell is non-numeric or non-finite. Errors from
// stats.LinearRegression, including a singular factor matrix, are returned
// unchanged.
func FactorModel(asset insyra.IDataList, factors insyra.IDataTable, riskFreeRate float64) (*FactorModelResult, error) {
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

	regression, err := stats.LinearRegression(newFloat64DataList(assetExcess), factorValues...)
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
	}, nil
}
