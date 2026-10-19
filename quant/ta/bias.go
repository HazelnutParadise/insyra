package ta

import (
	"errors"
	"math"

	"github.com/HazelnutParadise/insyra"
)

// Bias returns the deviation of each price from its n-day simple moving
// average, price / SMA(n) - 1. The first n-1 values, which lack a full window,
// are NaN.
func Bias(prices insyra.IDataList, n int) (*insyra.DataList, error) {
	if n < 1 {
		return nil, errors.New("n must be at least 1")
	}
	p, err := numericSeries(prices, "prices")
	if err != nil {
		return nil, err
	}
	b := biasValues(p, n)
	out := make([]any, len(b))
	for i, v := range b {
		out[i] = v
	}
	return insyra.NewDataList(out...), nil
}

func biasValues(p []float64, n int) []float64 {
	out := make([]float64, len(p))
	sum := 0.0
	for i, v := range p {
		out[i] = math.NaN()
		sum += v
		if i >= n {
			sum -= p[i-n]
		}
		if i >= n-1 {
			if mean := sum / float64(n); mean != 0 {
				out[i] = v/mean - 1
			}
		}
	}
	return out
}

// BiasZ returns the bias divided by the standard deviation of the bias over the
// previous window days, so a deviation is measured against how much that stock
// usually deviates. The standard deviation uses only days before the current
// one, never the current day's own bias, so nothing is seen ahead of time. A
// value is NaN until n-1+window earlier biases exist or when that deviation is
// 0.
func BiasZ(prices insyra.IDataList, n, window int) (*insyra.DataList, error) {
	if n < 1 || window < 2 {
		return nil, errors.New("n must be at least 1 and window at least 2")
	}
	p, err := numericSeries(prices, "prices")
	if err != nil {
		return nil, err
	}
	b := biasValues(p, n)
	out := make([]any, len(b))
	for t := range out {
		out[t] = math.NaN()
		if t-window < n-1 {
			continue
		}
		hist := b[t-window : t]
		mean := 0.0
		for _, v := range hist {
			mean += v
		}
		mean /= float64(window)
		ss := 0.0
		for _, v := range hist {
			ss += (v - mean) * (v - mean)
		}
		if sd := math.Sqrt(ss / float64(window-1)); sd > 0 {
			out[t] = b[t] / sd
		}
	}
	return insyra.NewDataList(out...), nil
}
