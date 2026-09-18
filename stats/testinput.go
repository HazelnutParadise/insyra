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
// The list goes through asDataList first, so any IDataList implementation is
// accepted and a nil or typed nil list reads as empty instead of panicking.
func testSeries(dl insyra.IDataList, label string) ([]float64, error) {
	values, _, err := numericSlice(asDataList(dl), label)
	return values, err
}

// testSeriesPair reads two series for a two-sample test as one step. Reading
// them with two testSeries calls locks each list on its own, so a writer that
// changes both together under insyra.AtomicDoAll can land between the two
// reads and the test then mixes one sample's old state with the other's new
// one. v0.3.2 took both under one AtomicDoAll and so does this.
//
// Only the snapshot is taken under the lock. DataList.Data() returns a copy,
// so the numeric validation — which allocates and formats errors — runs
// afterwards and does not hold two actors while it does.
func testSeriesPair(a, b insyra.IDataList, labelA, labelB string) ([]float64, []float64, error) {
	dlA := asDataList(a)
	dlB := asDataList(b)

	var rawA, rawB []any
	insyra.AtomicDoAll(func() {
		rawA = dlA.Data()
		rawB = dlB.Data()
	}, dlA, dlB)

	valuesA, err := numericValues(rawA, labelA)
	if err != nil {
		return nil, nil, err
	}
	valuesB, err := numericValues(rawB, labelB)
	if err != nil {
		return nil, nil, err
	}
	return valuesA, valuesB, nil
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
