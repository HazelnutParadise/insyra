package ta

import (
	"errors"
	"fmt"
	"math"

	"github.com/HazelnutParadise/insyra"
)

// barsWithVolume reads high, low, close and volume of equal length.
func barsWithVolume(high, low, close, volume insyra.IDataList) (h, l, c, v []float64, err error) {
	if h, l, c, err = barSeries(high, low, close); err != nil {
		return
	}
	if v, err = volumeSeries(volume); err != nil {
		return
	}
	if len(v) != len(c) {
		return nil, nil, nil, nil, fmt.Errorf("volume has %d rows but close has %d", len(v), len(c))
	}
	return
}

// moneyFlowVolume is the volume of each row weighted by where the close lies in
// the day's range: ((close - low) - (high - close)) / (high - low) * volume, from
// +volume at the high to -volume at the low, and 0 when high equals low.
func moneyFlowVolume(h, l, c, v []float64) []float64 {
	out := make([]float64, len(c))
	for i := range c {
		if h[i] != l[i] {
			out[i] = ((c[i] - l[i]) - (h[i] - c[i])) / (h[i] - l[i]) * v[i]
		}
	}
	return out
}

// MFI returns the money flow index over n rows (usually 14): an RSI of the
// typical price (high + low + close) / 3 weighted by volume, from 0 to 100. A
// window with no negative flow is 100 and one with no flow at all is NaN. The
// first n rows are NaN.
func MFI(high, low, close, volume insyra.IDataList, n int) (*insyra.DataList, error) {
	if err := checkPeriod("n", n, 1); err != nil {
		return nil, err
	}
	h, l, c, v, err := barsWithVolume(high, low, close, volume)
	if err != nil {
		return nil, err
	}
	tp := make([]float64, len(c))
	for i := range c {
		tp[i] = (h[i] + l[i] + c[i]) / 3
	}
	out := nanSeries(len(c))
	for i := n; i < len(c); i++ {
		var pos, neg float64
		for j := i - n + 1; j <= i; j++ {
			switch flow := tp[j] * v[j]; {
			case tp[j] > tp[j-1]:
				pos += flow
			case tp[j] < tp[j-1]:
				neg += flow
			}
		}
		if pos+neg != 0 {
			out[i] = 100 * pos / (pos + neg)
		}
	}
	return listOfFloats("MFI", out), nil
}

// AccumulationDistribution returns Marc Chaikin's accumulation/distribution
// line: the running total of the money flow volume, which adds the whole
// volume when the close is at the high, subtracts it at the low, and 0 when
// high equals low. It equals TA-Lib's AD and starts on row 0.
func AccumulationDistribution(high, low, close, volume insyra.IDataList) (*insyra.DataList, error) {
	h, l, c, v, err := barsWithVolume(high, low, close, volume)
	if err != nil {
		return nil, err
	}
	return listOfFloats("AccumulationDistribution", adLine(h, l, c, v)), nil
}

func adLine(h, l, c, v []float64) []float64 {
	flow := moneyFlowVolume(h, l, c, v)
	out := make([]float64, len(c))
	var sum float64
	for i, f := range flow {
		sum += f
		out[i] = sum
	}
	return out
}

// CMF returns Chaikin money flow over n rows (usually 20): the sum of the money
// flow volume over the sum of the volume, from -1 to 1. A window with no volume
// is NaN. The first n-1 rows are NaN.
func CMF(high, low, close, volume insyra.IDataList, n int) (*insyra.DataList, error) {
	if err := checkPeriod("n", n, 1); err != nil {
		return nil, err
	}
	h, l, c, v, err := barsWithVolume(high, low, close, volume)
	if err != nil {
		return nil, err
	}
	flow := moneyFlowVolume(h, l, c, v)
	out := nanSeries(len(c))
	for i := n - 1; i < len(c); i++ {
		var f, vol float64
		for j := i - n + 1; j <= i; j++ {
			f, vol = f+flow[j], vol+v[j]
		}
		if vol != 0 {
			out[i] = f / vol
		}
	}
	return listOfFloats("CMF", out), nil
}

// ChaikinOscillator returns the Chaikin oscillator: the fast EMA of the
// accumulation/distribution line minus the slow one (usually 3 and 10). As in
// TA-Lib both EMAs start from the first value of the line, and it is reported
// from row slow-1.
func ChaikinOscillator(high, low, close, volume insyra.IDataList, fast, slow int) (*insyra.DataList, error) {
	if err := checkPeriod("fast", fast, 1); err != nil {
		return nil, err
	}
	if slow <= fast {
		return nil, fmt.Errorf("slow (%d) must be longer than fast (%d)", slow, fast)
	}
	h, l, c, v, err := barsWithVolume(high, low, close, volume)
	if err != nil {
		return nil, err
	}
	ad := adLine(h, l, c, v)
	out := nanSeries(len(c))
	if len(ad) == 0 {
		return listOfFloats("ChaikinOscillator", out), nil
	}
	fk, sk := 2/float64(fast+1), 2/float64(slow+1)
	fastEMA, slowEMA := ad[0], ad[0]
	for i := 1; i < len(ad); i++ {
		fastEMA += fk * (ad[i] - fastEMA)
		slowEMA += sk * (ad[i] - slowEMA)
		if i >= slow-1 {
			out[i] = fastEMA - slowEMA
		}
	}
	if slow == 1 && len(ad) > 0 {
		out[0] = 0
	}
	return listOfFloats("ChaikinOscillator", out), nil
}

// EaseOfMovementOptions are the optional settings of EaseOfMovement.
type EaseOfMovementOptions struct {
	// Scale divides the volume so the values are of a readable size. Default
	// 100,000,000, the value StockCharts uses.
	Scale float64
}

// EaseOfMovement returns Richard Arms' ease of movement over n rows (usually
// 14): the n-row simple average of how far the midpoint of the day's range moved
// from the previous row's, divided by the box ratio (volume / Scale) / (high -
// low), so a big move on little volume reads high. A row with zero volume or a
// zero range is NaN. The first n rows are NaN.
func EaseOfMovement(high, low, volume insyra.IDataList, n int, opts ...EaseOfMovementOptions) (*insyra.DataList, error) {
	if err := checkPeriod("n", n, 1); err != nil {
		return nil, err
	}
	if len(opts) > 1 {
		return nil, errors.New("EaseOfMovement accepts at most one EaseOfMovementOptions")
	}
	o := EaseOfMovementOptions{Scale: 1e8}
	if len(opts) == 1 && opts[0].Scale != 0 {
		o = opts[0]
	}
	if math.IsNaN(o.Scale) || math.IsInf(o.Scale, 0) || o.Scale < 0 {
		return nil, errors.New("option Scale must be a finite number above 0")
	}
	h, l, err := highLowSeries(high, low)
	if err != nil {
		return nil, err
	}
	v, err := volumeSeries(volume)
	if err != nil {
		return nil, err
	}
	if len(v) != len(h) {
		return nil, fmt.Errorf("volume has %d rows but high has %d", len(v), len(h))
	}
	raw := nanSeries(len(h))
	for i := 1; i < len(h); i++ {
		if v[i] == 0 || h[i] == l[i] {
			continue
		}
		distance := (h[i]+l[i])/2 - (h[i-1]+l[i-1])/2
		box := (v[i] / o.Scale) / (h[i] - l[i])
		raw[i] = distance / box
	}
	out := nanSeries(len(h))
	for i := n; i < len(h); i++ {
		var s float64
		ok := true
		for j := i - n + 1; j <= i; j++ {
			if math.IsNaN(raw[j]) {
				ok = false
				break
			}
			s += raw[j]
		}
		if ok {
			out[i] = s / float64(n)
		}
	}
	return listOfFloats("EaseOfMovement", out), nil
}

// NVI returns Paul Dysart's negative volume index: it starts at 1000 and, on a
// row whose volume is lower than the previous row's, changes by the price's
// percentage change; otherwise it stays. It follows what the price does on quiet
// days. Row 0 is 1000.
func NVI(close, volume insyra.IDataList) (*insyra.DataList, error) {
	return volumeIndex("NVI", close, volume, func(v, prev float64) bool { return v < prev })
}

// PVI returns the positive volume index, NVI's counterpart: it changes by the
// price's percentage change on rows whose volume is higher than the previous
// row's. Row 0 is 1000.
func PVI(close, volume insyra.IDataList) (*insyra.DataList, error) {
	return volumeIndex("PVI", close, volume, func(v, prev float64) bool { return v > prev })
}

func volumeIndex(name string, close, volume insyra.IDataList, move func(v, prev float64) bool) (*insyra.DataList, error) {
	c, err := positiveCloseSeries(close, "close")
	if err != nil {
		return nil, err
	}
	v, err := volumeSeries(volume)
	if err != nil {
		return nil, err
	}
	if len(v) != len(c) {
		return nil, fmt.Errorf("volume has %d rows but close has %d", len(v), len(c))
	}
	out := make([]float64, len(c))
	for i := range c {
		switch {
		case i == 0:
			out[i] = 1000
		case move(v[i], v[i-1]):
			out[i] = out[i-1] * (c[i] / c[i-1])
		default:
			out[i] = out[i-1]
		}
	}
	return listOfFloats(name, out), nil
}

// positiveCloseSeries reads a series of prices that must be positive.
func positiveCloseSeries(dl insyra.IDataList, label string) ([]float64, error) {
	p, err := numericSeries(dl, label)
	if err != nil {
		return nil, err
	}
	for i, v := range p {
		if v <= 0 {
			return nil, fmt.Errorf("%s holds %v at row index %d; prices must be positive", label, v, i)
		}
	}
	return p, nil
}
