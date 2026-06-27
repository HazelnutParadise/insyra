package quant

import (
	"errors"
	"fmt"
)

// WalkForwardConfig configures the sliding train/test windows of a
// walk-forward analysis.
type WalkForwardConfig struct {
	// TrainSize is the number of in-sample periods used to pick parameters
	// for each fold.
	TrainSize int
	// TestSize is the number of out-of-sample periods evaluated per fold.
	// Folds advance by TestSize, so out-of-sample windows never overlap.
	TestSize int
	// Anchored controls the training window: false (default) uses a fixed
	// TrainSize rolling window; true uses an expanding window anchored at
	// period 0 (training always starts at 0 and grows each fold).
	Anchored bool
}

// WalkForwardFold records the index ranges and out-of-sample result of one
// fold. All ranges are half-open [Start, End).
type WalkForwardFold struct {
	TrainStart int
	TrainEnd   int
	TestStart  int
	TestEnd    int
	// OOSReturns is the per-period out-of-sample return series returned by
	// evaluate for this fold.
	OOSReturns []float64
}

// WalkForwardResult aggregates every fold of a walk-forward run.
type WalkForwardResult struct {
	Folds []WalkForwardFold
	// OOSReturns concatenates the out-of-sample returns of every fold in
	// chronological order — the stitched out-of-sample track record.
	OOSReturns []float64
	// Equity is the compounded out-of-sample equity curve starting at 1.0,
	// so len(Equity) == len(OOSReturns)+1.
	Equity []float64
}

// WalkForward runs a time-series walk-forward (out-of-sample) validation
// over n periods. For each fold it calls optimize on the in-sample window
// to pick parameters of type P, then evaluate on the out-of-sample window
// to obtain that fold's per-period returns; the out-of-sample returns are
// stitched together and compounded into a single equity curve. This is the
// standard guard against optimizing and evaluating on the same data (Pardo).
//
// Both callbacks receive half-open [start, end) index ranges into the
// caller's own data (the caller closes over the actual series), so any data
// layout works. evaluate should return one return per out-of-sample period
// (typically TestEnd-TestStart values).
//
// Windows advance by TestSize starting at TrainSize. With a rolling window
// fold k is train [TestStart-TrainSize, TestStart), test [TestStart,
// TestStart+TestSize); Anchored fixes the training start at 0 (expanding).
// If n-TrainSize is not a multiple of TestSize, the final out-of-sample
// window is shorter than TestSize rather than dropped, so all data is used.
//
// Returns an error if n, TrainSize, or TestSize is non-positive, TrainSize
// leaves no room for testing (TrainSize >= n), or either callback is nil.
func WalkForward[P any](
	n int,
	cfg WalkForwardConfig,
	optimize func(trainStart, trainEnd int) P,
	evaluate func(p P, testStart, testEnd int) []float64,
) (*WalkForwardResult, error) {
	if n <= 0 {
		return nil, errors.New("WalkForward: n must be positive")
	}
	if cfg.TrainSize <= 0 || cfg.TestSize <= 0 {
		return nil, errors.New("WalkForward: TrainSize and TestSize must be positive")
	}
	if cfg.TrainSize >= n {
		return nil, fmt.Errorf("WalkForward: TrainSize (%d) leaves no room for testing in %d periods", cfg.TrainSize, n)
	}
	if optimize == nil || evaluate == nil {
		return nil, errors.New("WalkForward: optimize and evaluate must be non-nil")
	}

	res := &WalkForwardResult{Equity: []float64{1.0}}
	equity := 1.0
	for testStart := cfg.TrainSize; testStart < n; testStart += cfg.TestSize {
		testEnd := min(testStart+cfg.TestSize, n)
		trainStart := 0
		if !cfg.Anchored {
			trainStart = testStart - cfg.TrainSize
		}
		trainEnd := testStart

		p := optimize(trainStart, trainEnd)
		oos := evaluate(p, testStart, testEnd)

		res.Folds = append(res.Folds, WalkForwardFold{
			TrainStart: trainStart,
			TrainEnd:   trainEnd,
			TestStart:  testStart,
			TestEnd:    testEnd,
			OOSReturns: oos,
		})
		for _, r := range oos {
			res.OOSReturns = append(res.OOSReturns, r)
			equity *= 1 + r
			res.Equity = append(res.Equity, equity)
		}
	}
	return res, nil
}

// Sharpe returns the annualized Sharpe ratio of the stitched out-of-sample
// returns. See SharpeRatio for the parameters and conventions.
func (r *WalkForwardResult) Sharpe(riskFreeRate, periodsPerYear float64) (float64, error) {
	return sharpeRatioF64(r.OOSReturns, riskFreeRate, periodsPerYear)
}

// MaxDrawdown returns the maximum drawdown of the out-of-sample equity
// curve. See MaxDrawdown for details.
func (r *WalkForwardResult) MaxDrawdown() (float64, error) {
	return maxDrawdownF64(r.Equity)
}

// AnnualizedReturn returns the annualized (CAGR-style) return of the
// out-of-sample equity curve over the given calendar-day span. See
// AnnualizedReturn for details.
func (r *WalkForwardResult) AnnualizedReturn(days int) (float64, error) {
	return annualizedReturnF64(r.Equity, days)
}
