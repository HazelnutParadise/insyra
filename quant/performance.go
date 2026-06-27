package quant

import (
	"errors"
	"math"

	"github.com/HazelnutParadise/insyra"
	"gonum.org/v1/gonum/stat"
)

// daysPerYear is the calendar-day basis used to annualize returns over a
// date span. Calendar days (not trading days) are used because the span
// passed to AnnualizedReturn is wall-clock elapsed time; 365 is the CAGR
// convention.
const daysPerYear = 365.0

// SharpeRatio returns the annualized Sharpe ratio of a periodic return
// series:
//
//	Sharpe = mean(returns - riskFreeRate) / stddev(returns) · √periodsPerYear
//
// returns are per-period simple returns (e.g. daily returns as 0.012 for
// +1.2%). riskFreeRate is the risk-free rate expressed in the SAME period
// as returns (pass 0 for an excess-return series or to ignore it).
// periodsPerYear is the annualization factor — 252 for daily Taiwan-stock
// returns, 12 for monthly, 52 for weekly. Pass periodsPerYear = 1 to obtain
// the per-period (non-annualized) Sharpe used by the overfitting diagnostics.
//
// The standard deviation is the sample (n-1) standard deviation, matching
// the common convention used by most backtesting tools and by gonum.
//
// Returns an error if fewer than 2 returns are given, periodsPerYear is
// non-positive, or the return series has zero volatility (Sharpe undefined).
func SharpeRatio(returns insyra.IDataList, riskFreeRate, periodsPerYear float64) (float64, error) {
	return sharpeRatioF64(returns.ToF64Slice(), riskFreeRate, periodsPerYear)
}

func sharpeRatioF64(returns []float64, riskFreeRate, periodsPerYear float64) (float64, error) {
	if len(returns) < 2 {
		return math.NaN(), errors.New("SharpeRatio: need at least 2 returns")
	}
	if periodsPerYear <= 0 {
		return math.NaN(), errors.New("SharpeRatio: periodsPerYear must be positive")
	}

	excess := make([]float64, len(returns))
	for i, r := range returns {
		excess[i] = r - riskFreeRate
	}

	mean := stat.Mean(excess, nil)
	sd := stat.StdDev(excess, nil) // sample stddev (ddof = 1)
	if sd == 0 {
		return math.NaN(), errors.New("SharpeRatio: zero volatility (standard deviation is 0)")
	}

	return mean / sd * math.Sqrt(periodsPerYear), nil
}

// MaxDrawdown returns the maximum drawdown of an equity (cumulative
// value / NAV) curve as a non-negative fraction: 0.2 means the curve fell
// 20% below a prior running peak at its worst point. A monotonically
// non-decreasing curve has a drawdown of 0.
//
// equity should be a positive value series; points where the running peak
// is non-positive are skipped (drawdown is undefined there).
//
// Returns an error if equity is empty.
func MaxDrawdown(equity insyra.IDataList) (float64, error) {
	return maxDrawdownF64(equity.ToF64Slice())
}

func maxDrawdownF64(equity []float64) (float64, error) {
	if len(equity) == 0 {
		return math.NaN(), errors.New("MaxDrawdown: equity is empty")
	}

	peak := math.Inf(-1)
	maxDD := 0.0
	for _, v := range equity {
		if v > peak {
			peak = v
		}
		if peak > 0 {
			if dd := (peak - v) / peak; dd > maxDD {
				maxDD = dd
			}
		}
	}
	return maxDD, nil
}

// AnnualizedReturn returns the annualized (CAGR-style) return implied by
// an equity curve spanning days calendar days:
//
//	(equity[last] / equity[0]) ^ (365 / days) - 1
//
// equity is a value/NAV curve (only its first and last points matter);
// days is the calendar-day span the curve covers.
//
// Returns an error if fewer than 2 points are given, days is non-positive,
// or the first/last equity value is non-positive (the growth ratio would
// be undefined).
func AnnualizedReturn(equity insyra.IDataList, days int) (float64, error) {
	return annualizedReturnF64(equity.ToF64Slice(), days)
}

func annualizedReturnF64(equity []float64, days int) (float64, error) {
	if len(equity) < 2 {
		return math.NaN(), errors.New("AnnualizedReturn: need at least 2 equity points")
	}
	if days <= 0 {
		return math.NaN(), errors.New("AnnualizedReturn: days must be positive")
	}

	begin := equity[0]
	end := equity[len(equity)-1]
	if begin <= 0 {
		return math.NaN(), errors.New("AnnualizedReturn: initial equity must be positive")
	}
	if end <= 0 {
		return math.NaN(), errors.New("AnnualizedReturn: final equity must be positive")
	}

	growth := end / begin
	return math.Pow(growth, daysPerYear/float64(days)) - 1, nil
}
