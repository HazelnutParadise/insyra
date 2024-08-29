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
	Pop() (interface{}, error)
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

func (dl *DataList) Append(value interface{}) {
	dl.data = append(dl.data, value)
}
