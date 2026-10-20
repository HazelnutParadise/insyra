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
	sig := emaOf(trix, signal, 3*(n-1)+1)
	return &TRIXResult{listOfFloats("TRIX", trix), listOfFloats("Signal", sig)}, nil
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
	// rows they are computed from. Default 26.
	Displacement int
}

// IchimokuResult holds the lines of Ichimoku Kinko Hyo.
type IchimokuResult struct {
	Tenkan  *insyra.DataList // conversion line: midpoint of the highest high and lowest low of tenkan rows
	Kijun   *insyra.DataList // base line: the same over kijun rows
	SenkouA *insyra.DataList // cloud line A in force on this row
	SenkouB *insyra.DataList // cloud line B in force on this row
}

// Ichimoku returns Goichi Hosoda's Ichimoku lines with the usual settings tenkan
// 9, kijun 26 and senkouB 52. Tenkan and Kijun are the midpoints of the highest
// high and lowest low of their last rows (the current one included). Senkou A
// is the average of the two, Senkou B the midpoint over senkouB rows, and both
// are drawn Displacement rows ahead on a chart; here SenkouA[t] and SenkouB[t]
// are the values in force on row t, computed from row t minus Displacement, so
// a close can be compared with the cloud it actually sat against, using only
// earlier rows. The chikou line, the close drawn back Displacement rows, is the
// close itself aligned this way and is not returned. Rows before a line exists
// are NaN.
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
	h, l, _, err := barSeries(high, low, close)
	if err != nil {
		return nil, err
	}
	mid := func(n int) []float64 {
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
	tk, kj, sb := mid(tenkan), mid(kijun), mid(senkouB)
	a, b := nanSeries(len(h)), nanSeries(len(h))
	for i := o.Displacement; i < len(h); i++ {
		j := i - o.Displacement
		if !math.IsNaN(tk[j]) && !math.IsNaN(kj[j]) {
			a[i] = (tk[j] + kj[j]) / 2
		}
		b[i] = sb[j]
	}
	return &IchimokuResult{listOfFloats("Tenkan", tk), listOfFloats("Kijun", kj), listOfFloats("SenkouA", a), listOfFloats("SenkouB", b)}, nil
}
