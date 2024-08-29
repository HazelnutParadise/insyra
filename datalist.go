package insyra

import (
	"fmt"
	"sort"
	"time"

	"github.com/HazelnutParadise/Go-Utils/conv"
	"github.com/HazelnutParadise/Go-Utils/sliceutil"
)

// DataList is a generic dynamic data list
type DataList struct {
	data []interface{}
}

// IDataList defines the behavior expected from a DataList
type IDataList interface {
	Append(value interface{})
	Get(index int) interface{}
	Pop() interface{}
	Len() int
	Sort(acending ...bool)
	Reverse()
	Max() interface{}
	Min() interface{}
	Mean() float64
	GMean() float64
	Median() float64
	Mode() interface{}
	Stdev() float64
	Variance() float64
	Range() float64
	IQR() float64
	Skewness() float64
	Kurtosis() float64
	ToF64Slice() []float64
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
		data: flatData,
	}
}

// Append adds a new value to the DataList.
// The value can be of any type.
// The value is appended to the end of the DataList.
func (dl *DataList) Append(value interface{}) {
	dl.data = append(dl.data, value)
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
	return n
}

func (dl *DataList) Len() int {
	return len(dl.data)
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
