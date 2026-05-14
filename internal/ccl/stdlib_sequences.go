package ccl

import (
	"fmt"
	"math"
)

// stdlib_sequences.go registers CCL sequence functions (whole-column input,
// same-length-column output). The semantics mirror the matching DataList
// methods in the insyra root package; logic is duplicated here because
// internal/ccl cannot import the parent package.

func init() {
	registerSequenceFunction("LAG", seqLag)
	registerSequenceFunction("LEAD", seqLead)
	registerSequenceFunction("DIFF", seqDiff)
	registerSequenceFunction("PCT_CHANGE", seqPctChange)
	registerSequenceFunction("CUMSUM", seqCumSum)
	registerSequenceFunction("CUMPROD", seqCumProd)
	registerSequenceFunction("CUMMAX", seqCumMax)
	registerSequenceFunction("CUMMIN", seqCumMin)
	registerSequenceFunction("ROLLING_SUM", seqRollingSum)
	registerSequenceFunction("ROLLING_MEAN", seqRollingMean)
	registerSequenceFunction("ROLLING_MIN", seqRollingMin)
	registerSequenceFunction("ROLLING_MAX", seqRollingMax)
	registerSequenceFunction("ROLLING_STD", seqRollingStd)
}

// scalarInt extracts an int scalar from a CCL argument column. The evaluator
// wraps row-independent scalars in a one-element []any; this helper accepts
// that shape and returns the int value or an error.
func scalarInt(arg []any, fnName, paramName string) (int, error) {
	if len(arg) == 0 {
		return 0, fmt.Errorf("%s: %s argument is empty", fnName, paramName)
	}
	if len(arg) > 1 {
		return 0, fmt.Errorf("%s: %s must be a constant, got column of length %d", fnName, paramName, len(arg))
	}
	f, ok := toFloat64(arg[0])
	if !ok {
		return 0, fmt.Errorf("%s: %s must be numeric, got %T", fnName, paramName, arg[0])
	}
	return int(f), nil
}

// =============================================================================
// LAG / LEAD (Shift)
// =============================================================================

func seqShiftImpl(col []any, periods int) []any {
	n := len(col)
	out := make([]any, n)
	switch {
	case periods == 0:
		copy(out, col)
	case periods > 0:
		for i := range n {
			if i < periods {
				out[i] = nil
			} else {
				out[i] = col[i-periods]
			}
		}
	default:
		k := -periods
		for i := range n {
			src := i + k
			if src >= n {
				out[i] = nil
			} else {
				out[i] = col[src]
			}
		}
	}
	return out
}

func seqLag(args ...[]any) ([]any, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("LAG requires 2 arguments (column, periods)")
	}
	periods, err := scalarInt(args[1], "LAG", "periods")
	if err != nil {
		return nil, err
	}
	return seqShiftImpl(args[0], periods), nil
}

func seqLead(args ...[]any) ([]any, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("LEAD requires 2 arguments (column, periods)")
	}
	periods, err := scalarInt(args[1], "LEAD", "periods")
	if err != nil {
		return nil, err
	}
	return seqShiftImpl(args[0], -periods), nil
}

// =============================================================================
// DIFF / PCT_CHANGE
// =============================================================================

func seqDiff(args ...[]any) ([]any, error) {
	if len(args) < 1 || len(args) > 2 {
		return nil, fmt.Errorf("DIFF requires 1 or 2 arguments (column, periods=1)")
	}
	periods := 1
	if len(args) == 2 {
		p, err := scalarInt(args[1], "DIFF", "periods")
		if err != nil {
			return nil, err
		}
		periods = p
	}
	if periods <= 0 {
		return nil, fmt.Errorf("DIFF: periods must be > 0, got %d", periods)
	}
	col := args[0]
	n := len(col)
	out := make([]any, n)
	for i := range n {
		if i < periods {
			out[i] = nil
			continue
		}
		a, okA := toFloat64(col[i])
		b, okB := toFloat64(col[i-periods])
		if !okA || !okB {
			out[i] = nil
			continue
		}
		out[i] = a - b
	}
	return out, nil
}

func seqPctChange(args ...[]any) ([]any, error) {
	if len(args) < 1 || len(args) > 2 {
		return nil, fmt.Errorf("PCT_CHANGE requires 1 or 2 arguments (column, periods=1)")
	}
	periods := 1
	if len(args) == 2 {
		p, err := scalarInt(args[1], "PCT_CHANGE", "periods")
		if err != nil {
			return nil, err
		}
		periods = p
	}
	if periods <= 0 {
		return nil, fmt.Errorf("PCT_CHANGE: periods must be > 0, got %d", periods)
	}
	col := args[0]
	n := len(col)
	out := make([]any, n)
	for i := range n {
		if i < periods {
			out[i] = nil
			continue
		}
		a, okA := toFloat64(col[i])
		b, okB := toFloat64(col[i-periods])
		if !okA || !okB || b == 0 || math.IsNaN(b) {
			out[i] = nil
			continue
		}
		out[i] = (a - b) / b
	}
	return out, nil
}

// =============================================================================
// CUMSUM / CUMPROD / CUMMAX / CUMMIN
// =============================================================================

func seqCumImpl(col []any, initial float64, seedFromFirst bool, combine func(acc, v float64) float64) []any {
	n := len(col)
	out := make([]any, n)
	acc := initial
	seeded := !seedFromFirst
	for i := range n {
		v, ok := toFloat64(col[i])
		if !ok || math.IsNaN(v) {
			out[i] = nil
			continue
		}
		if !seeded {
			acc = v
			seeded = true
		} else {
			acc = combine(acc, v)
		}
		out[i] = acc
	}
	return out
}

func seqCumSum(args ...[]any) ([]any, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("CUMSUM requires 1 argument")
	}
	return seqCumImpl(args[0], 0, false, func(a, v float64) float64 { return a + v }), nil
}

func seqCumProd(args ...[]any) ([]any, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("CUMPROD requires 1 argument")
	}
	return seqCumImpl(args[0], 1, false, func(a, v float64) float64 { return a * v }), nil
}

func seqCumMax(args ...[]any) ([]any, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("CUMMAX requires 1 argument")
	}
	return seqCumImpl(args[0], 0, true, math.Max), nil
}

func seqCumMin(args ...[]any) ([]any, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("CUMMIN requires 1 argument")
	}
	return seqCumImpl(args[0], 0, true, math.Min), nil
}

// =============================================================================
// ROLLING_* (window-aligned right, MinObs = Window)
// =============================================================================

func seqRollingReduce(col []any, window int, fn func(vals []float64) any) []any {
	n := len(col)
	out := make([]any, n)
	for i := range n {
		lo := max(i-window+1, 0)
		hi := i
		var vals []float64
		for j := lo; j <= hi; j++ {
			f, ok := toFloat64(col[j])
			if !ok || math.IsNaN(f) {
				continue
			}
			vals = append(vals, f)
		}
		if len(vals) < window {
			out[i] = nil
			continue
		}
		out[i] = fn(vals)
	}
	return out
}

func rollingArgs(name string, args [][]any) (col []any, window int, err error) {
	if len(args) != 2 {
		return nil, 0, fmt.Errorf("%s requires 2 arguments (column, window)", name)
	}
	w, err := scalarInt(args[1], name, "window")
	if err != nil {
		return nil, 0, err
	}
	if w <= 0 {
		return nil, 0, fmt.Errorf("%s: window must be > 0, got %d", name, w)
	}
	return args[0], w, nil
}

func seqRollingSum(args ...[]any) ([]any, error) {
	col, w, err := rollingArgs("ROLLING_SUM", args)
	if err != nil {
		return nil, err
	}
	return seqRollingReduce(col, w, func(vals []float64) any {
		var s float64
		for _, v := range vals {
			s += v
		}
		return s
	}), nil
}

func seqRollingMean(args ...[]any) ([]any, error) {
	col, w, err := rollingArgs("ROLLING_MEAN", args)
	if err != nil {
		return nil, err
	}
	return seqRollingReduce(col, w, func(vals []float64) any {
		var s float64
		for _, v := range vals {
			s += v
		}
		return s / float64(len(vals))
	}), nil
}

func seqRollingMin(args ...[]any) ([]any, error) {
	col, w, err := rollingArgs("ROLLING_MIN", args)
	if err != nil {
		return nil, err
	}
	return seqRollingReduce(col, w, func(vals []float64) any {
		m := vals[0]
		for _, v := range vals[1:] {
			if v < m {
				m = v
			}
		}
		return m
	}), nil
}

func seqRollingMax(args ...[]any) ([]any, error) {
	col, w, err := rollingArgs("ROLLING_MAX", args)
	if err != nil {
		return nil, err
	}
	return seqRollingReduce(col, w, func(vals []float64) any {
		m := vals[0]
		for _, v := range vals[1:] {
			if v > m {
				m = v
			}
		}
		return m
	}), nil
}

func seqRollingStd(args ...[]any) ([]any, error) {
	col, w, err := rollingArgs("ROLLING_STD", args)
	if err != nil {
		return nil, err
	}
	return seqRollingReduce(col, w, func(vals []float64) any {
		if len(vals) < 2 {
			return nil
		}
		var sum float64
		for _, v := range vals {
			sum += v
		}
		mean := sum / float64(len(vals))
		var ss float64
		for _, v := range vals {
			d := v - mean
			ss += d * d
		}
		return math.Sqrt(ss / float64(len(vals)-1))
	}), nil
}

