package ta

import (
	"errors"
	"fmt"
	"math"

	"github.com/HazelnutParadise/insyra"
)

func highLowSeries(high, low insyra.IDataList) (h, l []float64, err error) {
	if h, err = numericSeries(high, "high"); err != nil {
		return
	}
	if l, err = numericSeries(low, "low"); err != nil {
		return
	}
	if len(h) != len(l) {
		return nil, nil, fmt.Errorf("high and low must have the same length, got %d and %d", len(h), len(l))
	}
	for i := range h {
		if h[i] < l[i] {
			return nil, nil, fmt.Errorf("high %v is below low %v at row index %d", h[i], l[i], i)
		}
	}
	return
}

// ChannelResult holds the three lines of a channel.
type ChannelResult struct {
	Upper  *insyra.DataList
	Middle *insyra.DataList
	Lower  *insyra.DataList
}

// Keltner returns the Keltner channel in Raschke's form, the one trading
// platforms show: the middle line is the n-row EMA of the close, and the bands
// are k times the atrN-row ATR above and below it. Rows before both the EMA
// and the ATR exist are NaN. A common setting is n 20, atrN 10 and k 2. See
// KeltnerClassic for the 1960 form.
func Keltner(high, low, close insyra.IDataList, n, atrN int, k float64) (*ChannelResult, error) {
	if err := checkPeriod("n", n, 1); err != nil {
		return nil, err
	}
	if err := checkPeriod("atrN", atrN, 1); err != nil {
		return nil, err
	}
	if math.IsNaN(k) || math.IsInf(k, 0) || k <= 0 {
		return nil, errors.New("k must be a finite number of ATRs above 0")
	}
	h, l, c, err := barSeries(high, low, close)
	if err != nil {
		return nil, err
	}
	mid := emaOf(c, n, 0)
	atr := atrOf(h, l, c, atrN)
	up, lo := nanSeries(len(c)), nanSeries(len(c))
	for i := range c {
		if !math.IsNaN(mid[i]) && !math.IsNaN(atr[i]) {
			up[i], lo[i] = mid[i]+k*atr[i], mid[i]-k*atr[i]
		}
	}
	return &ChannelResult{listOfFloats("Upper", up), listOfFloats("Middle", mid), listOfFloats("Lower", lo)}, nil
}

// KeltnerClassic returns the Keltner channel as Chester Keltner described it in
// 1960: the middle line is the n-row simple average of the typical price
// (high + low + close) / 3, and the bands are the n-row simple average of the
// day's high minus low above and below it. The first n-1 rows are NaN.
func KeltnerClassic(high, low, close insyra.IDataList, n int) (*ChannelResult, error) {
	if err := checkPeriod("n", n, 1); err != nil {
		return nil, err
	}
	h, l, c, err := barSeries(high, low, close)
	if err != nil {
		return nil, err
	}
	typical := make([]float64, len(c))
	spread := make([]float64, len(c))
	for i := range c {
		typical[i] = (h[i] + l[i] + c[i]) / 3
		spread[i] = h[i] - l[i]
	}
	mid, width := smaOf(typical, n), smaOf(spread, n)
	up, lo := nanSeries(len(c)), nanSeries(len(c))
	for i := range c {
		if !math.IsNaN(mid[i]) {
			up[i], lo[i] = mid[i]+width[i], mid[i]-width[i]
		}
	}
	return &ChannelResult{listOfFloats("Upper", up), listOfFloats("Middle", mid), listOfFloats("Lower", lo)}, nil
}

// Donchian returns the Donchian channel: the highest high and the lowest low of
// the n rows before each row, and their midpoint. The current row is left out,
// so a close above Upper is a breakout of the earlier range and not a tie with
// its own high. The first n rows are NaN.
func Donchian(high, low insyra.IDataList, n int) (*ChannelResult, error) {
	if err := checkPeriod("n", n, 1); err != nil {
		return nil, err
	}
	h, l, err := highLowSeries(high, low)
	if err != nil {
		return nil, err
	}
	up, mid, lo := nanSeries(len(h)), nanSeries(len(h)), nanSeries(len(h))
	for i := n; i < len(h); i++ {
		hi, lw := h[i-n], l[i-n]
		for j := i - n + 1; j < i; j++ {
			hi, lw = math.Max(hi, h[j]), math.Min(lw, l[j])
		}
		up[i], lo[i], mid[i] = hi, lw, (hi+lw)/2
	}
	return &ChannelResult{listOfFloats("Upper", up), listOfFloats("Middle", mid), listOfFloats("Lower", lo)}, nil
}

// SuperTrendResult holds the SuperTrend line and which way it points.
type SuperTrendResult struct {
	Line      *insyra.DataList // the lower band while Direction is 1, the upper band while it is -1
	Direction *insyra.DataList // 1 up, -1 down
}

// SuperTrend returns the SuperTrend stop line as most charting platforms draw
// it: bands of (high + low) / 2 plus and minus k times the n-row ATR, each band
// only allowed to move toward the price, and a direction that flips when the
// close crosses the band the line is on. It starts on the first row the ATR
// exists (row n), pointing down on the upper band, and rows before that are NaN.
// A common setting is n 10 and k 3.
func SuperTrend(high, low, close insyra.IDataList, n int, k float64) (*SuperTrendResult, error) {
	if err := checkPeriod("n", n, 1); err != nil {
		return nil, err
	}
	if math.IsNaN(k) || math.IsInf(k, 0) || k <= 0 {
		return nil, errors.New("k must be a finite number of ATRs above 0")
	}
	h, l, c, err := barSeries(high, low, close)
	if err != nil {
		return nil, err
	}
	atr := atrOf(h, l, c, n)
	line, dir := nanSeries(len(c)), nanSeries(len(c))
	var upper, lower, prevUpper, prevLower float64
	prevDir := 0.0
	for i := range c {
		if math.IsNaN(atr[i]) {
			continue
		}
		mid := (h[i] + l[i]) / 2
		upper, lower = mid+k*atr[i], mid-k*atr[i]
		if prevDir == 0 {
			prevDir = -1
		} else {
			if !(upper < prevUpper || c[i-1] > prevUpper) {
				upper = prevUpper
			}
			if !(lower > prevLower || c[i-1] < prevLower) {
				lower = prevLower
			}
			if prevDir == -1 {
				if c[i] > upper {
					prevDir = 1
				}
			} else if c[i] < lower {
				prevDir = -1
			}
		}
		dir[i] = prevDir
		if prevDir == 1 {
			line[i] = lower
		} else {
			line[i] = upper
		}
		prevUpper, prevLower = upper, lower
	}
	return &SuperTrendResult{listOfFloats("SuperTrend", line), listOfFloats("Direction", dir)}, nil
}

// ParabolicSAR returns Wilder's parabolic stop and reverse. start is the
// acceleration factor a trend begins with, step how much it grows each time the
// trend makes a new extreme, and maxAF its cap; Wilder's are 0.02, 0.02 and 0.2.
// The first row is NaN, and the starting direction comes from the first two
// bars as in TA-Lib's SAR.
func ParabolicSAR(high, low insyra.IDataList, start, step, maxAF float64) (*insyra.DataList, error) {
	for name, v := range map[string]float64{"start": start, "step": step, "maxAF": maxAF} {
		if math.IsNaN(v) || math.IsInf(v, 0) || v <= 0 {
			return nil, fmt.Errorf("%s must be a finite number above 0", name)
		}
	}
	if maxAF < start {
		return nil, errors.New("maxAF must not be below start")
	}
	h, l, err := highLowSeries(high, low)
	if err != nil {
		return nil, err
	}
	out := nanSeries(len(h))
	if len(h) < 2 {
		return listOfFloats("SAR", out), nil
	}
	// A start where the low fell by more than the high rose (minus directional
	// movement above plus) is a short trend; anything else is long.
	isLong := !(l[0] > l[1] && l[0]-l[1] > h[1]-h[0])
	var sar, ep float64
	af := start
	newLow, newHigh := l[0], h[0]
	if isLong {
		ep, sar = h[1], newLow
	} else {
		ep, sar = l[1], newHigh
	}
	newLow, newHigh = l[1], h[1]
	for i := 1; i < len(h); i++ {
		prevLow, prevHigh := newLow, newHigh
		newLow, newHigh = l[i], h[i]
		if isLong {
			if newLow <= sar {
				isLong = false
				sar = ep
				sar = math.Max(sar, math.Max(prevHigh, newHigh))
				out[i] = sar
				af = start
				ep = newLow
				sar += af * (ep - sar)
				sar = math.Max(sar, math.Max(prevHigh, newHigh))
			} else {
				out[i] = sar
				if newHigh > ep {
					ep = newHigh
					af = math.Min(af+step, maxAF)
				}
				sar += af * (ep - sar)
				sar = math.Min(sar, math.Min(prevLow, newLow))
			}
		} else {
			if newHigh >= sar {
				isLong = true
				sar = ep
				sar = math.Min(sar, math.Min(prevLow, newLow))
				out[i] = sar
				af = start
				ep = newHigh
				sar += af * (ep - sar)
				sar = math.Min(sar, math.Min(prevLow, newLow))
			} else {
				out[i] = sar
				if newLow < ep {
					ep = newLow
					af = math.Min(af+step, maxAF)
				}
				sar += af * (ep - sar)
				sar = math.Max(sar, math.Max(prevHigh, newHigh))
			}
		}
	}
	return listOfFloats("SAR", out), nil
}

// ChandelierResult holds the two chandelier exit lines.
type ChandelierResult struct {
	Long  *insyra.DataList // highest high of n rows minus k ATRs: the stop for a long position
	Short *insyra.DataList // lowest low of n rows plus k ATRs: the stop for a short position
}

// Chandelier returns the chandelier exit: the highest high of the last n rows
// (the current one included) minus k times the n-row ATR as the stop for a long
// position, and the lowest low plus k ATRs for a short one. A common setting is
// n 22 and k 3. Rows before the ATR exists (before row n) are NaN.
func Chandelier(high, low, close insyra.IDataList, n int, k float64) (*ChandelierResult, error) {
	if err := checkPeriod("n", n, 1); err != nil {
		return nil, err
	}
	if math.IsNaN(k) || math.IsInf(k, 0) || k <= 0 {
		return nil, errors.New("k must be a finite number of ATRs above 0")
	}
	h, l, c, err := barSeries(high, low, close)
	if err != nil {
		return nil, err
	}
	atr := atrOf(h, l, c, n)
	long, short := nanSeries(len(c)), nanSeries(len(c))
	for i := n; i < len(c); i++ {
		hi, lw := h[i], l[i]
		for j := i - n + 1; j < i; j++ {
			hi, lw = math.Max(hi, h[j]), math.Min(lw, l[j])
		}
		long[i], short[i] = hi-k*atr[i], lw+k*atr[i]
	}
	return &ChandelierResult{listOfFloats("Long", long), listOfFloats("Short", short)}, nil
}
