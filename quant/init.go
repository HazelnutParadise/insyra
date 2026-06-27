// Package quant provides quantitative-finance tools for evaluating
// trading strategies and portfolios: performance metrics (Sharpe ratio,
// maximum drawdown, annualized return), backtest-overfitting diagnostics
// (CSCV PBO, Deflated Sharpe Ratio), and time-series walk-forward
// out-of-sample validation.
//
// Unlike the finance package (which uses high-precision decimals for TVM,
// NPV/IRR, and bond pricing), quant operates on plain float64 series — the
// industry convention for return/equity analytics, where statistical noise
// dwarfs floating-point error. Inputs are []float64 (a return series or an
// equity/NAV curve); exported functions follow an error-first convention,
// returning an error for invalid input rather than logging or panicking.
package quant

func init() {}
