package insyra

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/HazelnutParadise/Go-Utils/conv"
	"github.com/HazelnutParadise/Go-Utils/sliceutil"
)

// DataList is a generic dynamic data list
type DataList struct {
	data                  []interface{}
	creationTimestamp     int64
	lastModifiedTimestamp int64
}

// IDataList defines the behavior expected from a DataList
type IDataList interface {
	GetCreationTimestamp() int64
	GetLastModifiedTimestamp() int64
	updateTimestamp()
	Data() []interface{}
	Append(value interface{})
	Get(index int) interface{}
	Pop() interface{}
	Drop(index int)
	DropAll(...interface{})
	Clear()
	ClearStrings()
	ClearNumbers()
	Len() int
	Sort(acending ...bool)
	Reverse()
	Max() interface{}
	Min() interface{}
	Mean() interface{}
	GMean() interface{}
	Median() interface{}
	Mode() interface{}
	Stdev() interface{}
	Variance() interface{}
	Range() interface{}
	Quartile(int) interface{}
	IQR() interface{}
	Skewness() interface{}
	Kurtosis() interface{}
	ToF64Slice() []float64
}

// Data returns the data stored in the DataList.
func (dl *DataList) Data() []interface{} {
	return dl.data
}

// NewDataList creates a new DataList, supporting both slice and variadic inputs,
// and flattens the input before storing it.
func NewDataList(values ...interface{}) *DataList {
	// Flatten the input values using sliceutil.Flatten with the specified generic type
	flatData, err := sliceutil.Flatten[interface{}](values)
	if err != nil {
		flatData = values
	}
	return &DataList{
		data:                  flatData,
		creationTimestamp:     time.Now().Unix(),
		lastModifiedTimestamp: time.Now().Unix(),
	}
}

// Append adds a new value to the DataList.
// The value can be of any type.
// The value is appended to the end of the DataList.
func (dl *DataList) Append(value interface{}) {
	dl.data = append(dl.data, value)
	dl.updateTimestamp()
}

// Get retrieves the value at the specified index in the DataList.
// Supports negative indexing.
// Returns nil if the index is out of bounds.
// Returns the value at the specified index.
func (dl *DataList) Get(index int) interface{} {
	if index < -len(dl.data) || index >= len(dl.data) {
		return nil
	}
	return dl.data[index]
}

// Pop removes and returns the last element from the DataList.
// Returns the last element.
// Returns nil if the DataList is empty.
func (dl *DataList) Pop() interface{} {
	n, err := sliceutil.Drt_PopFrom(&dl.data)
	if err != nil {
		return nil
	}
	dl.updateTimestamp()
	return n
}

// Drop removes the element at the specified index from the DataList and updates the timestamp.
// Returns an error if the index is out of bounds.
func (dl *DataList) Drop(index int) {
	if index < 0 {
		index += len(dl.data)
	}
	if index >= len(dl.data) {
		return
	}
	dl.data = append(dl.data[:index], dl.data[index+1:]...)
	dl.updateTimestamp()
}

// DropAll removes all occurrences of the specified values from the DataList.
// Supports multiple values to drop.
func (dl *DataList) DropAll(toDrop ...interface{}) {
	for _, v := range toDrop {
		for i := 0; i < len(dl.data); i++ {
			if dl.data[i] == v {
				dl.Drop(i)
				dl.updateTimestamp()
				i--
			}
		}
	}
}

// Clear removes all elements from the DataList and updates the timestamp.
func (dl *DataList) Clear() {
	dl.data = []interface{}{}
	dl.updateTimestamp()
}

func (dl *DataList) Len() int {
	return len(dl.data)
}

// ClearStrings removes all string elements from the DataList and updates the timestamp.
func (dl *DataList) ClearStrings() {
	filteredData := dl.data[:0] // Create a new slice with the same length as the original

	for _, v := range dl.data {
		// If the element is not a string, keep it
		if _, ok := v.(string); !ok {
			filteredData = append(filteredData, v)
		}
	}

	dl.data = filteredData
	dl.updateTimestamp()
}

// ClearNumbers removes all numeric elements (int, float, etc.) from the DataList and updates the timestamp.
func (dl *DataList) ClearNumbers() {
	filteredData := dl.data[:0] // 创建一个新的切片，容量与现有切片相同

	for _, v := range dl.data {
		// 只保留不是数字的元素
		switch v.(type) {
		case int, int8, int16, int32, int64:
		case uint, uint8, uint16, uint32, uint64:
		case float32, float64:
			// 以上类型为数字，跳过
		default:
			filteredData = append(filteredData, v)
		}
	}

	dl.data = filteredData
	dl.updateTimestamp()
}

// Sort sorts the DataList using a mixed sorting logic.
// It handles string, numeric (including all integer and float types), and time data types.
// If sorting fails, it restores the original order.
func (dl *DataList) Sort(ascending ...bool) {
	if len(dl.data) == 0 {
		return
	}

	// Save the original order
	originalData := make([]interface{}, len(dl.data))
	copy(originalData, dl.data)

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Sorting failed, restoring original order:", r)
			dl.data = originalData
		}
	}()

	ascendingOrder := true
	if len(ascending) > 0 {
		ascendingOrder = ascending[0]
	}

	// Mixed sorting
	sort.SliceStable(dl.data, func(i, j int) bool {
		v1 := dl.data[i]
		v2 := dl.data[j]

		switch v1 := v1.(type) {
		case string:
			if v2, ok := v2.(string); ok {
				return (v1 < v2) == ascendingOrder
			}
		case int, int8, int16, int32, int64:
			v1Float := ToFloat64(v1)
			if v2Float, ok := ToFloat64Safe(v2); ok {
				return (v1Float < v2Float) == ascendingOrder
			}
		case uint, uint8, uint16, uint32, uint64:
			v1Float := ToFloat64(v1)
			if v2Float, ok := ToFloat64Safe(v2); ok {
				return (v1Float < v2Float) == ascendingOrder
			}
		case float32, float64:
			v1Float := ToFloat64(v1)
			if v2Float, ok := ToFloat64Safe(v2); ok {
				return (v1Float < v2Float) == ascendingOrder
			}
		case time.Time:
			if v2, ok := v2.(time.Time); ok {
				return v1.Before(v2) == ascendingOrder
			}
		}

		// Fallback: compare as strings
		return fmt.Sprint(v1) < fmt.Sprint(v2)
	})
}

// Reverse reverses the order of the elements in the DataList.
func (dl *DataList) Reverse() {
	sliceutil.Reverse(dl.data)
}

// ======================== Statistics ========================

// Max returns the maximum value in the DataList.
// Returns the maximum value.
// Returns nil if the DataList is empty.
// Max returns the maximum value in the DataList, or nil if the data types cannot be compared.
func (dl *DataList) Max() interface{} {
	if len(dl.data) == 0 {
		return nil
	}

	var max interface{}

	for _, v := range dl.data {
		if max == nil {
			max = v
			continue
		}

		switch maxVal := max.(type) {
		case int:
			if val, ok := v.(int); ok && val > maxVal {
				max = val
			} else if !ok {
				return nil
			}
		case float64:
			if val, ok := v.(float64); ok && val > maxVal {
				max = val
			} else if intVal, ok := v.(int); ok && float64(intVal) > maxVal {
				max = float64(intVal)
			} else if !ok {
				return nil
			}
		case string:
			if val, ok := v.(string); ok && conv.ParseF64(val) > conv.ParseF64(maxVal) {
				max = val
			} else if !ok {
				return nil
			}
		default:
			return nil
		}
	}

	return max
}

// Min returns the minimum value in the DataList.
// Returns the minimum value.
// Returns nil if the DataList is empty.
// Min returns the minimum value in the DataList, or nil if the data types cannot be compared.
func (dl *DataList) Min() interface{} {
	if len(dl.data) == 0 {
		return nil
	}

	var min interface{}

	for _, v := range dl.data {
		if min == nil {
			min = v
			continue
		}

		switch minVal := min.(type) {
		case int:
			if val, ok := v.(int); ok && val < minVal {
				min = val
			} else if !ok {
				return nil
			}
		case float64:
			if val, ok := v.(float64); ok && val < minVal {
				min = val
			} else if intVal, ok := v.(int); ok && float64(intVal) < minVal {
				min = float64(intVal)
			} else if !ok {
				return nil
			}
		case string:
			if val, ok := v.(string); ok && conv.ParseF64(val) < conv.ParseF64(minVal) {
				min = val
			} else if !ok {
				return nil
			}
		default:
			return nil
		}
	}

	return min
}

// Mean calculates the arithmetic mean of the DataList.
// Returns the arithmetic mean.
// Returns nil if the DataList is empty.
// Mean returns the arithmetic mean of the DataList.
func (dl *DataList) Mean() interface{} {
	if len(dl.data) == 0 {
		return nil
	}

	var sum float64
	for _, v := range dl.data {
		if val, ok := ToFloat64Safe(v); ok {
			sum += val
		}
	}

	return sum / float64(len(dl.data))
}

// GMean calculates the geometric mean of the DataList.
// Returns the geometric mean.
// Returns nil if the DataList is empty.
// GMean returns the geometric mean of the DataList.
func (dl *DataList) GMean() interface{} {
	if len(dl.data) == 0 {
		return nil
	}

	product := 1.0
	for _, v := range dl.data {
		if val, ok := ToFloat64Safe(v); ok {
			product *= val
		} else {
			return nil
		}
	}

	return math.Pow(product, 1.0/float64(len(dl.data)))
}

// Median calculates the median of the DataList.
// Returns the median.
// Returns nil if the DataList is empty.
// Median returns the median of the DataList.
func (dl *DataList) Median() interface{} {
	if len(dl.data) == 0 {
		return nil
	}

	sortedData := make([]interface{}, len(dl.data))
	copy(sortedData, dl.data)
	dl.Sort()

	mid := len(sortedData) / 2
	if len(sortedData)%2 == 0 {
		return (ToFloat64(sortedData[mid-1]) + ToFloat64(sortedData[mid])) / 2
	}

	return ToFloat64(sortedData[mid])
}

// Mode calculates the mode of the DataList.
// Returns the mode.
// Returns nil if the DataList is empty.
// Mode returns the mode of the DataList.
func (dl *DataList) Mode() interface{} {
	if len(dl.data) == 0 {
		return nil
	}

	freqMap := make(map[interface{}]int)
	for _, v := range dl.data {
		freqMap[v]++
	}

	var mode interface{}
	maxFreq := 0
	for k, v := range freqMap {
		if v > maxFreq {
			mode = k
			maxFreq = v
		}
	}

	return mode
}

// Stdev calculates the standard deviation of the DataList.
// Returns the standard deviation.
// Returns nil if the DataList is empty.
// Stdev returns the standard deviation of the DataList.
func (dl *DataList) Stdev() interface{} {
	if len(dl.data) == 0 {
		return nil
	}
	variance := dl.Variance()
	if variance == nil {
		return nil
	}
	return math.Sqrt(ToFloat64(variance))
}

// Variance calculates the variance of the DataList.
// Returns the variance.
// Returns nil if the DataList is empty.
// Variance returns the variance of the DataList.
func (dl *DataList) Variance() interface{} {
	n := float64(dl.Len())
	if n == 0.0 {
		return nil
	}
	m := dl.Mean()
	mean, ok := ToFloat64Safe(m)
	if !ok {
		return nil
	}

	y1 := 1.0 / n
	y2 := 0.0
	for i := 0; i < len(dl.data); i++ {
		xi, ok := ToFloat64Safe(dl.data[i])
		if !ok {
			return nil
		}
		y2 += math.Pow(xi-mean, 2)
	}
	return y1 * y2
}

// Range calculates the range of the DataList.
// Returns the range.
// Returns nil if the DataList is empty.
// Range returns the range of the DataList.
func (dl *DataList) Range() interface{} {
	if len(dl.data) == 0 {
		return nil
	}

	max := ToFloat64(dl.Max())
	min := ToFloat64(dl.Min())

	return max - min
}

// Quartile calculates the quartile based on the input value (1 to 3).
// 1 corresponds to the first quartile (Q1), 2 to the median (Q2), and 3 to the third quartile (Q3).
func (dl *DataList) Quartile(q int) interface{} {
	if len(dl.data) == 0 {
		return nil
	}
	if q < 1 || q > 3 {
		return nil
	}

	// Convert the DataList to a slice of float64 for numeric operations
	numericData := dl.ToF64Slice()

	// Sort the data
	sort.Float64s(numericData)

	n := len(numericData)

	var pos float64
	var lower, upper float64

	switch q {
	case 1:
		pos = 0.25 * float64(n-1)
	case 2:
		pos = 0.5 * float64(n-1)
	case 3:
		pos = 0.75 * float64(n-1)
	}

	// Get the index for lower and upper bounds
	lowerIndex := int(math.Floor(pos))
	upperIndex := int(math.Ceil(pos))

	// Handle the case where the position is exactly an integer
	if lowerIndex == upperIndex {
		return numericData[lowerIndex]
	}

	// Interpolate between the lower and upper bounds
	lower = numericData[lowerIndex]
	upper = numericData[upperIndex]

	return lower + (pos-float64(lowerIndex))*(upper-lower)
}

// IQR calculates the interquartile range of the DataList.
// Returns the interquartile range.
// Returns nil if the DataList is empty.
func (dl *DataList) IQR() interface{} {
	if len(dl.data) == 0 {
		return nil
	}
	q3, ok := ToFloat64Safe(dl.Quartile(3))
	if !ok {
		return nil
	}
	q1, ok := ToFloat64Safe(dl.Quartile(1))
	if !ok {
		return nil
	}
	return q3 - q1
}

// Skewness calculates the skewness of the DataList.
// Returns the skewness.
// Returns nil if the DataList is empty or the skewness cannot be calculated.
// 錯誤！
// Skewness calculates the skewness of the DataList using Gonum's Skew function.
func (dl *DataList) Skewness() interface{} {
	data := dl.ToF64Slice() // 将 DataList 转换为 float64 切片
	if len(data) == 0 {
		return nil
	}
	n := float64(len(data))
	mean, ok := ToFloat64Safe(dl.Mean())
	if !ok {
		return nil
	}
	stdev, ok := ToFloat64Safe(dl.Stdev())
	if !ok {
		return nil
	}
	y := 0.0
	denominator1 := (n - 1) * (n - 2)
	if denominator1 == 0 || stdev == 0 {
		return nil
	}
	for i := 0; i < len(data); i++ {
		xi, ok := ToFloat64Safe(data[i])
		if !ok {
			return nil
		}
		y += math.Pow((xi-mean)/stdev, 3)
	}
	return (n / denominator1) * y
}

// Kurtosis calculates the kurtosis of the DataList.
// Returns the kurtosis.
// Returns nil if the DataList is empty or the kurtosis cannot be calculated.
// 錯誤！
func (dl *DataList) Kurtosis() interface{} {
	n := float64(dl.Len())
	if n == 0.0 {
		return nil
	}
	mean, ok := ToFloat64Safe(dl.Mean())
	if !ok {
		return nil
	}
	stdev, ok := ToFloat64Safe(dl.Stdev())
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
	for i := 0; i < len(dl.data); i++ {
		xi, ok := ToFloat64Safe(dl.data[i])
		if !ok {
			return nil
		}
		y2 += math.Pow((xi-mean)/stdev, 4)
	}
	y3 := (3 * math.Pow(n-1, 2)) / denominator2

	return y1*y2 - y3
}

// ======================== Conversion ========================

// ToF64Slice converts the DataList to a float64 slice.
// Returns the float64 slice.
// Returns nil if the DataList is empty.
// ToF64Slice converts the DataList to a float64 slice.
func (dl *DataList) ToF64Slice() []float64 {
	if len(dl.data) == 0 {
		return nil
	}

	floatData := make([]float64, len(dl.data))
	for i, v := range dl.data {
		floatData[i] = ToFloat64(v)
	}

	return floatData
}

// ======================== Timestamp ========================

// GetCreationTimestamp returns the creation time of the DataList in Unix timestamp.
func (dl *DataList) GetCreationTimestamp() int64 {
	return dl.creationTimestamp
}

// GetLastModifiedTimestamp returns the last updated time of the DataList in Unix timestamp.
func (dl *DataList) GetLastModifiedTimestamp() int64 {
	return dl.lastModifiedTimestamp
}

// updateTimestamp updates the lastModifiedTimestamp to the current Unix time.
func (dl *DataList) updateTimestamp() {
	dl.lastModifiedTimestamp = time.Now().Unix()
}
