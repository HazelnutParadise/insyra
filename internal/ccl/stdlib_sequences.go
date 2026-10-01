package ccl

import (
	"fmt"
	"math"
	"strings"
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

// rowCapableSequenceFunctions lists the sequence functions that only move
// values around and never do arithmetic on them, so they can take a whole row
// per element — `LAG(@, 1)` gives every row the row before it. Every other
// sequence function needs numbers, and a row is never one. Adding a sequence
// function means deciding which of the two it is.
var rowCapableSequenceFunctions = map[string]bool{
	"LAG":  true,
	"LEAD": true,
}

// SequenceFunctionTakesRows reports whether name accepts a row-shaped column
// ('@' or a column range) instead of a column of values.
func SequenceFunctionTakesRows(name string) bool {
	return rowCapableSequenceFunctions[strings.ToUpper(name)]
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
	// int(f) on NaN, ±Inf or a value past the int range is undefined and
	// differs by platform; anything beyond int32 is never a sane shift or
	// window, so refuse it before it can index a slice with garbage.
	if math.IsNaN(f) || math.IsInf(f, 0) || f > math.MaxInt32 || f < math.MinInt32 {
		return 0, fmt.Errorf("%s: %s %v is out of range", fnName, paramName, arg[0])
	}
	// A fractional window or period silently became its floor, so
	// ROLLING_MEAN(A, 2.9) averaged over two rows and said nothing.
	if f != math.Trunc(f) {
		return 0, fmt.Errorf("%s: %s must be a whole number, got %v", fnName, paramName, arg[0])
	}
	return int(f), nil
}

// =============================================================================
// Windowed sequences: LAG, LEAD, DIFF, PCT_CHANGE and ROLLING_*
// =============================================================================

// windowedSequence computes a sequence function for the positions from up to, not
// including, to of col, given the params after the column as the function reads
// them, and returns to - from outputs. A position reads the rows of the whole col
// whatever from is, so its output is the one the function gives it on the whole
// column. The params have the length the function's own argument check lets
// through.
type windowedSequence func(col []any, params [][]any, from, to int) ([]any, error)

// windowedSequences computes each windowed sequence function from a position
// onwards, given its params as the function reads them; the functions and their
// streams both go through it, so the two run one piece of arithmetic and cannot
// drift apart. The table also holds the params' meaning: how a param is read and
// which values of it are refused.
var windowedSequences = map[string]windowedSequence{
	"LAG":          windowedShift("LAG", 1),
	"LEAD":         windowedShift("LEAD", -1),
	"DIFF":         windowedChange("DIFF", diffRange),
	"PCT_CHANGE":   windowedChange("PCT_CHANGE", pctChangeRange),
	"ROLLING_SUM":  windowedRolling("ROLLING_SUM", rollingSum),
	"ROLLING_MEAN": windowedRolling("ROLLING_MEAN", rollingMean),
	"ROLLING_MIN":  windowedRolling("ROLLING_MIN", rollingMin),
	"ROLLING_MAX":  windowedRolling("ROLLING_MAX", rollingMax),
	"ROLLING_STD":  windowedRolling("ROLLING_STD", rollingStd),
}

// windowedShift is LAG for a direction of 1 and LEAD for -1: LEAD is the shift by
// the negative of the periods it is given.
func windowedShift(name string, direction int) windowedSequence {
	return func(col []any, params [][]any, from, to int) ([]any, error) {
		periods, err := scalarInt(params[0], name, "periods")
		if err != nil {
			return nil, err
		}
		return shiftRange(col, direction*periods, from, to), nil
	}
}

// windowedChange is DIFF or PCT_CHANGE, which take an optional periods that
// defaults to 1 and must be positive.
func windowedChange(name string, compute func(col []any, periods, from, to int) []any) windowedSequence {
	return func(col []any, params [][]any, from, to int) ([]any, error) {
		periods := 1
		if len(params) > 0 {
			p, err := scalarInt(params[0], name, "periods")
			if err != nil {
				return nil, err
			}
			periods = p
		}
		if periods <= 0 {
			return nil, fmt.Errorf("%s: periods must be > 0, got %d", name, periods)
		}
		return compute(col, periods, from, to), nil
	}
}

// windowedRolling is one of the ROLLING_* functions: the window is its one param
// and fn reduces the values of a complete window.
func windowedRolling(name string, fn func(vals []float64) any) windowedSequence {
	return func(col []any, params [][]any, from, to int) ([]any, error) {
		window, err := scalarInt(params[0], name, "window")
		if err != nil {
			return nil, err
		}
		if window <= 0 {
			return nil, fmt.Errorf("%s: window must be > 0, got %d", name, window)
		}
		return rollingReduceRange(col, window, from, to, fn), nil
	}
}

// =============================================================================
// LAG / LEAD (Shift)
// =============================================================================

// shiftRange returns the outputs of a shift by periods for positions from up
// to, not including, to: out[i-from] is col[i-periods] when that index is inside
// col, nil otherwise. A shift of the whole column is
// shiftRange(col, periods, 0, len(col)).
func shiftRange(col []any, periods, from, to int) []any {
	n := len(col)
	out := make([]any, to-from)
	switch {
	case periods == 0:
		copy(out, col[from:to])
	case periods > 0:
		for i := from; i < to; i++ {
			if i < periods {
				out[i-from] = nil
			} else {
				out[i-from] = col[i-periods]
			}
		}
	default:
		k := -periods
		for i := from; i < to; i++ {
			src := i + k
			if src >= n {
				out[i-from] = nil
			} else {
				out[i-from] = col[src]
			}
		}
	}
	return out
}

func seqLag(args ...[]any) ([]any, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("LAG requires 2 arguments (column, periods)")
	}
	return windowedSequences["LAG"](args[0], args[1:], 0, len(args[0]))
}

func seqLead(args ...[]any) ([]any, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("LEAD requires 2 arguments (column, periods)")
	}
	return windowedSequences["LEAD"](args[0], args[1:], 0, len(args[0]))
}

// =============================================================================
// DIFF / PCT_CHANGE
// =============================================================================

// diffRange and pctChangeRange return the outputs of DIFF and PCT_CHANGE for
// positions from up to, not including, to, in the way shiftRange does for a
// shift: out[i-from] is what the function gives position i on the whole column.
func diffRange(col []any, periods, from, to int) []any {
	out := make([]any, to-from)
	for i := from; i < to; i++ {
		if i < periods {
			out[i-from] = nil
			continue
		}
		a, okA := toFloat64(col[i])
		b, okB := toFloat64(col[i-periods])
		if !okA || !okB {
			out[i-from] = nil
			continue
		}
		out[i-from] = a - b
	}
	return out
}

func pctChangeRange(col []any, periods, from, to int) []any {
	out := make([]any, to-from)
	for i := from; i < to; i++ {
		if i < periods {
			out[i-from] = nil
			continue
		}
		a, okA := toFloat64(col[i])
		b, okB := toFloat64(col[i-periods])
		if !okA || !okB || b == 0 || math.IsNaN(b) {
			out[i-from] = nil
			continue
		}
		out[i-from] = (a - b) / b
	}
	return out
}

func seqDiff(args ...[]any) ([]any, error) {
	if len(args) < 1 || len(args) > 2 {
		return nil, fmt.Errorf("DIFF requires 1 or 2 arguments (column, periods=1)")
	}
	return windowedSequences["DIFF"](args[0], args[1:], 0, len(args[0]))
}

func seqPctChange(args ...[]any) ([]any, error) {
	if len(args) < 1 || len(args) > 2 {
		return nil, fmt.Errorf("PCT_CHANGE requires 1 or 2 arguments (column, periods=1)")
	}
	return windowedSequences["PCT_CHANGE"](args[0], args[1:], 0, len(args[0]))
}

// =============================================================================
// CUMSUM / CUMPROD / CUMMAX / CUMMIN
// =============================================================================

// cumulativeSequences holds how each cumulative function starts and combines,
// read both by the function and by its stream so the two cannot drift apart.
var cumulativeSequences = map[string]struct {
	initial       float64
	seedFromFirst bool
	combine       func(acc, v float64) float64
}{
	"CUMSUM":  {initial: 0, seedFromFirst: false, combine: func(a, v float64) float64 { return a + v }},
	"CUMPROD": {initial: 1, seedFromFirst: false, combine: func(a, v float64) float64 { return a * v }},
	"CUMMAX":  {initial: 0, seedFromFirst: true, combine: math.Max},
	"CUMMIN":  {initial: 0, seedFromFirst: true, combine: math.Min},
}

// cumAccumulator is the running state of CUMSUM, CUMPROD, CUMMAX and CUMMIN,
// shared by the functions and their streams so both run one loop. The state is
// the accumulator itself rather than the last value it produced: a running sum
// that reached NaN through +Inf and -Inf answers NaN for every row after it, and
// read back as a missing value it would start from the initial again.
type cumAccumulator struct {
	acc     float64
	seeded  bool
	combine func(acc, v float64) float64
}

// feed runs the accumulator over col and returns an output per value. A value
// that is not a number, or is NaN, leaves the running value alone and answers
// nothing for its row, exactly as the whole-column form does.
func (c *cumAccumulator) feed(col []any) []any {
	n := len(col)
	out := make([]any, n)
	for i := range n {
		v, ok := toFloat64(col[i])
		if !ok || math.IsNaN(v) {
			out[i] = nil
			continue
		}
		if !c.seeded {
			c.acc = v
			c.seeded = true
		} else {
			c.acc = c.combine(c.acc, v)
		}
		out[i] = c.acc
	}
	return out
}

func seqCumImpl(col []any, initial float64, seedFromFirst bool, combine func(acc, v float64) float64) []any {
	return (&cumAccumulator{acc: initial, seeded: !seedFromFirst, combine: combine}).feed(col)
}

func seqCumSum(args ...[]any) ([]any, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("CUMSUM requires 1 argument")
	}
	spec := cumulativeSequences["CUMSUM"]
	return seqCumImpl(args[0], spec.initial, spec.seedFromFirst, spec.combine), nil
}

func seqCumProd(args ...[]any) ([]any, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("CUMPROD requires 1 argument")
	}
	spec := cumulativeSequences["CUMPROD"]
	return seqCumImpl(args[0], spec.initial, spec.seedFromFirst, spec.combine), nil
}

func seqCumMax(args ...[]any) ([]any, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("CUMMAX requires 1 argument")
	}
	spec := cumulativeSequences["CUMMAX"]
	return seqCumImpl(args[0], spec.initial, spec.seedFromFirst, spec.combine), nil
}

func seqCumMin(args ...[]any) ([]any, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("CUMMIN requires 1 argument")
	}
	spec := cumulativeSequences["CUMMIN"]
	return seqCumImpl(args[0], spec.initial, spec.seedFromFirst, spec.combine), nil
}

// =============================================================================
// ROLLING_* (window-aligned right, MinObs = Window)
// =============================================================================

// rollingReduceRange reduces the window of each position from up to, not
// including, to: the window of position i is col[max(i-window+1, 0)..i], as it is
// for the whole column, and out[i-from] is nil unless the window holds window
// usable numbers. fn is handed those numbers in order and must not keep the
// slice.
func rollingReduceRange(col []any, window, from, to int, fn func(vals []float64) any) []any {
	out := make([]any, to-from)

	// Convert the column once, from the first row any window here reaches. Each
	// element used to be pulled out of its interface once per window it appeared
	// in: at 100,000 rows and a window of 5,000 that is 500 million unboxings of
	// 100,000 distinct values. The slice handed to fn holds the same values in the
	// same order, so every result is bit-identical: this takes the constant
	// factor, not the arithmetic.
	base := max(from-window+1, 0)
	nums := make([]float64, to-base)
	usable := make([]bool, to-base)
	for i := base; i < to; i++ {
		f, ok := toFloat64(col[i])
		if ok && !math.IsNaN(f) {
			nums[i-base] = f
			usable[i-base] = true
		}
	}

	// One buffer for every window instead of one allocation per window. None
	// of the reducers keeps the slice, so reusing it is safe. A window cannot
	// hold more values than the rows converted.
	vals := make([]float64, 0, min(window, to-base))
	for i := from; i < to; i++ {
		lo := max(i-window+1, 0)
		vals = vals[:0]
		for j := lo; j <= i; j++ {
			if usable[j-base] {
				vals = append(vals, nums[j-base])
			}
		}
		if len(vals) < window {
			out[i-from] = nil
			continue
		}
		out[i-from] = fn(vals)
	}
	return out
}

func rollingSum(vals []float64) any {
	var s float64
	for _, v := range vals {
		s += v
	}
	return s
}

func rollingMean(vals []float64) any {
	var s float64
	for _, v := range vals {
		s += v
	}
	return s / float64(len(vals))
}

func rollingMin(vals []float64) any {
	m := vals[0]
	for _, v := range vals[1:] {
		if v < m {
			m = v
		}
	}
	return m
}

func rollingMax(vals []float64) any {
	m := vals[0]
	for _, v := range vals[1:] {
		if v > m {
			m = v
		}
	}
	return m
}

func rollingStd(vals []float64) any {
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
}

func seqRollingSum(args ...[]any) ([]any, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("ROLLING_SUM requires 2 arguments (column, window)")
	}
	return windowedSequences["ROLLING_SUM"](args[0], args[1:], 0, len(args[0]))
}

func seqRollingMean(args ...[]any) ([]any, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("ROLLING_MEAN requires 2 arguments (column, window)")
	}
	return windowedSequences["ROLLING_MEAN"](args[0], args[1:], 0, len(args[0]))
}

func seqRollingMin(args ...[]any) ([]any, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("ROLLING_MIN requires 2 arguments (column, window)")
	}
	return windowedSequences["ROLLING_MIN"](args[0], args[1:], 0, len(args[0]))
}

func seqRollingMax(args ...[]any) ([]any, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("ROLLING_MAX requires 2 arguments (column, window)")
	}
	return windowedSequences["ROLLING_MAX"](args[0], args[1:], 0, len(args[0]))
}

func seqRollingStd(args ...[]any) ([]any, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("ROLLING_STD requires 2 arguments (column, window)")
	}
	return windowedSequences["ROLLING_STD"](args[0], args[1:], 0, len(args[0]))
}
