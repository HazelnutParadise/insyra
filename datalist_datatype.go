package insyra

import (
	"reflect"
	"time"
)

// DataType says what kind of values a DataList holds, judged over its values
// with missing ones (nil and NaN) left out. It answers the question an
// analysis asks before it runs — can this column be averaged, grouped as text,
// read as dates — which ShowTypes, printing the Go type of every cell, does
// not answer.
type DataType int

const (
	// DataTypeEmpty means the list holds no values: it is empty, or every
	// cell is nil or NaN.
	DataTypeEmpty DataType = iota
	// DataTypeNumber means every value is a number: any int, uint or float
	// width, a named type over one, or a decimal. Integers and floats are the
	// same data type; ShowTypes still shows which Go type each cell is.
	DataTypeNumber
	// DataTypeString means every value is text. Text that looks numeric, such
	// as "0050", is still text.
	DataTypeString
	// DataTypeBool means every value is true or false.
	DataTypeBool
	// DataTypeTime means every value is a time.Time.
	DataTypeTime
	// DataTypeOther means every value is of one kind none of the above covers,
	// such as []byte.
	DataTypeOther
	// DataTypeMixed means the values are of more than one of the kinds above.
	DataTypeMixed
)

// String returns the data type's name: "number", "string", "mixed", ...
func (t DataType) String() string {
	switch t {
	case DataTypeEmpty:
		return "empty"
	case DataTypeNumber:
		return "number"
	case DataTypeString:
		return "string"
	case DataTypeBool:
		return "bool"
	case DataTypeTime:
		return "time"
	case DataTypeOther:
		return "other"
	case DataTypeMixed:
		return "mixed"
	}
	return "unknown"
}

// DataType returns the kind of values the list holds, ignoring missing ones.
func (dl *DataList) DataType() DataType {
	var t DataType
	dl.AtomicDo(func(dl *DataList) {
		t = dataTypeOf(dl.data)
	})
	return t
}

// ColDataTypes returns each column's DataType, in column order, the way
// ColNames returns each column's name.
func (dt *DataTable) ColDataTypes() []DataType {
	var types []DataType
	dt.AtomicDo(func(dt *DataTable) {
		types = make([]DataType, len(dt.columns))
		for i, col := range dt.columns {
			types[i] = dataTypeOf(col.data)
		}
	})
	return types
}

func dataTypeOf(values []any) DataType {
	found := DataTypeEmpty
	for _, v := range values {
		if isMissing(v) {
			continue
		}
		t := valueDataType(v)
		switch found {
		case DataTypeEmpty:
			found = t
		case t:
		default:
			return DataTypeMixed
		}
	}
	return found
}

func valueDataType(v any) DataType {
	switch v.(type) {
	case string:
		return DataTypeString
	case bool:
		return DataTypeBool
	case time.Time:
		return DataTypeTime
	}
	if _, ok := ToFloat64Safe(v); ok {
		return DataTypeNumber
	}
	switch reflect.ValueOf(v).Kind() {
	case reflect.String:
		return DataTypeString
	case reflect.Bool:
		return DataTypeBool
	}
	return DataTypeOther
}
