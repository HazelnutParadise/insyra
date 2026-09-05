// Package quant provides quantitative-finance tools for evaluating
// trading strategies and portfolios: performance and risk metrics (Sharpe,
// Sortino, Calmar, VaR, CVaR, maximum drawdown, annualized return, and
// information ratio), backtest-overfitting diagnostics (CSCV PBO, Deflated
// Sharpe Ratio), time-series walk-forward out-of-sample validation, and
// probabilistic forecasting by block bootstrap (moving block or stationary)
// with percentile bands, plus market beta, CAPM, and multi-factor exposure
// analysis, European option pricing with Black-Scholes-Merton greeks, and
// implied volatility.
//
// Unlike the finance package (which uses high-precision decimals for TVM,
// NPV/IRR, and bond pricing), quant operates on plain float64 series — the
// industry convention for return/equity analytics, where statistical noise
// dwarfs floating-point error. Inputs are []float64 (a return series or an
// equity/NAV curve); exported functions follow an error-first convention,
// returning an error for invalid input rather than logging or panicking.
package quant

func init() {}
