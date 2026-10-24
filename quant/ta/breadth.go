package ta

import (
	"errors"
	"fmt"
	"math"

	"github.com/HazelnutParadise/insyra"
)

// breadthCounts reads the daily advancing and declining counts. A missing day
// (nil or NaN in either list) is reported as valid[t] == false; anything else
// that is not a finite, non-negative number is an error.
func breadthCounts(adv, dec insyra.IDataList) (a, d []float64, valid []bool, err error) {
	read := func(dl insyra.IDataList, label string) ([]float64, error) {
		if insyra.IsNilList(dl) {
			return nil, fmt.Errorf("%s is nil", label)
		}
		var raw []any
		dl.AtomicDo(func(l *insyra.DataList) { raw = l.Data() })
		out := make([]float64, len(raw))
		for i, v := range raw {
			if v == nil {
				out[i] = math.NaN()
				continue
			}
			f, ok := insyra.ToFloat64Safe(v)
			if !ok {
				return nil, fmt.Errorf("%s contains a non-numeric value at row index %d: %v", label, i, v)
			}
			if math.IsInf(f, 0) || f < 0 {
				return nil, fmt.Errorf("%s holds %v at row index %d; a count must be a finite number of at least 0", label, f, i)
			}
			out[i] = f
		}
		return out, nil
	}
	if a, err = read(adv, "adv"); err != nil {
		return nil, nil, nil, err
	}
	if d, err = read(dec, "dec"); err != nil {
		return nil, nil, nil, err
	}
	if len(a) != len(d) {
		return nil, nil, nil, fmt.Errorf("adv has %d rows and dec has %d", len(a), len(d))
	}
	valid = make([]bool, len(a))
	for i := range a {
		valid[i] = !math.IsNaN(a[i]) && !math.IsNaN(d[i])
	}
	return a, d, valid, nil
}

// windowedBreadth applies f to the sums of advancing and declining counts over
// the last window valid days. A missing day, and any day before window valid
// days have been seen, is NaN.
func windowedBreadth(adv, dec insyra.IDataList, window int, f func(sumAdv, sumDec float64) float64) (*insyra.DataList, error) {
	if window < 1 {
		return nil, errors.New("window must be at least 1")
	}
	a, d, valid, err := breadthCounts(adv, dec)
	if err != nil {
		return nil, err
	}
	out := make([]any, len(a))
	var days []int // positions of the valid days seen so far
	for t := range a {
		out[t] = math.NaN()
		if !valid[t] {
			continue
		}
		days = append(days, t)
		if len(days) < window {
			continue
		}
		sumA, sumD := 0.0, 0.0
		for _, i := range days[len(days)-window:] {
			sumA += a[i]
			sumD += d[i]
		}
		out[t] = f(sumA, sumD)
	}
	return insyra.NewDataList(out...), nil
}

// AdvanceDeclineRatio returns ADR: the advancing count over the last window
// valid days divided by the declining count over the same days. It is NaN when
// the declining total is 0, on a missing day, and before window valid days
// exist. The window counts valid days, so a missing day does not shorten it.
func AdvanceDeclineRatio(adv, dec insyra.IDataList, window int) (*insyra.DataList, error) {
	return windowedBreadth(adv, dec, window, func(sumAdv, sumDec float64) float64 {
		if sumDec == 0 {
			return math.NaN()
		}
		return sumAdv / sumDec
	})
}

// OverboughtOversold returns OBOS: the sum of advancing minus declining over
// the last window valid days, which is the window-period change of the
// advance-decline line. Missing days are handled as in AdvanceDeclineRatio.
// The value grows with the number of listed stocks, so it is not comparable
// across decades; NetAdvanceRatio removes that scale.
func OverboughtOversold(adv, dec insyra.IDataList, window int) (*insyra.DataList, error) {
	return windowedBreadth(adv, dec, window, func(sumAdv, sumDec float64) float64 {
		return sumAdv - sumDec
	})
}

// NetAdvanceRatio returns (sum of advancing - sum of declining) divided by
// (sum of advancing + sum of declining) over the last window valid days. It
// lies in [-1, 1] whatever the number of listed stocks. It is NaN when both
// totals are 0, on a missing day, and before window valid days exist.
func NetAdvanceRatio(adv, dec insyra.IDataList, window int) (*insyra.DataList, error) {
	return windowedBreadth(adv, dec, window, func(sumAdv, sumDec float64) float64 {
		if sumAdv+sumDec == 0 {
			return math.NaN()
		}
		return (sumAdv - sumDec) / (sumAdv + sumDec)
	})
}

// AdvanceDeclineLine returns ADL: start plus the running sum of advancing
// minus declining. A missing day is NaN and adds nothing, so the line carries
// on from its last level.
func AdvanceDeclineLine(adv, dec insyra.IDataList, start float64) (*insyra.DataList, error) {
	if math.IsNaN(start) || math.IsInf(start, 0) {
		return nil, errors.New("start must be a finite number")
	}
	a, d, valid, err := breadthCounts(adv, dec)
	if err != nil {
		return nil, err
	}
	out := make([]any, len(a))
	level := start
	for t := range a {
		if !valid[t] {
			out[t] = math.NaN()
			continue
		}
		level += a[t] - d[t]
		out[t] = level
	}
	return insyra.NewDataList(out...), nil
}
