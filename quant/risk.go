package quant

import (
	"errors"
	"fmt"
	"math"
	"sort"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
	"gonum.org/v1/gonum/stat"
)

// VaRMethod selects the distribution used to calculate value at risk and
// conditional value at risk.
type VaRMethod uint8

const (
	// VaRHistorical uses the empirical R type-7 return quantile. At 0.95
	// confidence it negates the 5th percentile of returns.
	VaRHistorical VaRMethod = iota
	// VaRParametric assumes normally distributed returns and uses their sample
	// mean and sample standard deviation.
	VaRParametric
)

// ValueAtRisk returns the loss fraction exceeded with probability
// 1-confidence, as a positive loss under the package's sign convention. For
// example, confidence 0.95 uses the 5th percentile and returns its negation.
// VaRHistorical uses the empirical R type-7 quantile; VaRParametric uses
// -(mean + NormPPF(1-confidence)*sample standard deviation).
//
// Returns an error if confidence is not in (0, 1), method is unknown, fewer
// than 2 returns are supplied, or a return is unreadable or non-finite.
func ValueAtRisk(returns insyra.IDataList, confidence float64, method VaRMethod) (float64, error) {
	values, err := numericSeries(returns, "returns")
	if err != nil {
		return math.NaN(), err
	}
	return valueAtRiskF64(values, confidence, method)
}

func valueAtRiskF64(returns []float64, confidence float64, method VaRMethod) (float64, error) {
	if err := validateVaRInputs(returns, confidence, method); err != nil {
		return math.NaN(), err
	}

	mean := stat.Mean(returns, nil)
	sd := stat.StdDev(returns, nil)
	switch method {
	case VaRHistorical:
		sorted := append([]float64(nil), returns...)
		sort.Float64s(sorted)
		return -quantileType7(sorted, 1-confidence), nil
	case VaRParametric:
		z, err := stats.NormPPF(1 - confidence)
		if err != nil {
			return math.NaN(), err
		}
		return -(mean + z*sd), nil
	default:
		return math.NaN(), fmt.Errorf("ValueAtRisk: unknown method %d", method)
	}
}

// ConditionalValueAtRisk returns the expected loss conditional on being in
// the loss tail named by confidence. It uses the same positive-loss sign
// convention and method definitions as ValueAtRisk. Historical CVaR is the
// negated mean of returns at or below the historical VaR quantile;
// parametric CVaR is -(mean - sample standard deviation*φ(z)/(1-confidence)).
//
// Returns an error if confidence is not in (0, 1), method is unknown, fewer
// than 2 returns are supplied, or a return is unreadable or non-finite.
func ConditionalValueAtRisk(returns insyra.IDataList, confidence float64, method VaRMethod) (float64, error) {
	values, err := numericSeries(returns, "returns")
	if err != nil {
		return math.NaN(), err
	}
	return conditionalValueAtRiskF64(values, confidence, method)
}

func conditionalValueAtRiskF64(returns []float64, confidence float64, method VaRMethod) (float64, error) {
	if err := validateVaRInputs(returns, confidence, method); err != nil {
		return math.NaN(), err
	}

	mean := stat.Mean(returns, nil)
	sd := stat.StdDev(returns, nil)
	switch method {
	case VaRHistorical:
		sorted := append([]float64(nil), returns...)
		sort.Float64s(sorted)
		quantile := quantileType7(sorted, 1-confidence)
		tailSum := 0.0
		tailCount := 0
		for _, value := range returns {
			if value <= quantile {
				tailSum += value
				tailCount++
			}
		}
		return -(tailSum / float64(tailCount)), nil
	case VaRParametric:
		z, err := stats.NormPPF(1 - confidence)
		if err != nil {
			return math.NaN(), err
		}
		return -(mean - sd*normPDF(z)/(1-confidence)), nil
	default:
		return math.NaN(), fmt.Errorf("ConditionalValueAtRisk: unknown method %d", method)
	}
}

func validateVaRInputs(returns []float64, confidence float64, method VaRMethod) error {
	if len(returns) < 2 {
		return fmt.Errorf("VaR: need at least 2 returns, got %d", len(returns))
	}
	if !(confidence > 0 && confidence < 1) {
		return fmt.Errorf("VaR: confidence must be in (0, 1), got %v", confidence)
	}
	if method != VaRHistorical && method != VaRParametric {
		return fmt.Errorf("VaR: unknown method %d", method)
	}
	return nil
}

func normPDF(x float64) float64 {
	return math.Exp(-0.5*x*x) / math.Sqrt(2*math.Pi)
}

// SortinoRatio returns the annualized Sortino ratio of a periodic return
// series. The downside deviation is the root mean square of the negative
// shortfalls from minimumAcceptableReturn over all periods, including periods
// without a shortfall:
//
//	Sortino = mean(returns-MAR) / sqrt(mean(min(returns-MAR, 0)^2)) · √periodsPerYear
//
// Returns an error if fewer than 2 returns are supplied, periodsPerYear is not
// positive, or downside deviation is zero.
func SortinoRatio(returns insyra.IDataList, minimumAcceptableReturn, periodsPerYear float64) (float64, error) {
	values, err := numericSeries(returns, "returns")
	if err != nil {
		return math.NaN(), err
	}
	return sortinoRatioF64(values, minimumAcceptableReturn, periodsPerYear)
}

func sortinoRatioF64(returns []float64, minimumAcceptableReturn, periodsPerYear float64) (float64, error) {
	if len(returns) < 2 {
		return math.NaN(), errors.New("SortinoRatio: need at least 2 returns")
	}
	if !(periodsPerYear > 0) || math.IsInf(periodsPerYear, 0) {
		return math.NaN(), errors.New("SortinoRatio: periodsPerYear must be positive and finite")
	}
	if math.IsNaN(minimumAcceptableReturn) || math.IsInf(minimumAcceptableReturn, 0) {
		return math.NaN(), errors.New("SortinoRatio: minimumAcceptableReturn must be finite")
	}

	mean := 0.0
	downsideSquared := 0.0
	for _, value := range returns {
		excess := value - minimumAcceptableReturn
		mean += excess
		if excess < 0 {
			downsideSquared += excess * excess
		}
	}
	mean /= float64(len(returns))
	downsideDeviation := math.Sqrt(downsideSquared / float64(len(returns)))
	if downsideDeviation == 0 {
		return math.NaN(), errors.New("SortinoRatio: downside deviation is 0")
	}
	return mean / downsideDeviation * math.Sqrt(periodsPerYear), nil
}

// CalmarRatio returns annualized return divided by maximum drawdown for an
// equity curve. The annualized return uses the calendar-day span in days and
// the same drawdown definition as AnnualizedReturn and MaxDrawdown.
//
// Returns an error if the existing annualized-return or maximum-drawdown
// validation fails, or if maximum drawdown is zero.
func CalmarRatio(equity insyra.IDataList, days int) (float64, error) {
	values, err := numericSeries(equity, "equity")
	if err != nil {
		return math.NaN(), err
	}
	return calmarRatioF64(values, days)
}

func calmarRatioF64(equity []float64, days int) (float64, error) {
	annualized, err := annualizedReturnF64(equity, days)
	if err != nil {
		return math.NaN(), err
	}
	drawdown, err := maxDrawdownF64(equity)
	if err != nil {
		return math.NaN(), err
	}
	if drawdown == 0 {
		return math.NaN(), errors.New("CalmarRatio: maximum drawdown is 0")
	}
	return annualized / drawdown, nil
}

// InformationRatio returns the annualized mean active return divided by the
// sample standard deviation of active returns. Active return is returns minus
// benchmark for each aligned period.
//
// Returns an error if the series lengths differ, fewer than 2 observations are
// supplied, periodsPerYear is not positive, or tracking error is zero.
func InformationRatio(returns, benchmark insyra.IDataList, periodsPerYear float64) (float64, error) {
	returnValues, err := numericSeries(returns, "returns")
	if err != nil {
		return math.NaN(), err
	}
	benchmarkValues, err := numericSeries(benchmark, "benchmark")
	if err != nil {
		return math.NaN(), err
	}
	return informationRatioF64(returnValues, benchmarkValues, periodsPerYear)
}

func informationRatioF64(returns, benchmark []float64, periodsPerYear float64) (float64, error) {
	if len(returns) != len(benchmark) {
		return math.NaN(), fmt.Errorf("InformationRatio: returns and benchmark lengths differ (%d vs %d)", len(returns), len(benchmark))
	}
	if len(returns) < 2 {
		return math.NaN(), errors.New("InformationRatio: need at least 2 observations")
	}
	if !(periodsPerYear > 0) || math.IsInf(periodsPerYear, 0) {
		return math.NaN(), errors.New("InformationRatio: periodsPerYear must be positive and finite")
	}

	active := make([]float64, len(returns))
	for i := range active {
		active[i] = returns[i] - benchmark[i]
	}
	trackingError := stat.StdDev(active, nil)
	if trackingError == 0 {
		return math.NaN(), errors.New("InformationRatio: tracking error is 0")
	}
	return stat.Mean(active, nil) / trackingError * math.Sqrt(periodsPerYear), nil
}

// DrawdownSeries returns the per-period non-negative drawdown from the
// running equity peak. Positions whose running peak is non-positive contain
// nil because their drawdown is undefined. Its maximum equals MaxDrawdown for
// the same equity curve.
//
// Returns an error if equity is empty or contains an unreadable or non-finite
// value.
func DrawdownSeries(equity insyra.IDataList) (*insyra.DataList, error) {
	values, err := numericSeries(equity, "equity")
	if err != nil {
		return nil, err
	}
	series, err := drawdownSeriesF64(values)
	if err != nil {
		return nil, err
	}
	return insyra.NewDataList(series...), nil
}

func drawdownSeriesF64(equity []float64) ([]any, error) {
	if len(equity) == 0 {
		return nil, errors.New("DrawdownSeries: equity is empty")
	}

	series := make([]any, len(equity))
	peak := math.Inf(-1)
	for i, value := range equity {
		if value > peak {
			peak = value
		}
		if peak > 0 {
			series[i] = (peak - value) / peak
		}
	}
	return series, nil
}
