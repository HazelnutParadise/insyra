package ta

import (
	"errors"
	"fmt"
	"math"

	"github.com/HazelnutParadise/insyra"
)

// AroonResult holds the three Aroon lines, each from 0 to 100 (the oscillator
// from -100 to 100).
type AroonResult struct {
	Up         *insyra.DataList // how recently the highest high of the last n rows happened
	Down       *insyra.DataList // how recently the lowest low did
	Oscillator *insyra.DataList // Up minus Down
}

// Aroon returns Tushar Chande's Aroon indicator: Up is 100 times (n minus the
// rows since the highest high of the last n+1 rows) over n, so 100 means the
// high is today, and Down is the same for the lowest low. When the same extreme
// occurs more than once, the most recent one counts. The first n rows are NaN.
func Aroon(high, low insyra.IDataList, n int) (*AroonResult, error) {
	if err := checkPeriod("n", n, 1); err != nil {
		return nil, err
	}
	h, l, err := highLowSeries(high, low)
	if err != nil {
		return nil, err
	}
	up, down, osc := nanSeries(len(h)), nanSeries(len(h)), nanSeries(len(h))
	for i := n; i < len(h); i++ {
		hi, lo := i-n, i-n
		for j := i - n + 1; j <= i; j++ {
			if h[j] >= h[hi] {
				hi = j
			}
			if l[j] <= l[lo] {
				lo = j
			}
		}
		up[i] = 100 * float64(n-(i-hi)) / float64(n)
		down[i] = 100 * float64(n-(i-lo)) / float64(n)
		osc[i] = up[i] - down[i]
	}
	return &AroonResult{listOfFloats("Up", up), listOfFloats("Down", down), listOfFloats("Oscillator", osc)}, nil
}

// TRIXResult holds the TRIX line and its signal line.
type TRIXResult struct {
	TRIX   *insyra.DataList
	Signal *insyra.DataList // EMA of TRIX
}

// TRIX returns Jack Hutson's TRIX: the one-row percentage change of an EMA of
// an EMA of an EMA of the price, each n rows long, so 5 means the triple EMA
// rose 5%. Signal is the signal-row EMA of TRIX, which is how it is usually
// traded. TRIX is NaN before row 3(n-1)+1, Signal before row 3(n-1)+signal.
func TRIX(prices insyra.IDataList, n, signal int) (*TRIXResult, error) {
	if err := checkPeriod("n", n, 1); err != nil {
		return nil, err
	}
	if err := checkPeriod("signal", signal, 1); err != nil {
		return nil, err
	}
	p, err := numericSeries(prices, "prices")
	if err != nil {
		return nil, err
	}
	e1 := emaOf(p, n, 0)
	e2 := emaOf(e1, n, n-1)
	e3 := emaOf(e2, n, 2*(n-1))
	trix := nanSeries(len(p))
	for i := 1; i < len(p); i++ {
		if !math.IsNaN(e3[i]) && !math.IsNaN(e3[i-1]) && e3[i-1] != 0 {
			trix[i] = 100 * (e3[i]/e3[i-1] - 1)
		}
	}
	sig := signalOf(trix, signal)
	return &TRIXResult{listOfFloats("TRIX", trix), listOfFloats("Signal", sig)}, nil
}

// signalOf is the EMA of the defined values of v, seeded with the simple average
// of the first n consecutive defined values. A NaN row (TRIX is undefined when
// the triple EMA before it is 0) is NaN in the result and ends the average, which
// starts again from the next n consecutive defined values, so one undefined row
// does not silence the signal for good. Without gaps it equals emaOf from the
// first defined value.
func signalOf(v []float64, n int) []float64 {
	out := nanSeries(len(v))
	k := 2 / float64(n+1)
	run, seed := 0, 0.0
	have, prev := false, 0.0
	for i, x := range v {
		if math.IsNaN(x) {
			run, seed, have = 0, 0, false
			continue
		}
		if have {
			prev = (x-prev)*k + prev
			out[i] = prev
			continue
		}
		run++
		seed += x
		if run == n {
			prev, have = seed/float64(n), true
			out[i] = prev
		}
	}
	return out
}

// KAMA returns Perry Kaufman's adaptive moving average. The efficiency ratio of
// the last n rows (the net change over the sum of the absolute changes) picks a
// smoothing constant between that of a fast EMA and that of a slow one, so the
// line follows a trend closely and hardly moves in a sideways market. The usual
// settings are n 10, fast 2 and slow 30. As in TA-Lib the line starts from the
// price on row n-1 and is first reported on row n; the first n rows are NaN.
func KAMA(prices insyra.IDataList, n, fast, slow int) (*insyra.DataList, error) {
	if err := checkPeriod("n", n, 1); err != nil {
		return nil, err
	}
	if err := checkPeriod("fast", fast, 1); err != nil {
		return nil, err
	}
	if slow <= fast {
		return nil, fmt.Errorf("slow (%d) must be longer than fast (%d)", slow, fast)
	}
	p, err := numericSeries(prices, "prices")
	if err != nil {
		return nil, err
	}
	out := nanSeries(len(p))
	if len(p) < n {
		return listOfFloats("KAMA", out), nil
	}
	fastSC, slowSC := 2/float64(fast+1), 2/float64(slow+1)
	prev := p[n-1] // the line starts from the price of the last row without a full window
	for i := n; i < len(p); i++ {
		var path float64
		for j := i - n + 1; j <= i; j++ {
			path += math.Abs(p[j] - p[j-1])
		}
		er := 0.0
		if path != 0 {
			er = math.Abs(p[i]-p[i-n]) / path
		}
		sc := er*(fastSC-slowSC) + slowSC
		prev += sc * sc * (p[i] - prev)
		out[i] = prev
	}
	return listOfFloats("KAMA", out), nil
}

// IchimokuOptions are the optional settings of Ichimoku. A zero field takes the
// default shown.
type IchimokuOptions struct {
	// Displacement is how many rows the two cloud lines are drawn ahead of the
	// rows they are computed from, and how many rows back the lagging span is
	// drawn. Default 26.
	Displacement int
	// SegmentStarts are the rows on which a new data segment starts, for
	// example the first row after a trading halt or a relisting, as in
	// RSIOptions and KDOptions. Each segment is computed on its own: every line
	// starts again on the segment's first row and shows NaN until the segment has
	// its own data, and the displacement of the cloud and of Chikou never crosses
	// a boundary, so a value that would come from the next segment is NaN. Row 0
	// starts the first segment and need not be listed.
	SegmentStarts []int
}

// IchimokuResult holds the lines of Ichimoku Kinko Hyo.
type IchimokuResult struct {
	Tenkan  *insyra.DataList // conversion line: midpoint of the highest high and lowest low of tenkan rows
	Kijun   *insyra.DataList // base line: the same over kijun rows
	SenkouA *insyra.DataList // cloud line A in force on this row
	SenkouB *insyra.DataList // cloud line B in force on this row
	Chikou  *insyra.DataList // lagging span: the close Displacement rows later, NaN for the last Displacement rows of a segment
}

// Ichimoku returns Goichi Hosoda's Ichimoku lines with the usual settings tenkan
// 9, kijun 26 and senkouB 52. Tenkan and Kijun are the midpoints of the highest
// high and lowest low of their last rows (the current one included). Senkou A
// is the average of the two, Senkou B the midpoint over senkouB rows, and both
// are drawn Displacement rows ahead on a chart; here SenkouA[t] and SenkouB[t]
// are the values in force on row t, computed from row t minus Displacement, so
// a close can be compared with the cloud it actually sat against, using only
// earlier rows.
//
// Chikou is the lagging span, the close plotted Displacement rows behind:
// Chikou[t] is close[t+Displacement], so reading it at row t compares the close
// of Displacement rows later with the prices of row t. It therefore looks ahead
// by construction and is for studying the chart in hindsight, never for a rule
// that must be decided on row t; every other line uses only rows up to t. The
// last Displacement rows of a segment are NaN. Rows before a line exists are
// NaN.
func Ichimoku(high, low, close insyra.IDataList, tenkan, kijun, senkouB int, opts ...IchimokuOptions) (*IchimokuResult, error) {
	if len(opts) > 1 {
		return nil, errors.New("Ichimoku accepts at most one IchimokuOptions")
	}
	var o IchimokuOptions
	if len(opts) == 1 {
		o = opts[0]
	}
	if o.Displacement < 0 {
		return nil, errors.New("option Displacement must not be negative")
	}
	if o.Displacement == 0 {
		o.Displacement = 26
	}
	for name, v := range map[string]int{"tenkan": tenkan, "kijun": kijun, "senkouB": senkouB} {
		if err := checkPeriod(name, v, 1); err != nil {
			return nil, err
		}
	}
	h, l, c, err := barSeries(high, low, close)
	if err != nil {
		return nil, err
	}
	starts, err := cleanSegmentStarts(o.SegmentStarts, len(h))
	if err != nil {
		return nil, err
	}
	tk, kj, a, b, ch := nanSeries(len(h)), nanSeries(len(h)), nanSeries(len(h)), nanSeries(len(h)), nanSeries(len(h))
	for _, bd := range segmentBounds(len(h), starts) {
		lo, hi := bd[0], bd[1]
		hs, ls := h[lo:hi], l[lo:hi]
		stk, skj, ssb := ichimokuMid(hs, ls, tenkan), ichimokuMid(hs, ls, kijun), ichimokuMid(hs, ls, senkouB)
		copy(tk[lo:hi], stk)
		copy(kj[lo:hi], skj)
		for i := o.Displacement; i < hi-lo; i++ {
			j := i - o.Displacement
			if !math.IsNaN(stk[j]) && !math.IsNaN(skj[j]) {
				a[lo+i] = (stk[j] + skj[j]) / 2
			}
			b[lo+i] = ssb[j]
		}
		for i := 0; i+o.Displacement < hi-lo; i++ {
			ch[lo+i] = c[lo+i+o.Displacement]
		}
	}
	return &IchimokuResult{listOfFloats("Tenkan", tk), listOfFloats("Kijun", kj), listOfFloats("SenkouA", a), listOfFloats("SenkouB", b), listOfFloats("Chikou", ch)}, nil
}

// ichimokuMid is the midpoint of the highest high and lowest low of the last n
// rows, NaN before the n-th row.
func ichimokuMid(h, l []float64, n int) []float64 {
	out := nanSeries(len(h))
	for i := n - 1; i < len(h); i++ {
		hi, lo := h[i], l[i]
		for j := i - n + 1; j < i; j++ {
			hi, lo = math.Max(hi, h[j]), math.Min(lo, l[j])
		}
		out[i] = (hi + lo) / 2
	}
	return out
}
