package insyra

import "math"

// numericCells converts every cell of data to float64 without writing anything
// back. It is the single read path for the DataList methods that transform or
// reduce numbers, so that none of them can turn an unreadable cell into 0 or
// rewrite part of the list before discovering a cell it cannot read.
//
// With allowMissing, nil and NaN cells convert to NaN and are not failures;
// every other non-numeric cell is. Without it, nil and NaN are failures too.
// On failure the 1-based row of the first offending cell is returned with
// ok == false and the caller must leave the list untouched.
func numericCells(data []any, allowMissing bool) (values []float64, badRow int, ok bool) {
	values = make([]float64, len(data))
	for i, v := range data {
		if allowMissing && isMissing(v) {
			values[i] = math.NaN()
			continue
		}
		f, isNum := ToFloat64Safe(v)
		if !isNum || (!allowMissing && math.IsNaN(f)) {
			return nil, i + 1, false
		}
		values[i] = f
	}
	return values, 0, true
}

// observedStats summarises the non-NaN entries of values with the same
// formulas Mean, Min, Max and Stdev use (sum/n, sample variance over n-1), so a
// fully numeric list gets bit-identical results, while nil/NaN cells are
// simply not counted. stdev is NaN when fewer than two values are observed.
func observedStats(values []float64) (n int, min, max, mean, stdev float64) {
	min, max, stdev = math.NaN(), math.NaN(), math.NaN()
	var sum float64
	for _, v := range values {
		if math.IsNaN(v) {
			continue
		}
		if n == 0 || v < min {
			min = v
		}
		if n == 0 || v > max {
			max = v
		}
		sum += v
		n++
	}
	if n == 0 {
		return n, min, max, math.NaN(), stdev
	}
	mean = sum / float64(n)
	if n < 2 {
		return n, min, max, mean, stdev
	}
	var ss float64
	for _, v := range values {
		if math.IsNaN(v) {
			continue
		}
		ss += (v - mean) * (v - mean)
	}
	return n, min, max, mean, math.Sqrt(ss / float64(n-1))
}
