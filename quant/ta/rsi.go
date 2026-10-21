package ta

import (
	"errors"
	"fmt"
	"math"

	"github.com/HazelnutParadise/insyra"
)

// RSIMethod chooses how RSI averages gains and losses.
type RSIMethod int

const (
	// RSIWilder smooths with an exponential average of weight 1/n, Wilder's
	// original definition (1978) and the default of most software.
	RSIWilder RSIMethod = iota
	// RSISimple takes the plain mean of the last n gains and losses, Cutler's
	// version, which some textbooks use.
	RSISimple
)

// RSIMissing chooses what RSI does with a price that is nil or NaN.
type RSIMissing int

const (
	// RSIMissingError refuses a missing price.
	RSIMissingError RSIMissing = iota
	// RSIMissingFlat carries the last price over, so the missing day counts as
	// a change of 0. The missing day itself is NaN. Missing prices before the
	// first real one are NaN and the series starts at the first real price.
	RSIMissingFlat
	// RSIMissingBreak ends the series at a missing price and starts a new one
	// after it, which needs n prices of its own before it shows a value. The
	// missing day is NaN.
	RSIMissingBreak
)

// String returns the constant name of m, or RSIMethod(n) for a value that is
// not one of the constants.
func (m RSIMethod) String() string {
	switch m {
	case RSIWilder:
		return "RSIWilder"
	case RSISimple:
		return "RSISimple"
	}
	return fmt.Sprintf("RSIMethod(%d)", int(m))
}

// String returns the constant name of m, or RSIMissing(n) for a value that is
// not one of the constants.
func (m RSIMissing) String() string {
	switch m {
	case RSIMissingError:
		return "RSIMissingError"
	case RSIMissingFlat:
		return "RSIMissingFlat"
	case RSIMissingBreak:
		return "RSIMissingBreak"
	}
	return fmt.Sprintf("RSIMissing(%d)", int(m))
}

// RSIOptions are the optional settings of RSI. The zero value is Wilder's
// average and an error on a missing price.
type RSIOptions struct {
	Method  RSIMethod
	Missing RSIMissing
}

// RSI returns the relative strength index of a price series over n days, in
// [0, 100]: 100 * avgGain / (avgGain + avgLoss), where the averages come from
// the day-to-day changes in price and are taken as RSIOptions.Method says.
// The first n values, which lack n changes, are NaN. A stretch of n changes with
// no gain and no loss has no defined RSI and is NaN, and one with gains only is
// 100.
func RSI(prices insyra.IDataList, n int, opts ...RSIOptions) (*insyra.DataList, error) {
	if n < 1 {
		return nil, errors.New("n must be at least 1")
	}
	if len(opts) > 1 {
		return nil, errors.New("RSI accepts at most one RSIOptions")
	}
	var o RSIOptions
	if len(opts) == 1 {
		o = opts[0]
	}
	if o.Method != RSIWilder && o.Method != RSISimple {
		return nil, fmt.Errorf("unsupported RSI method %v", o.Method)
	}
	if o.Missing != RSIMissingError && o.Missing != RSIMissingFlat && o.Missing != RSIMissingBreak {
		return nil, fmt.Errorf("unsupported RSI missing-value policy %v", o.Missing)
	}
	if insyra.IsNilList(prices) {
		return nil, errors.New("prices is nil")
	}
	var raw []any
	prices.AtomicDo(func(l *insyra.DataList) { raw = l.Data() })

	values := make([]float64, len(raw))
	missing := make([]bool, len(raw))
	for i, v := range raw {
		if v == nil {
			missing[i] = true
			continue
		}
		f, ok := insyra.ToFloat64Safe(v)
		if !ok {
			return nil, fmt.Errorf("prices contains a non-numeric value at row index %d: %v", i, v)
		}
		if math.IsInf(f, 0) {
			return nil, fmt.Errorf("prices contains a non-finite value at row index %d: %v", i, f)
		}
		if math.IsNaN(f) {
			missing[i] = true
			continue
		}
		values[i] = f
	}
	if o.Missing == RSIMissingError {
		for i, m := range missing {
			if m {
				return nil, fmt.Errorf("prices is missing a value at row index %d; choose RSIMissingFlat or RSIMissingBreak to allow gaps", i)
			}
		}
	}

	out := make([]float64, len(values))
	for i := range out {
		out[i] = math.NaN()
	}
	switch o.Missing {
	case RSIMissingFlat:
		first := -1
		for i, m := range missing {
			if !m {
				first = i
				break
			}
		}
		if first >= 0 {
			filled := append([]float64(nil), values[first:]...)
			for i := 1; i < len(filled); i++ {
				if missing[first+i] {
					filled[i] = filled[i-1]
				}
			}
			for i, r := range rsiRun(filled, n, o.Method) {
				if !missing[first+i] {
					out[first+i] = r
				}
			}
		}
	default:
		// RSIMissingError has no gaps left, so this is one run; RSIMissingBreak
		// computes each stretch between missing prices on its own.
		for start := 0; start < len(values); {
			if missing[start] {
				start++
				continue
			}
			end := start
			for end < len(values) && !missing[end] {
				end++
			}
			copy(out[start:end], rsiRun(values[start:end], n, o.Method))
			start = end
		}
	}

	result := make([]any, len(out))
	for i, v := range out {
		result[i] = v
	}
	return insyra.NewDataList(result...), nil
}

// rsiRun computes RSI over one stretch of prices without gaps. The first n
// values are NaN.
func rsiRun(p []float64, n int, method RSIMethod) []float64 {
	out := make([]float64, len(p))
	for i := range out {
		out[i] = math.NaN()
	}
	if len(p) <= n {
		return out
	}
	gain := make([]float64, len(p))
	loss := make([]float64, len(p))
	for t := 1; t < len(p); t++ {
		change := p[t] - p[t-1]
		gain[t], loss[t] = math.Max(change, 0), math.Max(-change, 0)
	}
	fn := float64(n)
	avgGain, avgLoss := 0.0, 0.0
	for t := 1; t <= n; t++ {
		avgGain += gain[t]
		avgLoss += loss[t]
	}
	avgGain, avgLoss = avgGain/fn, avgLoss/fn
	out[n] = rsiValue(avgGain, avgLoss)
	for t := n + 1; t < len(p); t++ {
		if method == RSIWilder {
			avgGain = (avgGain*(fn-1) + gain[t]) / fn
			avgLoss = (avgLoss*(fn-1) + loss[t]) / fn
		} else {
			// Summed afresh, since a running sum drifts and would leave a
			// flat stretch with a tiny nonzero average.
			avgGain, avgLoss = 0, 0
			for i := t - n + 1; i <= t; i++ {
				avgGain += gain[i]
				avgLoss += loss[i]
			}
			avgGain, avgLoss = avgGain/fn, avgLoss/fn
		}
		out[t] = rsiValue(avgGain, avgLoss)
	}
	return out
}

func rsiValue(avgGain, avgLoss float64) float64 {
	if avgGain+avgLoss == 0 {
		return math.NaN()
	}
	return 100 * avgGain / (avgGain + avgLoss)
}
