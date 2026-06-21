package insyra

import (
	"math"
	"reflect"
	"sort"
)

func isMissing(v any) bool {
	if v == nil {
		return true
	}
	if f, ok := v.(float64); ok && math.IsNaN(f) {
		return true
	}
	return false
}

func imputeLimit(limit []int) int {
	if len(limit) == 0 || limit[0] <= 0 {
		return 0
	}
	return limit[0]
}

func numericObservedValues(data []any) []float64 {
	values := make([]float64, 0, len(data))
	for _, v := range data {
		if isMissing(v) {
			continue
		}
		if f, ok := ToFloat64Safe(v); ok {
			values = append(values, f)
		}
	}
	return values
}

func numericObservedPoints(data []any) ([]int, []float64, bool) {
	indices := make([]int, 0, len(data))
	values := make([]float64, 0, len(data))
	for i, v := range data {
		if isMissing(v) {
			continue
		}
		f, ok := ToFloat64Safe(v)
		if !ok {
			return nil, nil, false
		}
		indices = append(indices, i)
		values = append(values, f)
	}
	return indices, values, true
}

func hasOnlyNumericObservedValues(dl *DataList) bool {
	found := false
	for _, v := range dl.data {
		if isMissing(v) {
			continue
		}
		if _, ok := ToFloat64Safe(v); !ok {
			return false
		}
		found = true
	}
	return found
}

// FillForward replaces missing values with the most recent non-missing value.
func (dl *DataList) FillForward(limit ...int) *DataList {
	defer func() {
		go dl.updateTimestamp()
	}()
	maxFill := imputeLimit(limit)
	dl.AtomicDo(func(dl *DataList) {
		var last any
		hasLast := false
		fillCount := 0
		for i, v := range dl.data {
			if isMissing(v) {
				if hasLast && (maxFill == 0 || fillCount < maxFill) {
					dl.data[i] = last
				}
				if hasLast {
					fillCount++
				}
				continue
			}
			last = v
			hasLast = true
			fillCount = 0
		}
	})
	return dl
}

// FillBackward replaces missing values with the next non-missing value.
func (dl *DataList) FillBackward(limit ...int) *DataList {
	defer func() {
		go dl.updateTimestamp()
	}()
	maxFill := imputeLimit(limit)
	dl.AtomicDo(func(dl *DataList) {
		var next any
		hasNext := false
		fillCount := 0
		for i := len(dl.data) - 1; i >= 0; i-- {
			v := dl.data[i]
			if isMissing(v) {
				if hasNext && (maxFill == 0 || fillCount < maxFill) {
					dl.data[i] = next
				}
				if hasNext {
					fillCount++
				}
				continue
			}
			next = v
			hasNext = true
			fillCount = 0
		}
	})
	return dl
}

// FillWithMean replaces missing values (NaN and nil) with the mean of observed numeric values.
func (dl *DataList) FillWithMean() *DataList {
	defer func() {
		go dl.updateTimestamp()
	}()
	dl.AtomicDo(func(dl *DataList) {
		values := numericObservedValues(dl.data)
		if len(values) == 0 {
			dl.warn("FillWithMean", "No numeric values to compute mean")
			return
		}
		var sum float64
		for _, v := range values {
			sum += v
		}
		mean := sum / float64(len(values))
		for i, v := range dl.data {
			if isMissing(v) {
				dl.data[i] = mean
			}
		}
	})
	return dl
}

// FillWithMedian replaces missing values with the median of observed numeric values.
func (dl *DataList) FillWithMedian() *DataList {
	defer func() {
		go dl.updateTimestamp()
	}()
	dl.AtomicDo(func(dl *DataList) {
		values := numericObservedValues(dl.data)
		if len(values) == 0 {
			dl.warn("FillWithMedian", "No numeric values to compute median")
			return
		}
		sort.Float64s(values)
		mid := len(values) / 2
		median := values[mid]
		if len(values)%2 == 0 {
			median = (values[mid-1] + values[mid]) / 2
		}
		for i, v := range dl.data {
			if isMissing(v) {
				dl.data[i] = median
			}
		}
	})
	return dl
}

// FillWithMode replaces missing values with the first-occurring mode of observed values.
func (dl *DataList) FillWithMode() *DataList {
	defer func() {
		go dl.updateTimestamp()
	}()
	type modeEntry struct {
		value any
		count int
		first int
	}
	dl.AtomicDo(func(dl *DataList) {
		entries := []modeEntry{}
		for i, v := range dl.data {
			if isMissing(v) {
				continue
			}
			found := false
			for j := range entries {
				if reflect.DeepEqual(entries[j].value, v) {
					entries[j].count++
					found = true
					break
				}
			}
			if !found {
				entries = append(entries, modeEntry{value: v, count: 1, first: i})
			}
		}
		if len(entries) == 0 {
			dl.warn("FillWithMode", "No non-missing values to compute mode")
			return
		}
		mode := entries[0]
		for _, entry := range entries[1:] {
			if entry.count > mode.count || entry.count == mode.count && entry.first < mode.first {
				mode = entry
			}
		}
		for i, v := range dl.data {
			if isMissing(v) {
				dl.data[i] = mode.value
			}
		}
	})
	return dl
}

// FillByInterpolation fills missing sequence values by index, unlike LinearInterpolation which evaluates y at x.
func (dl *DataList) FillByInterpolation(extrapolate ...bool) *DataList {
	defer func() {
		go dl.updateTimestamp()
	}()
	shouldExtrapolate := len(extrapolate) > 0 && extrapolate[0]
	dl.AtomicDo(func(dl *DataList) {
		indices, values, ok := numericObservedPoints(dl.data)
		if !ok {
			dl.warn("FillByInterpolation", "DataList contains non-numeric values")
			return
		}
		if len(indices) == 0 {
			dl.warn("FillByInterpolation", "No numeric values to interpolate")
			return
		}

		for p := 0; p < len(indices)-1; p++ {
			leftIdx := indices[p]
			rightIdx := indices[p+1]
			if rightIdx-leftIdx <= 1 {
				continue
			}
			leftVal := values[p]
			rightVal := values[p+1]
			step := (rightVal - leftVal) / float64(rightIdx-leftIdx)
			for i := leftIdx + 1; i < rightIdx; i++ {
				if isMissing(dl.data[i]) {
					dl.data[i] = leftVal + step*float64(i-leftIdx)
				}
			}
		}

		if !shouldExtrapolate {
			return
		}
		if len(indices) == 1 {
			for i, v := range dl.data {
				if isMissing(v) {
					dl.data[i] = values[0]
				}
			}
			return
		}

		firstIdx := indices[0]
		firstVal := values[0]
		firstStep := (values[1] - firstVal) / float64(indices[1]-firstIdx)
		for i := range firstIdx {
			if isMissing(dl.data[i]) {
				dl.data[i] = firstVal + firstStep*float64(i-firstIdx)
			}
		}

		lastPos := len(indices) - 1
		lastIdx := indices[lastPos]
		lastVal := values[lastPos]
		lastStep := (lastVal - values[lastPos-1]) / float64(lastIdx-indices[lastPos-1])
		for i := lastIdx + 1; i < len(dl.data); i++ {
			if isMissing(dl.data[i]) {
				dl.data[i] = lastVal + lastStep*float64(i-lastIdx)
			}
		}
	})
	return dl
}
