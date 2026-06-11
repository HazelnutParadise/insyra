package insyra

import (
	"math"
	"sort"
)

// =============================================================================
// Sequence transforms (Shift / Diff / PctChange)
// =============================================================================

// Shift returns a new DataList shifted by periods positions, keeping the
// original length. Positive periods shift right (lag — value at output index i
// is the input value at i-periods). Negative periods shift left (lead). Empty
// positions are filled with nil by default; pass a single fill value to
// override. Non-numeric values pass through unchanged, so Shift works on any
// column type. When |periods| >= len(dl) the output is all-fill of the same
// length.
func (dl *DataList) Shift(periods int, fill ...any) *DataList {
	var fillVal any
	if len(fill) > 0 {
		fillVal = fill[0]
	}
	var result *DataList
	dl.AtomicDo(func(dl *DataList) {
		n := len(dl.data)
		out := make([]any, n)
		switch {
		case periods == 0:
			copy(out, dl.data)
		case periods > 0:
			for i := range n {
				if i < periods {
					out[i] = fillVal
				} else {
					out[i] = dl.data[i-periods]
				}
			}
		default:
			k := -periods
			for i := range n {
				src := i + k
				if src >= n {
					out[i] = fillVal
				} else {
					out[i] = dl.data[src]
				}
			}
		}
		result = NewDataList(out...)
		result.name = dl.name
	})
	return result
}

// Diff returns the periods-step backward difference of the DataList:
// out[i] = in[i] - in[i-periods] (numeric subtraction). The first periods
// positions are nil. Cells where either operand is non-numeric or nil are
// emitted as nil. periods must be > 0; non-positive periods produce a warning
// and return nil.
//
// Unlike the legacy Difference (which returns length n-1), Diff preserves the
// input length so the result lines up with neighbouring columns.
func (dl *DataList) Diff(periods int) *DataList {
	if periods <= 0 {
		dl.warn("Diff", "periods must be > 0, got %d", periods)
		return nil
	}
	var result *DataList
	dl.AtomicDo(func(dl *DataList) {
		n := len(dl.data)
		out := make([]any, n)
		for i := range n {
			if i < periods {
				out[i] = nil
				continue
			}
			a, okA := ToFloat64Safe(dl.data[i])
			b, okB := ToFloat64Safe(dl.data[i-periods])
			if !okA || !okB {
				out[i] = nil
				continue
			}
			out[i] = a - b
		}
		result = NewDataList(out...)
		result.name = dl.name
	})
	return result
}

// PctChange returns the periods-step percentage change of the DataList:
// out[i] = (in[i] - in[i-periods]) / in[i-periods]. The first periods
// positions are nil. Cells where either operand is non-numeric, or where the
// denominator is zero, are emitted as nil. periods must be > 0.
func (dl *DataList) PctChange(periods int) *DataList {
	if periods <= 0 {
		dl.warn("PctChange", "periods must be > 0, got %d", periods)
		return nil
	}
	var result *DataList
	dl.AtomicDo(func(dl *DataList) {
		n := len(dl.data)
		out := make([]any, n)
		for i := range n {
			if i < periods {
				out[i] = nil
				continue
			}
			a, okA := ToFloat64Safe(dl.data[i])
			b, okB := ToFloat64Safe(dl.data[i-periods])
			if !okA || !okB || b == 0 || math.IsNaN(b) {
				out[i] = nil
				continue
			}
			out[i] = (a - b) / b
		}
		result = NewDataList(out...)
		result.name = dl.name
	})
	return result
}

// =============================================================================
// Cumulative reductions
// =============================================================================

// CumSum returns a same-length DataList where out[i] is the cumulative sum of
// the numeric values in in[0..=i]. Non-numeric or nil cells produce nil at
// that output position but do not break the running accumulator, matching
// pandas .cumsum(skipna=True).
func (dl *DataList) CumSum() *DataList {
	return dl.cumulative(0, false, func(acc, v float64) float64 { return acc + v })
}

// CumProd returns the cumulative product. See CumSum for nil semantics.
func (dl *DataList) CumProd() *DataList {
	return dl.cumulative(1, false, func(acc, v float64) float64 { return acc * v })
}

// CumMax returns the cumulative maximum. The accumulator is seeded by the
// first numeric value. See CumSum for nil semantics.
func (dl *DataList) CumMax() *DataList {
	return dl.cumulative(0, true, func(acc, v float64) float64 { return math.Max(acc, v) })
}

// CumMin returns the cumulative minimum. The accumulator is seeded by the
// first numeric value. See CumSum for nil semantics.
func (dl *DataList) CumMin() *DataList {
	return dl.cumulative(0, true, func(acc, v float64) float64 { return math.Min(acc, v) })
}

// cumulative is the shared implementation for CumSum/CumProd/CumMax/CumMin.
// When seedFromFirst is true, the accumulator starts uninitialised and is
// seeded by the first numeric input cell (used by min/max to avoid spurious
// 0 starts). Otherwise the supplied initial value is used.
func (dl *DataList) cumulative(initial float64, seedFromFirst bool, combine func(acc, v float64) float64) *DataList {
	var result *DataList
	dl.AtomicDo(func(dl *DataList) {
		n := len(dl.data)
		out := make([]any, n)
		acc := initial
		seeded := !seedFromFirst
		for i := range n {
			v, ok := ToFloat64Safe(dl.data[i])
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
		result = NewDataList(out...)
		result.name = dl.name
	})
	return result
}

// =============================================================================
// Rolling window
// =============================================================================

// RollingOptions configures a rolling-window computation. Window is required
// and must be positive. MinObs is the minimum number of valid (non-nil,
// numeric) observations a window must contain for the reducer to emit a
// value; when fewer valid observations are available the output is nil.
// MinObs defaults to Window when zero. Center, when true, anchors the window
// at the central index following pandas conventions (window covers
// [i-(w-1)/2, i+w/2], clipped to [0, n-1]). Weights, when set, must have
// length equal to Window and are used by Sum and Mean only.
type RollingOptions struct {
	Window  int
	MinObs  int
	Center  bool
	Weights []float64
}

// RollingDataList is the intermediate produced by DataList.Rolling. The
// terminal reducers (Sum / Mean / Min / Max / Median / Std / Var / Apply /
// Corr) each return a new DataList of the same length as the source. A
// RollingDataList carries a snapshot of the source data; the source itself is
// not held under lock while reducers run.
type RollingDataList struct {
	srcData []any
	srcName string
	opts    RollingOptions
	parent  *DataList
	err     string
}

// Rolling builds a rolling-window view over dl. The returned RollingDataList
// captures dl's current contents; subsequent mutations to dl do not affect
// the rolling computation.
func (dl *DataList) Rolling(opts RollingOptions) *RollingDataList {
	r := &RollingDataList{opts: opts, parent: dl}
	if opts.Window <= 0 {
		r.err = "Rolling: Window must be > 0"
		dl.warn("Rolling", "%s", r.err)
		return r
	}
	if opts.MinObs <= 0 {
		r.opts.MinObs = opts.Window
	} else if opts.MinObs > opts.Window {
		r.err = "Rolling: MinObs cannot exceed Window"
		dl.warn("Rolling", "%s", r.err)
		return r
	}
	if len(opts.Weights) > 0 && len(opts.Weights) != opts.Window {
		r.err = "Rolling: Weights length must equal Window"
		dl.warn("Rolling", "%s", r.err)
		return r
	}
	dl.AtomicDo(func(dl *DataList) {
		r.srcData = make([]any, len(dl.data))
		copy(r.srcData, dl.data)
		r.srcName = dl.name
	})
	return r
}

// windowBounds returns the inclusive [lo, hi] indices of the rolling window
// for output position i, clipped to [0, n-1]. The conceptual window may
// extend beyond the data on either side; the missing positions are accounted
// for via MinObs.
func (r *RollingDataList) windowBounds(i, n int) (int, int) {
	w := r.opts.Window
	var lo, hi int
	if r.opts.Center {
		lo = i - (w-1)/2
		hi = i + w/2
	} else {
		lo = i - w + 1
		hi = i
	}
	if lo < 0 {
		lo = 0
	}
	if hi >= n {
		hi = n - 1
	}
	return lo, hi
}

// collect returns the float64 values and matching weights from the window
// [lo, hi]. Only positions that are numeric and non-nil are included. The
// returned weights slice is nil when no Weights are configured.
func (r *RollingDataList) collect(i, lo, hi int) (vals []float64, weights []float64) {
	// Note: when Center is true and Weights are configured, the mapping from
	// window position to weight slot below still aligns by offset from the
	// conceptual leftmost cell — see the (j - (i - (w-1)/2)) computation.
	w := r.opts.Window
	for j := lo; j <= hi; j++ {
		f, ok := ToFloat64Safe(r.srcData[j])
		if !ok || math.IsNaN(f) {
			continue
		}
		vals = append(vals, f)
		if len(r.opts.Weights) == w {
			// For non-Center mode the rightmost cell aligns with the last weight.
			// For Center we map j-lo to the same offset.
			var off int
			if r.opts.Center {
				off = j - (i - (w-1)/2)
			} else {
				off = j - (i - w + 1)
			}
			if off >= 0 && off < w {
				weights = append(weights, r.opts.Weights[off])
			}
		}
	}
	return vals, weights
}

// reduce applies fn to every position 0..n-1 and produces a same-length
// DataList. fn receives the (already-validated, numeric) window values plus
// matching weights (nil when not configured). When fewer than MinObs values
// are present, nil is emitted at that position.
func (r *RollingDataList) reduce(fn func(vals, weights []float64) any) *DataList {
	if r.err != "" {
		out := NewDataList()
		out.name = r.srcName
		return out
	}
	n := len(r.srcData)
	out := make([]any, n)
	for i := range n {
		lo, hi := r.windowBounds(i, n)
		vals, weights := r.collect(i, lo, hi)
		if len(vals) < r.opts.MinObs {
			out[i] = nil
			continue
		}
		out[i] = fn(vals, weights)
	}
	dl := NewDataList(out...)
	dl.name = r.srcName
	return dl
}

// Sum returns the rolling sum. Weights, when set, multiply each value
// element-wise (length must equal Window).
func (r *RollingDataList) Sum() *DataList {
	return r.reduce(func(vals, weights []float64) any {
		var s float64
		if len(weights) == len(vals) && len(weights) > 0 {
			for i, v := range vals {
				s += v * weights[i]
			}
		} else {
			for _, v := range vals {
				s += v
			}
		}
		return s
	})
}

// Mean returns the rolling mean. When Weights are set the result is the
// weighted mean (sum of v_i * w_i divided by sum of w_i).
func (r *RollingDataList) Mean() *DataList {
	return r.reduce(func(vals, weights []float64) any {
		if len(weights) == len(vals) && len(weights) > 0 {
			var num, den float64
			for i, v := range vals {
				num += v * weights[i]
				den += weights[i]
			}
			if den == 0 {
				return nil
			}
			return num / den
		}
		var s float64
		for _, v := range vals {
			s += v
		}
		return s / float64(len(vals))
	})
}

// Min returns the rolling minimum.
func (r *RollingDataList) Min() *DataList {
	return r.reduce(func(vals, _ []float64) any {
		m := vals[0]
		for _, v := range vals[1:] {
			if v < m {
				m = v
			}
		}
		return m
	})
}

// Max returns the rolling maximum.
func (r *RollingDataList) Max() *DataList {
	return r.reduce(func(vals, _ []float64) any {
		m := vals[0]
		for _, v := range vals[1:] {
			if v > m {
				m = v
			}
		}
		return m
	})
}

// Median returns the rolling median (linear interpolation of the two middle
// values when the window has an even count of valid observations).
func (r *RollingDataList) Median() *DataList {
	return r.reduce(func(vals, _ []float64) any {
		tmp := make([]float64, len(vals))
		copy(tmp, vals)
		sort.Float64s(tmp)
		k := len(tmp)
		if k%2 == 1 {
			return tmp[k/2]
		}
		return (tmp[k/2-1] + tmp[k/2]) / 2
	})
}

// Std returns the rolling sample (n-1) standard deviation. Windows with
// fewer than 2 valid values emit nil regardless of MinObs.
func (r *RollingDataList) Std() *DataList {
	return r.reduce(func(vals, _ []float64) any {
		if len(vals) < 2 {
			return nil
		}
		return math.Sqrt(sampleVariance(vals))
	})
}

// Var returns the rolling sample (n-1) variance. Windows with fewer than 2
// valid values emit nil regardless of MinObs.
func (r *RollingDataList) Var() *DataList {
	return r.reduce(func(vals, _ []float64) any {
		if len(vals) < 2 {
			return nil
		}
		return sampleVariance(vals)
	})
}

// Apply runs fn over each window and writes its return value to the output
// column. fn receives the raw window slice (with the original any values,
// nils preserved), letting callers implement custom reducers. MinObs is
// counted on numeric values, but the slice passed to fn covers the full
// in-range window including nils.
func (r *RollingDataList) Apply(fn func(window []any) any) *DataList {
	if r.err != "" {
		out := NewDataList()
		out.name = r.srcName
		return out
	}
	if fn == nil {
		r.parent.warn("RollingApply", "fn must not be nil")
		out := NewDataList()
		out.name = r.srcName
		return out
	}
	n := len(r.srcData)
	out := make([]any, n)
	for i := range n {
		lo, hi := r.windowBounds(i, n)
		validCount := 0
		for j := lo; j <= hi; j++ {
			if _, ok := ToFloat64Safe(r.srcData[j]); ok {
				validCount++
			}
		}
		if validCount < r.opts.MinObs {
			out[i] = nil
			continue
		}
		window := make([]any, hi-lo+1)
		copy(window, r.srcData[lo:hi+1])
		out[i] = fn(window)
	}
	dl := NewDataList(out...)
	dl.name = r.srcName
	return dl
}

// Corr returns the rolling Pearson correlation against other. The two
// DataLists are aligned by index; pairs where either side is non-numeric or
// nil are skipped within each window. Windows with fewer than 2 valid pairs
// emit nil.
func (r *RollingDataList) Corr(other *DataList) *DataList {
	if r.err != "" {
		out := NewDataList()
		out.name = r.srcName
		return out
	}
	if other == nil {
		r.parent.warn("RollingCorr", "other DataList is nil")
		out := NewDataList()
		out.name = r.srcName
		return out
	}
	var otherData []any
	other.AtomicDo(func(o *DataList) {
		otherData = make([]any, len(o.data))
		copy(otherData, o.data)
	})
	n := len(r.srcData)
	if len(otherData) < n {
		// Align by truncating to the shorter length.
		n = len(otherData)
	}
	out := make([]any, len(r.srcData))
	for i := 0; i < len(r.srcData); i++ {
		if i >= n {
			out[i] = nil
			continue
		}
		lo, hi := r.windowBounds(i, n)
		var xs, ys []float64
		for j := lo; j <= hi; j++ {
			fx, okx := ToFloat64Safe(r.srcData[j])
			fy, oky := ToFloat64Safe(otherData[j])
			if !okx || !oky || math.IsNaN(fx) || math.IsNaN(fy) {
				continue
			}
			xs = append(xs, fx)
			ys = append(ys, fy)
		}
		if len(xs) < 2 || len(xs) < r.opts.MinObs {
			out[i] = nil
			continue
		}
		out[i] = pearson(xs, ys)
	}
	dl := NewDataList(out...)
	dl.name = r.srcName
	return dl
}

// =============================================================================
// Expanding window
// =============================================================================

// ExpandingDataList is the intermediate produced by DataList.Expanding. Each
// position i is reduced over in[0..=i] when at least MinObs valid observations
// are available, else nil.
type ExpandingDataList struct {
	srcData []any
	srcName string
	minObs  int
	parent  *DataList
	err     string
}

// Expanding builds an expanding-window view over dl. MinObs <= 0 defaults to 1.
func (dl *DataList) Expanding(minObs int) *ExpandingDataList {
	e := &ExpandingDataList{minObs: minObs, parent: dl}
	if e.minObs <= 0 {
		e.minObs = 1
	}
	dl.AtomicDo(func(dl *DataList) {
		e.srcData = make([]any, len(dl.data))
		copy(e.srcData, dl.data)
		e.srcName = dl.name
	})
	return e
}

func (e *ExpandingDataList) reduce(fn func(vals []float64) any) *DataList {
	if e.err != "" {
		out := NewDataList()
		out.name = e.srcName
		return out
	}
	n := len(e.srcData)
	out := make([]any, n)
	var vals []float64
	for i := range n {
		f, ok := ToFloat64Safe(e.srcData[i])
		if ok && !math.IsNaN(f) {
			vals = append(vals, f)
		}
		if len(vals) < e.minObs {
			out[i] = nil
			continue
		}
		out[i] = fn(vals)
	}
	dl := NewDataList(out...)
	dl.name = e.srcName
	return dl
}

// Sum returns the expanding sum.
func (e *ExpandingDataList) Sum() *DataList {
	return e.reduce(func(vals []float64) any {
		var s float64
		for _, v := range vals {
			s += v
		}
		return s
	})
}

// Mean returns the expanding mean.
func (e *ExpandingDataList) Mean() *DataList {
	return e.reduce(func(vals []float64) any {
		var s float64
		for _, v := range vals {
			s += v
		}
		return s / float64(len(vals))
	})
}

// Min returns the expanding minimum.
func (e *ExpandingDataList) Min() *DataList {
	return e.reduce(func(vals []float64) any {
		m := vals[0]
		for _, v := range vals[1:] {
			if v < m {
				m = v
			}
		}
		return m
	})
}

// Max returns the expanding maximum.
func (e *ExpandingDataList) Max() *DataList {
	return e.reduce(func(vals []float64) any {
		m := vals[0]
		for _, v := range vals[1:] {
			if v > m {
				m = v
			}
		}
		return m
	})
}

// Median returns the expanding median.
func (e *ExpandingDataList) Median() *DataList {
	return e.reduce(func(vals []float64) any {
		tmp := make([]float64, len(vals))
		copy(tmp, vals)
		sort.Float64s(tmp)
		k := len(tmp)
		if k%2 == 1 {
			return tmp[k/2]
		}
		return (tmp[k/2-1] + tmp[k/2]) / 2
	})
}

// Std returns the expanding sample (n-1) standard deviation. Positions with
// fewer than 2 valid observations emit nil regardless of MinObs.
func (e *ExpandingDataList) Std() *DataList {
	return e.reduce(func(vals []float64) any {
		if len(vals) < 2 {
			return nil
		}
		return math.Sqrt(sampleVariance(vals))
	})
}

// Var returns the expanding sample (n-1) variance. Positions with fewer than
// 2 valid observations emit nil regardless of MinObs.
func (e *ExpandingDataList) Var() *DataList {
	return e.reduce(func(vals []float64) any {
		if len(vals) < 2 {
			return nil
		}
		return sampleVariance(vals)
	})
}

// =============================================================================
// shared numerics
// =============================================================================

// sampleVariance returns the sample (n-1) variance of vals. Caller must ensure
// len(vals) >= 2.
func sampleVariance(vals []float64) float64 {
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
	return ss / float64(len(vals)-1)
}

// pearson returns the sample Pearson correlation between xs and ys. Caller
// must ensure len(xs) == len(ys) >= 2.
func pearson(xs, ys []float64) any {
	n := float64(len(xs))
	var sx, sy float64
	for i := range xs {
		sx += xs[i]
		sy += ys[i]
	}
	mx := sx / n
	my := sy / n
	var num, dx2, dy2 float64
	for i := range xs {
		dx := xs[i] - mx
		dy := ys[i] - my
		num += dx * dy
		dx2 += dx * dx
		dy2 += dy * dy
	}
	den := math.Sqrt(dx2 * dy2)
	if den == 0 {
		return nil
	}
	return num / den
}
