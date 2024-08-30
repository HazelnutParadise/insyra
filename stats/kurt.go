package stats

import (
	"math"

	"github.com/HazelnutParadise/insyra"
)

// Kurtosis calculates the kurtosis(sample) of the DataList.
// Returns the kurtosis.
// Returns nil if the DataList is empty or the kurtosis cannot be calculated.
// 錯誤！
func Kurtosis(dl insyra.IDataList) interface{} {
	n := float64(dl.Len())
	if n == 0.0 {
		return nil
	}
	mean, ok := insyra.ToFloat64Safe(dl.Mean())
	if !ok {
		return nil
	}
	stdev, ok := insyra.ToFloat64Safe(dl.Stdev())
	if !ok {
		return nil
	}
	if stdev == 0 {
		return nil
	}
	denominator1 := (n - 1) * (n - 2) * (n - 3)
	if denominator1 == 0 {
		return nil
	}
	denominator2 := (n - 2) * (n - 3)
	if denominator2 == 0 {
		return nil
	}
	y1 := (n * (n + 1)) / denominator1
	y2 := 0.0
	for i := 0; i < dl.Len(); i++ {
		xi, ok := insyra.ToFloat64Safe(dl.Get(i))
		if !ok {
			return nil
		}
		y2 += math.Pow((xi-mean)/stdev, 4)
	}
	y3 := (3 * math.Pow(n-1, 2)) / denominator2

	return y1*y2 - y3
}
