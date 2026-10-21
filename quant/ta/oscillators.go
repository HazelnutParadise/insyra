package ta

import (
	"errors"
	"fmt"
	"math"

	"github.com/HazelnutParadise/insyra"
)

// CCI returns Donald Lambert's commodity channel index: how far the typical
// price (high + low + close) / 3 is from its n-row simple average, in units of
// 0.015 times the mean absolute deviation, so that about 70% to 80% of values
// fall between -100 and 100. The first n-1 rows are NaN, and a window whose
// typical price does not move is NaN.
func CCI(high, low, close insyra.IDataList, n int) (*insyra.DataList, error) {
	if err := checkPeriod("n", n, 1); err != nil {
		return nil, err
	}
	h, l, c, err := barSeries(high, low, close)
	if err != nil {
		return nil, err
	}
	tp := make([]float64, len(c))
	for i := range c {
		tp[i] = (h[i] + l[i] + c[i]) / 3
	}
	mean := smaOf(tp, n)
	out := nanSeries(len(c))
	for i := n - 1; i < len(c); i++ {
		var dev float64
		for j := i - n + 1; j <= i; j++ {
			dev += math.Abs(tp[j] - mean[i])
		}
		if dev = dev / float64(n); dev != 0 {
			out[i] = (tp[i] - mean[i]) / (0.015 * dev)
		}
	}
	return listOfFloats("CCI", out), nil
}

// Momentum returns the price change over n rows, price minus the price n rows
// earlier, in the units of the price. The first n rows are NaN.
func Momentum(prices insyra.IDataList, n int) (*insyra.DataList, error) {
	if err := checkPeriod("n", n, 1); err != nil {
		return nil, err
	}
	p, err := numericSeries(prices, "prices")
	if err != nil {
		return nil, err
	}
	out := nanSeries(len(p))
	for i := n; i < len(p); i++ {
		out[i] = p[i] - p[i-n]
	}
	return listOfFloats("Momentum", out), nil
}

// ROC returns the rate of change over n rows as a percentage, as TA-Lib does:
// 5 means the price rose 5% from n rows earlier. (Bias and the returns used by
// quant are fractions; ROC, like RSI and KD, is on a 0 to 100 scale.) A price
// of 0 n rows earlier gives NaN. The first n rows are NaN.
func ROC(prices insyra.IDataList, n int) (*insyra.DataList, error) {
	if err := checkPeriod("n", n, 1); err != nil {
		return nil, err
	}
	p, err := numericSeries(prices, "prices")
	if err != nil {
		return nil, err
	}
	out := nanSeries(len(p))
	for i := n; i < len(p); i++ {
		if p[i-n] != 0 {
			out[i] = (p[i]/p[i-n] - 1) * 100
		}
	}
	return listOfFloats("ROC", out), nil
}

// UltimateOscillator returns Larry Williams' ultimate oscillator: the buying
// pressure (close minus the lower of the low and the previous close) over the
// true range, summed over a short, a medium and a long window, and combined with
// weights 4, 2 and 1. The usual windows are 7, 14 and 28. It lies between 0 and
// 100, and the first long rows are NaN.
func UltimateOscillator(high, low, close insyra.IDataList, short, medium, long int) (*insyra.DataList, error) {
	for name, v := range map[string]int{"short": short, "medium": medium, "long": long} {
		if err := checkPeriod(name, v, 1); err != nil {
			return nil, err
		}
	}
	if short >= medium || medium >= long {
		return nil, fmt.Errorf("short (%d) must be below medium (%d), which must be below long (%d)", short, medium, long)
	}
	h, l, c, err := barSeries(high, low, close)
	if err != nil {
		return nil, err
	}
	bp, tr := make([]float64, len(c)), make([]float64, len(c))
	for i := 1; i < len(c); i++ {
		lo, hi := math.Min(l[i], c[i-1]), math.Max(h[i], c[i-1])
		bp[i], tr[i] = c[i]-lo, hi-lo
	}
	avg := func(i, n int) float64 {
		var b, r float64
		for j := i - n + 1; j <= i; j++ {
			b, r = b+bp[j], r+tr[j]
		}
		if r == 0 {
			return math.NaN()
		}
		return b / r
	}
	out := nanSeries(len(c))
	for i := long; i < len(c); i++ {
		out[i] = 100 * (4*avg(i, short) + 2*avg(i, medium) + avg(i, long)) / 7
	}
	return listOfFloats("UltimateOscillator", out), nil
}

// StochRSI returns the stochastic oscillator of the RSI: where the n-row
// (Wilder) RSI lies between its lowest and highest values of the last stochN
// rows, from 0 to 100, smoothed into K and D as KDOptions say, with the same
// default as KD (the Taiwanese recursive form, a flat window counting as 50).
// The first n+stochN-1 rows are NaN.
//
// The default smoothing is not TA-Lib's or TradingView's, so the numbers differ
// from theirs. To get TA-Lib's STOCHRSI (fastk_period stochN, fastd_period 3)
// pass KDOptions{Method: KDMovingAverage, SlowK: 1, SlowD: 3}: K is then the raw
// value itself and D its 3-row average, and the tests check this against TA-Lib.
// TradingView's "Stoch RSI" averages the raw value by simple moving averages too,
// K over 3 rows and D over 3 rows, which is KDMovingAverage with SlowK 3 and
// SlowD 3 (taken from its published definition, not checked against it here). A
// window in which the RSI is undefined (no gain and no loss yet) is NaN, and
// recursive smoothing starts from 50 on the first row that has a value.
//
// KDOptions.SegmentStarts restarts everything on its own at each segment: the
// RSI, the stochastic window and the smoothing, so nothing is carried across a
// data break and a segment shows NaN through its own warm-up.
func StochRSI(prices insyra.IDataList, n, stochN int, opts ...KDOptions) (*KDResult, error) {
	if err := checkPeriod("n", n, 1); err != nil {
		return nil, err
	}
	if err := checkPeriod("stochN", stochN, 1); err != nil {
		return nil, err
	}
	o, err := kdOptions("StochRSI", opts)
	if err != nil {
		return nil, err
	}
	p, err := numericSeries(prices, "prices")
	if err != nil {
		return nil, err
	}
	starts, err := cleanSegmentStarts(o.SegmentStarts, len(p))
	if err != nil {
		return nil, err
	}
	k, d := nanSeries(len(p)), nanSeries(len(p))
	for _, b := range segmentBounds(len(p), starts) {
		sk, sd := stochRSISegment(p[b[0]:b[1]], n, stochN, o)
		copy(k[b[0]:b[1]], sk)
		copy(d[b[0]:b[1]], sd)
	}
	return &KDResult{listOfFloats("K", k), listOfFloats("D", d)}, nil
}

// stochRSISegment is StochRSI over one data segment, with the settings of o
// already checked.
func stochRSISegment(p []float64, n, stochN int, o KDOptions) (k, d []float64) {
	rsi := rsiRun(p, n, RSIWilder)
	first := n + stochN - 1
	rsv := nanSeries(len(p))
	for i := first; i < len(p); i++ {
		hi, lo := rsi[i], rsi[i]
		for j := i - stochN + 1; j < i; j++ {
			hi, lo = math.Max(hi, rsi[j]), math.Min(lo, rsi[j])
		}
		switch {
		case math.IsNaN(hi) || math.IsNaN(lo):
			// a window of an undefined RSI (no gain and no loss) stays NaN
		case hi == lo:
			rsv[i] = 50
		default:
			rsv[i] = (rsi[i] - lo) / (hi - lo) * 100
		}
	}
	return smoothStochastic(rsv, first, o)
}

// ARBRResult holds the two sentiment indicators of the AR/BR pair.
type ARBRResult struct {
	AR *insyra.DataList
	BR *insyra.DataList
}

// ARBR returns the AR and BR popularity and willingness indicators, common in
// Taiwan and China, over n rows (usually 26). AR is 100 times the sum of
// (high minus open) over the sum of (open minus low); BR is 100 times the sum of
// the rise of the high above the previous close over the sum of the fall of the
// low below it, each taken as 0 when negative. 100 is balance. A window with a
// zero denominator is NaN, AR is NaN before row n-1 and BR before row n.
func ARBR(open, high, low, close insyra.IDataList, n int) (*ARBRResult, error) {
	if err := checkPeriod("n", n, 1); err != nil {
		return nil, err
	}
	o, err := numericSeries(open, "open")
	if err != nil {
		return nil, err
	}
	h, l, c, err := barSeries(high, low, close)
	if err != nil {
		return nil, err
	}
	if len(o) != len(c) {
		return nil, fmt.Errorf("open has %d rows but close has %d", len(o), len(c))
	}
	ar, br := nanSeries(len(c)), nanSeries(len(c))
	for i := n - 1; i < len(c); i++ {
		var up, down float64
		for j := i - n + 1; j <= i; j++ {
			up, down = up+(h[j]-o[j]), down+(o[j]-l[j])
		}
		if down != 0 {
			ar[i] = 100 * up / down
		}
	}
	for i := n; i < len(c); i++ {
		var up, down float64
		for j := i - n + 1; j <= i; j++ {
			up += math.Max(0, h[j]-c[j-1])
			down += math.Max(0, c[j-1]-l[j])
		}
		if down != 0 {
			br[i] = 100 * up / down
		}
	}
	return &ARBRResult{listOfFloats("AR", ar), listOfFloats("BR", br)}, nil
}

// CRMidpoint chooses the price CR compares each day's high and low with.
type CRMidpoint int

const (
	// CRMidHighLow is (high + low) / 2 of the previous row, the original form and
	// the default.
	CRMidHighLow CRMidpoint = iota
	// CRMidWeightedClose is (2 × close + high + low) / 4.
	CRMidWeightedClose
	// CRMidTypical is (high + low + close) / 3.
	CRMidTypical
	// CRMidOHLC is (open + high + low + close) / 4 and needs CROptions.Open.
	CRMidOHLC
)

// String returns the constant name of m, or CRMidpoint(n) for a value that is not
// one of the constants.
func (m CRMidpoint) String() string {
	switch m {
	case CRMidHighLow:
		return "CRMidHighLow"
	case CRMidWeightedClose:
		return "CRMidWeightedClose"
	case CRMidTypical:
		return "CRMidTypical"
	case CRMidOHLC:
		return "CRMidOHLC"
	}
	return fmt.Sprintf("CRMidpoint(%d)", int(m))
}

// CROptions are the optional settings of CR.
type CROptions struct {
	// Midpoint is the price the day's high and low are compared with. Platforms
	// differ on it; the default is CRMidHighLow.
	Midpoint CRMidpoint
	// Open is the open prices, required for CRMidOHLC and ignored otherwise.
	Open insyra.IDataList
	// SegmentStarts are the rows on which a new data segment starts, for
	// example the first row after a trading halt or a relisting. Each segment is
	// computed on its own: a window never reaches back across a start, and a
	// segment shows NaN through its first n rows. Row 0 starts the first segment
	// and need not be listed. Left empty, the result does not change.
	SegmentStarts []int
}

// CR returns the intermediate willingness index over n rows (usually 26): 100
// times the sum of how far each day's high is above the previous row's midpoint
// over the sum of how far its low is below it, each taken as 0 when negative.
// CROptions chooses the midpoint. A window with a zero denominator is NaN, and
// the first n rows are NaN.
func CR(high, low, close insyra.IDataList, n int, opts ...CROptions) (*insyra.DataList, error) {
	if err := checkPeriod("n", n, 1); err != nil {
		return nil, err
	}
	if len(opts) > 1 {
		return nil, errors.New("CR accepts at most one CROptions")
	}
	var opt CROptions
	if len(opts) == 1 {
		opt = opts[0]
	}
	if opt.Midpoint < CRMidHighLow || opt.Midpoint > CRMidOHLC {
		return nil, fmt.Errorf("unsupported CR midpoint %v", opt.Midpoint)
	}
	h, l, c, err := barSeries(high, low, close)
	if err != nil {
		return nil, err
	}
	var o []float64
	if opt.Midpoint == CRMidOHLC {
		if insyra.IsNilList(opt.Open) {
			return nil, errors.New("CRMidOHLC needs CROptions.Open")
		}
		if o, err = numericSeries(opt.Open, "open"); err != nil {
			return nil, err
		}
		if len(o) != len(c) {
			return nil, fmt.Errorf("open has %d rows but close has %d", len(o), len(c))
		}
	}
	mid := make([]float64, len(c))
	for i := range c {
		switch opt.Midpoint {
		case CRMidHighLow:
			mid[i] = (h[i] + l[i]) / 2
		case CRMidWeightedClose:
			mid[i] = (2*c[i] + h[i] + l[i]) / 4
		case CRMidTypical:
			mid[i] = (h[i] + l[i] + c[i]) / 3
		case CRMidOHLC:
			mid[i] = (o[i] + h[i] + l[i] + c[i]) / 4
		}
	}
	starts, err := cleanSegmentStarts(opt.SegmentStarts, len(c))
	if err != nil {
		return nil, err
	}
	out := nanSeries(len(c))
	for _, b := range segmentBounds(len(c), starts) {
		// a row needs the midpoint of the row before it, so the first n rows of
		// every segment have no value
		for i := b[0] + n; i < b[1]; i++ {
			var up, down float64
			for j := i - n + 1; j <= i; j++ {
				up += math.Max(0, h[j]-mid[j-1])
				down += math.Max(0, mid[j-1]-l[j])
			}
			if down != 0 {
				out[i] = 100 * up / down
			}
		}
	}
	return listOfFloats("CR", out), nil
}

// ForceIndex returns Alexander Elder's force index: the n-row EMA of the price
// change times the volume, (close minus the previous close) × volume. With n 1 it
// is the raw force. The first n rows are NaN.
func ForceIndex(close, volume insyra.IDataList, n int) (*insyra.DataList, error) {
	if err := checkPeriod("n", n, 1); err != nil {
		return nil, err
	}
	c, err := numericSeries(close, "close")
	if err != nil {
		return nil, err
	}
	v, err := numericSeries(volume, "volume")
	if err != nil {
		return nil, err
	}
	if len(c) != len(v) {
		return nil, fmt.Errorf("close has %d rows but volume has %d", len(c), len(v))
	}
	raw := nanSeries(len(c))
	for i := 1; i < len(c); i++ {
		raw[i] = (c[i] - c[i-1]) * v[i]
	}
	return listOfFloats("ForceIndex", emaOf(raw, n, 1)), nil
}

// ElderRayResult holds the bull power and bear power.
type ElderRayResult struct {
	BullPower *insyra.DataList // high minus the EMA
	BearPower *insyra.DataList // low minus the EMA
}

// ElderRay returns Alexander Elder's bull power (the high minus the n-row EMA of
// the close) and bear power (the low minus it): how far buyers pushed the price
// above the consensus and sellers below it. The usual n is 13. The first n-1
// rows are NaN.
func ElderRay(high, low, close insyra.IDataList, n int) (*ElderRayResult, error) {
	if err := checkPeriod("n", n, 1); err != nil {
		return nil, err
	}
	h, l, c, err := barSeries(high, low, close)
	if err != nil {
		return nil, err
	}
	ema := emaOf(c, n, 0)
	bull, bear := nanSeries(len(c)), nanSeries(len(c))
	for i := range c {
		if !math.IsNaN(ema[i]) {
			bull[i], bear[i] = h[i]-ema[i], l[i]-ema[i]
		}
	}
	return &ElderRayResult{listOfFloats("BullPower", bull), listOfFloats("BearPower", bear)}, nil
}
