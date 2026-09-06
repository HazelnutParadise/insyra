package stats

import (
	"math"
	"sort"

	"github.com/HazelnutParadise/insyra"
)

// The parametric tests below take every observation through numericSlice so a
// blank, text, NaN or Inf cell is an error naming its row instead of a cell
// that DataList.Mean skips while DataList.Len still counts it — which used to
// turn p = 0.074 into p = 0.028 on [1, 2, nil, 3].

// testSeries reads one series for a hypothesis test. label names it in errors.
func testSeries(dl insyra.IDataList, label string) ([]float64, error) {
	values, _, err := numericSlice(dl, label)
	return values, err
}

// meanOfF64 mirrors DataList.Mean: an in-order sum divided by the count.
func meanOfF64(values []float64) float64 {
	if len(values) == 0 {
		return math.NaN()
	}
	var sum float64
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

// sampleVarianceF64 mirrors DataList.Var: two passes, n-1 denominator, NaN
// below two observations. Same summation order, so results are bit-identical.
func sampleVarianceF64(values []float64) float64 {
	if len(values) < 2 {
		return math.NaN()
	}
	mean := meanOfF64(values)
	var numerator float64
	for _, v := range values {
		numerator += (v - mean) * (v - mean)
	}
	return numerator / float64(len(values)-1)
}

// medianOfF64 mirrors DataList.Median on an all-numeric series.
func medianOfF64(values []float64) float64 {
	if len(values) == 0 {
		return math.NaN()
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	mid := len(sorted) / 2
	if len(sorted)%2 == 0 {
		return (sorted[mid-1] + sorted[mid]) / 2
	}
	return sorted[mid]
}
