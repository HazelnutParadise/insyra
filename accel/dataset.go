package accel

import (
	"fmt"
	"reflect"

	"github.com/HazelnutParadise/insyra"
)

func (s *Session) ProjectDataList(dl *insyra.DataList) (*Dataset, error) {
	if dl == nil {
		return nil, fmt.Errorf("accel: nil datalist")
	}

	buf, err := projectValues(dl.GetName(), dl.Data())
	if err != nil {
		return nil, err
	}

	ds := &Dataset{
		Name:    dl.GetName(),
		Rows:    buf.Len,
		Buffers: []Buffer{buf},
	}
	assignDatasetFingerprint(ds)
	s.cacheDataset(ds)
	return ds, nil
}

func (s *Session) ProjectDataTable(dt *insyra.DataTable) (*Dataset, error) {
	if dt == nil {
		return nil, fmt.Errorf("accel: nil datatable")
	}

	cols := make([]Buffer, 0, dt.NumCols())
	for i := 0; i < dt.NumCols(); i++ {
		col := dt.GetColByNumber(i)
		buf, err := projectValues(col.GetName(), col.Data())
		if err != nil {
			return nil, err
		}
		cols = append(cols, buf)
	}

	ds := &Dataset{
		Name:    dt.GetName(),
		Rows:    dt.NumRows(),
		Buffers: cols,
	}
	assignDatasetFingerprint(ds)
	s.cacheDataset(ds)
	return ds, nil
}

func projectValues(name string, values []any) (Buffer, error) {
	dtype := inferDataType(values)
	nulls := make([]bool, len(values))

	switch dtype {
	case DataTypeBool:
		out := make([]bool, len(values))
		for i, value := range values {
			if value == nil {
				nulls[i] = true
				continue
			}
			out[i] = value.(bool)
		}
		return Buffer{Name: name, Type: dtype, Values: out, Nulls: nulls, Validity: buildValidityBitmap(nulls), Len: len(values)}, nil
	case DataTypeInt64:
		out := make([]int64, len(values))
		for i, value := range values {
			if value == nil {
				nulls[i] = true
				continue
			}
			converted, ok := toInt64(value)
			if !ok {
				return Buffer{}, fmt.Errorf("accel: value at index %d is not convertible to int64", i)
			}
			out[i] = converted
		}
		return Buffer{Name: name, Type: dtype, Values: out, Nulls: nulls, Validity: buildValidityBitmap(nulls), Len: len(values)}, nil
	case DataTypeFloat64:
		out := make([]float64, len(values))
		for i, value := range values {
			if value == nil {
				nulls[i] = true
				continue
			}
			converted, ok := toFloat64(value)
			if !ok {
				return Buffer{}, fmt.Errorf("accel: value at index %d is not convertible to float64", i)
			}
			out[i] = converted
		}
		return Buffer{Name: name, Type: dtype, Values: out, Nulls: nulls, Validity: buildValidityBitmap(nulls), Len: len(values)}, nil
	case DataTypeString:
		out := make([]string, len(values))
		offsets := make([]uint32, 0, len(values)+1)
		data := make([]byte, 0, len(values)*4)
		offsets = append(offsets, 0)
		for i, value := range values {
			if value == nil {
				nulls[i] = true
				offsets = append(offsets, uint32(len(data)))
				continue
			}
			text := value.(string)
			out[i] = text
			data = append(data, []byte(text)...)
			offsets = append(offsets, uint32(len(data)))
		}
		return Buffer{
			Name:          name,
			Type:          dtype,
			Values:        out,
			Nulls:         nulls,
			Validity:      buildValidityBitmap(nulls),
			StringOffsets: offsets,
			StringData:    data,
			Len:           len(values),
		}, nil
	default:
		out := make([]any, len(values))
		copy(out, values)
		for i, value := range values {
			if value == nil {
				nulls[i] = true
			}
		}
		return Buffer{Name: name, Type: DataTypeAny, Values: out, Nulls: nulls, Validity: buildValidityBitmap(nulls), Len: len(values)}, nil
	}
}

func buildValidityBitmap(nulls []bool) []byte {
	if len(nulls) == 0 {
		return nil
	}
	bitmap := make([]byte, (len(nulls)+7)/8)
	for idx, isNull := range nulls {
		if isNull {
			continue
		}
		byteIdx := idx / 8
		bitIdx := idx % 8
		bitmap[byteIdx] |= 1 << bitIdx
	}
	return bitmap
}

func inferDataType(values []any) DataType {
	seenString := false
	seenBool := false
	seenFloat := false
	seenInt := false
	seenOther := false

	for _, value := range values {
		if value == nil {
			continue
		}
		switch {
		case isBool(value):
			seenBool = true
		case isInt(value):
			seenInt = true
		case isFloat(value):
			seenFloat = true
		case isString(value):
			seenString = true
		default:
			seenOther = true
		}
	}

	switch {
	case seenOther:
		return DataTypeAny
	case seenString && !seenBool && !seenInt && !seenFloat:
		return DataTypeString
	case seenBool && !seenString && !seenInt && !seenFloat:
		return DataTypeBool
	case seenFloat && !seenString && !seenBool:
		return DataTypeFloat64
	case seenInt && !seenString && !seenBool && !seenFloat:
		return DataTypeInt64
	case (seenInt || seenFloat) && !seenString && !seenBool:
		return DataTypeFloat64
	default:
		return DataTypeAny
	}
}

func isBool(v any) bool {
	_, ok := v.(bool)
	return ok
}

func isString(v any) bool {
	_, ok := v.(string)
	return ok
}

func isInt(v any) bool {
	switch reflect.TypeOf(v).Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return true
	default:
		return false
	}
}

func isFloat(v any) bool {
	switch reflect.TypeOf(v).Kind() {
	case reflect.Float32, reflect.Float64:
		return true
	default:
		return false
	}
}

func toInt64(v any) (int64, bool) {
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return rv.Int(), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return int64(rv.Uint()), true
	default:
		return 0, false
	}
}

func toFloat64(v any) (float64, bool) {
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Float32, reflect.Float64:
		return rv.Convert(reflect.TypeOf(float64(0))).Float(), true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(rv.Int()), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(rv.Uint()), true
	default:
		return 0, false
	}
}
