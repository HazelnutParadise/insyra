package insyra

import "github.com/HazelnutParadise/Go-Utils/sliceutil"

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
	Sort(acending ...bool) error
	Reverse()
	Max() (interface{}, error)
	Min() (interface{}, error)
	Mean() (float64, error)
	GMean() (float64, error)
	Median() (float64, error)
	Mode() (interface{}, error)
	Stdev() (float64, error)
	Variance() (float64, error)
	Range() (float64, error)
	IQR() (float64, error)
	Skewness() (float64, error)
	Kurtosis() (float64, error)
	ToF64Slice() ([]float64, error)
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

func (dl *DataList) Sort(acending ...bool) error {
	// TODO
	// 支援時間排序
	// 支援字母排序
	// 支援數字排序(sliceutil.Sort)
	// 支援混合排序(Aa-Zz, 0-9, 時間)
	return nil
}

func (dl *DataList) Reverse() {
	sliceutil.Reverse(dl.data)
}
