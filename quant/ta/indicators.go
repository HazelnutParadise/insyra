package ta

import (
	"errors"
	"fmt"
	"math"

	"github.com/HazelnutParadise/insyra"
)

// The indicators in this file follow the definitions of TA-Lib, the usual
// reference, except KD, whose default is the recursive form used by Taiwanese
// brokers' software (use KDMovingAverage for TA-Lib's STOCH): a window that is
// not full is NaN, EMA starts from the simple average of its first n values,
// and a standard deviation is the population one. Every returned list is named
// after its line (SMA, MACD, Signal, Upper, K and so on), so it can be added to
// a table as it is. They refuse a missing, non-numeric or non-finite value instead of
// guessing, since a moving average over a hole is not the average of anything.

func nanSeries(n int) []float64 {
	out := make([]float64, n)
	for i := range out {
		out[i] = math.NaN()
	}
	return out
}

func listOfFloats(name string, v []float64) *insyra.DataList {
	out := make([]any, len(v))
	for i, x := range v {
		out[i] = x
	}
	return insyra.NewDataList(out...).SetName(name)
}

func checkPeriod(name string, n, min int) error {
	if n < min {
		return fmt.Errorf("%s must be at least %d, got %d", name, min, n)
	}
	return nil
}

func smaOf(v []float64, n int) []float64 {
	out := nanSeries(len(v))
	var sum float64
	for i, x := range v {
		sum += x
		if i >= n {
			sum -= v[i-n]
		}
		if i >= n-1 {
			out[i] = sum / float64(n)
		}
	}
	return out
}

// emaOf is the exponential average of v[from:], seeded with the simple average
// of its first n values, which is where the first value appears. Rows before
// that are NaN.
func emaOf(v []float64, n, from int) []float64 {
	out := nanSeries(len(v))
	if len(v)-from < n {
		return out
	}
	var seed float64
	for i := from; i < from+n; i++ {
		seed += v[i]
	}
	prev := seed / float64(n)
	out[from+n-1] = prev
	k := 2 / float64(n+1)
	for i := from + n; i < len(v); i++ {
		prev = (v[i]-prev)*k + prev
		out[i] = prev
	}
	return out
}

// SMA returns the simple moving average of the last n prices. The first n-1
// rows are NaN.
func SMA(prices insyra.IDataList, n int) (*insyra.DataList, error) {
	if err := checkPeriod("n", n, 1); err != nil {
		return nil, err
	}
	p, err := numericSeries(prices, "prices")
	if err != nil {
		return nil, err
	}
	return listOfFloats("SMA", smaOf(p, n)), nil
}

// EMA returns the exponential moving average with weight 2/(n+1), started from
// the simple average of the first n prices, so the first n-1 rows are NaN and
// row n-1 is that average.
func EMA(prices insyra.IDataList, n int) (*insyra.DataList, error) {
	if err := checkPeriod("n", n, 1); err != nil {
		return nil, err
	}
	p, err := numericSeries(prices, "prices")
	if err != nil {
		return nil, err
	}
	return listOfFloats("EMA", emaOf(p, n, 0)), nil
}

// MACDResult holds the three lines of MACD.
type MACDResult struct {
	MACD      *insyra.DataList // fast EMA minus slow EMA
	Signal    *insyra.DataList // EMA of MACD
	Histogram *insyra.DataList // MACD minus Signal
}

// MACD returns Appel's moving average convergence divergence: the fast EMA
// minus the slow EMA, its own EMA over signal rows, and the difference. The
// usual settings are 12, 26 and 9.
//
// As in TA-Lib, both EMAs first appear on row slow-1, the fast one started
// from the simple average of the fast prices ending there. MACD is NaN before
// row slow-1, Signal and Histogram before row slow+signal-2. TA-Lib leaves
// the MACD line empty until the signal line exists; the values agree wherever
// both have one.
func MACD(prices insyra.IDataList, fast, slow, signal int) (*MACDResult, error) {
	if err := checkPeriod("fast", fast, 2); err != nil {
		return nil, err
	}
	if err := checkPeriod("signal", signal, 1); err != nil {
		return nil, err
	}
	if slow <= fast {
		return nil, fmt.Errorf("slow (%d) must be longer than fast (%d)", slow, fast)
	}
	p, err := numericSeries(prices, "prices")
	if err != nil {
		return nil, err
	}
	slowEMA := emaOf(p, slow, 0)
	fastEMA := emaOf(p, fast, slow-fast)
	macd := nanSeries(len(p))
	for i := range p {
		if !math.IsNaN(slowEMA[i]) && !math.IsNaN(fastEMA[i]) {
			macd[i] = fastEMA[i] - slowEMA[i]
		}
	}
	sig := emaOf(macd, signal, slow-1)
	hist := nanSeries(len(p))
	for i := range p {
		if !math.IsNaN(sig[i]) {
			hist[i] = macd[i] - sig[i]
		}
	}
	return &MACDResult{listOfFloats("MACD", macd), listOfFloats("Signal", sig), listOfFloats("Histogram", hist)}, nil
}

// BollingerResult holds the three bands.
type BollingerResult struct {
	Upper  *insyra.DataList
	Middle *insyra.DataList
	Lower  *insyra.DataList
}

// Bollinger returns Bollinger bands: the simple average of the last n prices
// and bands k standard deviations above and below it, using the population
// standard deviation. The usual settings are 20 and 2. The first n-1 rows are
// NaN.
func Bollinger(prices insyra.IDataList, n int, k float64) (*BollingerResult, error) {
	if err := checkPeriod("n", n, 2); err != nil {
		return nil, err
	}
	if math.IsNaN(k) || math.IsInf(k, 0) || k < 0 {
		return nil, errors.New("k must be a finite number of standard deviations, 0 or more")
	}
	p, err := numericSeries(prices, "prices")
	if err != nil {
		return nil, err
	}
	mid := smaOf(p, n)
	up, lo := nanSeries(len(p)), nanSeries(len(p))
	for i := n - 1; i < len(p); i++ {
		var ss float64
		for j := i - n + 1; j <= i; j++ {
			d := p[j] - mid[i]
			ss += d * d
		}
		sd := math.Sqrt(ss / float64(n))
		up[i], lo[i] = mid[i]+k*sd, mid[i]-k*sd
	}
	return &BollingerResult{listOfFloats("Upper", up), listOfFloats("Middle", mid), listOfFloats("Lower", lo)}, nil
}

func barSeries(high, low, close insyra.IDataList) (h, l, c []float64, err error) {
	if h, err = numericSeries(high, "high"); err != nil {
		return
	}
	if l, err = numericSeries(low, "low"); err != nil {
		return
	}
	if c, err = numericSeries(close, "close"); err != nil {
		return
	}
	if len(h) != len(l) || len(h) != len(c) {
		return nil, nil, nil, fmt.Errorf("high, low and close must have the same length, got %d, %d and %d", len(h), len(l), len(c))
	}
	for i := range h {
		if h[i] < l[i] {
			return nil, nil, nil, fmt.Errorf("high %v is below low %v at row index %d", h[i], l[i], i)
		}
	}
	return
}

func trueRangeOf(h, l, c []float64) []float64 {
	tr := nanSeries(len(h))
	for i := 1; i < len(h); i++ {
		tr[i] = math.Max(h[i]-l[i], math.Max(math.Abs(h[i]-c[i-1]), math.Abs(l[i]-c[i-1])))
	}
	return tr
}

// TrueRange returns Wilder's true range: the largest of the day's high minus
// low and the distances from the previous close to the high and to the low.
// Row 0, which has no previous close, is NaN.
func TrueRange(high, low, close insyra.IDataList) (*insyra.DataList, error) {
	h, l, c, err := barSeries(high, low, close)
	if err != nil {
		return nil, err
	}
	return listOfFloats("TrueRange", trueRangeOf(h, l, c)), nil
}

// ATR returns Wilder's average true range over n rows: row n is the simple
// average of the true ranges of rows 1 to n, and each later row is
// (previous*(n-1) + true range)/n. The first n rows are NaN.
func ATR(high, low, close insyra.IDataList, n int) (*insyra.DataList, error) {
	if err := checkPeriod("n", n, 1); err != nil {
		return nil, err
	}
	h, l, c, err := barSeries(high, low, close)
	if err != nil {
		return nil, err
	}
	return listOfFloats("ATR", atrOf(h, l, c, n)), nil
}

// atrOf is Wilder's average true range of the bars h, l, c: row n is the mean of
// the true ranges of rows 1 to n and each later row is smoothed with weight 1/n.
func atrOf(h, l, c []float64, n int) []float64 {
	tr := trueRangeOf(h, l, c)
	out := nanSeries(len(h))
	if len(h) <= n {
		return out
	}
	var sum float64
	for i := 1; i <= n; i++ {
		sum += tr[i]
	}
	prev := sum / float64(n)
	out[n] = prev
	for i := n + 1; i < len(h); i++ {
		prev = (prev*float64(n-1) + tr[i]) / float64(n)
		out[i] = prev
	}
	return out
}

// KDMethod chooses how the raw stochastic value is smoothed into K and D.
type KDMethod int

const (
	// KDRecursive smooths with K = (1-1/SlowK)*previous K + RSV/SlowK, and D the
	// same way from K, both starting at 50: the KD of Taiwanese brokers'
	// software, with the usual 3 for both.
	KDRecursive KDMethod = iota
	// KDMovingAverage takes K as the simple average of the last SlowK RSV values
	// and D as the simple average of the last SlowD values of K: TA-Lib's STOCH.
	KDMovingAverage
)

// String returns the constant name of m, or KDMethod(n) for a value that is not
// one of the constants.
func (m KDMethod) String() string {
	switch m {
	case KDRecursive:
		return "KDRecursive"
	case KDMovingAverage:
		return "KDMovingAverage"
	}
	return fmt.Sprintf("KDMethod(%d)", int(m))
}

// KDOptions are the optional settings of KD. A zero field takes the default
// shown.
type KDOptions struct {
	Method KDMethod // KDRecursive (default) or KDMovingAverage
	SlowK  int      // smoothing of K, default 3
	SlowD  int      // smoothing of D, default 3
}

// KDResult holds the two lines of the stochastic oscillator.
type KDResult struct {
	K *insyra.DataList
	D *insyra.DataList
}

// KD returns the stochastic oscillator. The raw stochastic value RSV is where
// the close lies between the lowest low and the highest high of the last n rows,
// from 0 to 100; a flat window, whose high equals its low, counts as 50. K and D
// are smoothed from it as KDOptions.Method says. The usual setting is n = 9.
//
// With KDRecursive the first value is on row n-1, from a previous K and D of 50.
// With KDMovingAverage the first K is on row n+SlowK-2 and the first D SlowD-1
// rows later. Earlier rows are NaN. TA-Lib starts K together with D; the
// values agree wherever both have one.
func KD(high, low, close insyra.IDataList, n int, opts ...KDOptions) (*KDResult, error) {
	if err := checkPeriod("n", n, 1); err != nil {
		return nil, err
	}
	if len(opts) > 1 {
		return nil, errors.New("KD accepts at most one KDOptions")
	}
	var o KDOptions
	if len(opts) == 1 {
		o = opts[0]
	}
	if o.Method != KDRecursive && o.Method != KDMovingAverage {
		return nil, fmt.Errorf("unsupported KD method %v", o.Method)
	}
	if o.SlowK == 0 {
		o.SlowK = 3
	}
	if o.SlowD == 0 {
		o.SlowD = 3
	}
	if err := checkPeriod("SlowK", o.SlowK, 1); err != nil {
		return nil, err
	}
	if err := checkPeriod("SlowD", o.SlowD, 1); err != nil {
		return nil, err
	}
	h, l, c, err := barSeries(high, low, close)
	if err != nil {
		return nil, err
	}
	rsv := nanSeries(len(c))
	for i := n - 1; i < len(c); i++ {
		hh, ll := h[i], l[i]
		for j := i - n + 1; j < i; j++ {
			hh, ll = math.Max(hh, h[j]), math.Min(ll, l[j])
		}
		if hh == ll {
			rsv[i] = 50
		} else {
			rsv[i] = (c[i] - ll) / (hh - ll) * 100
		}
	}
	k, d := nanSeries(len(c)), nanSeries(len(c))
	switch o.Method {
	case KDRecursive:
		pk, pd := 50.0, 50.0
		for i := n - 1; i < len(c); i++ {
			pk = pk*float64(o.SlowK-1)/float64(o.SlowK) + rsv[i]/float64(o.SlowK)
			pd = pd*float64(o.SlowD-1)/float64(o.SlowD) + pk/float64(o.SlowD)
			k[i], d[i] = pk, pd
		}
	case KDMovingAverage:
		first := n - 1
		for i := first + o.SlowK - 1; i < len(c); i++ {
			var s float64
			for j := i - o.SlowK + 1; j <= i; j++ {
				s += rsv[j]
			}
			k[i] = s / float64(o.SlowK)
		}
		kFirst := first + o.SlowK - 1
		for i := kFirst + o.SlowD - 1; i < len(c); i++ {
			var s float64
			for j := i - o.SlowD + 1; j <= i; j++ {
				s += k[j]
			}
			d[i] = s / float64(o.SlowD)
		}
	}
	return &KDResult{listOfFloats("K", k), listOfFloats("D", d)}, nil
}
