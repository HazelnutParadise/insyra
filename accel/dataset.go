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
		Lineage: "project:datalist",
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
		Lineage: "project:datatable",
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
	// The bitmap starts all-valid and gets cleared as nulls turn up, so the work
	// is proportional to the number of nulls rather than to the column length.
	// Building it here also saves a second full pass over the values.
	validity := newValidityBitmap(len(values))

	switch dtype {
	case DataTypeBool:
		out := make([]bool, len(values))
		for i, value := range values {
			if value == nil {
				nulls[i] = true
				clearValidityBit(validity, i)
				continue
			}
			out[i] = value.(bool)
		}
		return Buffer{Name: name, Type: dtype, Values: out, Nulls: nulls, Validity: maskValidityTail(validity, len(values)), Len: len(values)}, nil
	case DataTypeInt64:
		out := make([]int64, len(values))
		for i, value := range values {
			if value == nil {
				nulls[i] = true
				clearValidityBit(validity, i)
				continue
			}
			converted, ok := toInt64(value)
			if !ok {
				return Buffer{}, fmt.Errorf("accel: value at index %d is not convertible to int64", i)
			}
			out[i] = converted
		}
		return Buffer{Name: name, Type: dtype, Values: out, Nulls: nulls, Validity: maskValidityTail(validity, len(values)), Len: len(values)}, nil
	case DataTypeFloat64:
		out := make([]float64, len(values))
		for i, value := range values {
			if value == nil {
				nulls[i] = true
				clearValidityBit(validity, i)
				continue
			}
			converted, ok := toFloat64(value)
			if !ok {
				return Buffer{}, fmt.Errorf("accel: value at index %d is not convertible to float64", i)
			}
			out[i] = converted
		}
		return Buffer{Name: name, Type: dtype, Values: out, Nulls: nulls, Validity: maskValidityTail(validity, len(values)), Len: len(values)}, nil
	case DataTypeString:
		out := make([]string, len(values))
		offsets := make([]uint32, 0, len(values)+1)
		data := make([]byte, 0, len(values)*4)
		offsets = append(offsets, 0)
		for i, value := range values {
			if value == nil {
				nulls[i] = true
				clearValidityBit(validity, i)
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
			Validity:      maskValidityTail(validity, len(values)),
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
				clearValidityBit(validity, i)
			}
		}
		return Buffer{Name: name, Type: DataTypeAny, Values: out, Nulls: nulls, Validity: maskValidityTail(validity, len(values)), Len: len(values)}, nil
	}
}

// newValidityBitmap returns a bitmap with every value marked valid. A set bit
// means "not null", so starting from all-ones lets the projection loop touch
// the bitmap only where a null actually appears.
func newValidityBitmap(n int) []byte {
	if n == 0 {
		return nil
	}
	bitmap := make([]byte, (n+7)/8)
	for i := range bitmap {
		bitmap[i] = 0xFF
	}
	return bitmap
}

func clearValidityBit(bitmap []byte, idx int) {
	bitmap[idx>>3] &^= 1 << (idx & 7)
}

// maskValidityTail clears the padding bits above n in the last byte, so the
// bitmap is byte-identical to one built by setting bits only for valid indices.
func maskValidityTail(bitmap []byte, n int) []byte {
	if len(bitmap) == 0 {
		return bitmap
	}
	if rem := n % 8; rem != 0 {
		bitmap[len(bitmap)-1] &= byte(1<<rem) - 1
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
		// One type switch answers what four predicate calls used to, and two of
		// those built a reflect.Type per element. Named types such as
		// `type Celsius float64` do not match a concrete case, so they fall
		// through to the reflect-based predicates that handled them before.
		switch value.(type) {
		case bool:
			seenBool = true
		case string:
			seenString = true
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			seenInt = true
		case float32, float64:
			seenFloat = true
		default:
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

// The type-dispatch helpers below run once per value, so they take the concrete
// types first and only reach for reflect when a value is a named type such as
// `type Celsius float64`. reflect.Value.Convert in particular allocated on the
// heap for every element, which cost a 4 Mi column four million allocations to
// produce values it already held.

func isInt(v any) bool {
	switch v.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return true
	}
	rt := reflect.TypeOf(v)
	if rt == nil {
		return false
	}
	switch rt.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return true
	default:
		return false
	}
}

func isFloat(v any) bool {
	switch v.(type) {
	case float32, float64:
		return true
	}
	rt := reflect.TypeOf(v)
	if rt == nil {
		return false
	}
	switch rt.Kind() {
	case reflect.Float32, reflect.Float64:
		return true
	default:
		return false
	}
}

func toInt64(v any) (int64, bool) {
	switch x := v.(type) {
	case int:
		return int64(x), true
	case int8:
		return int64(x), true
	case int16:
		return int64(x), true
	case int32:
		return int64(x), true
	case int64:
		return x, true
	case uint:
		return int64(x), true
	case uint8:
		return int64(x), true
	case uint16:
		return int64(x), true
	case uint32:
		return int64(x), true
	case uint64:
		// Values above MaxInt64 wrap, exactly as int64(rv.Uint()) did.
		return int64(x), true
	}
	return reflectToInt64(v)
}

func reflectToInt64(v any) (int64, bool) {
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
	switch x := v.(type) {
	case float64:
		return x, true
	case float32:
		return float64(x), true
	case int:
		return float64(x), true
	case int8:
		return float64(x), true
	case int16:
		return float64(x), true
	case int32:
		return float64(x), true
	case int64:
		return float64(x), true
	case uint:
		return float64(x), true
	case uint8:
		return float64(x), true
	case uint16:
		return float64(x), true
	case uint32:
		return float64(x), true
	case uint64:
		return float64(x), true
	}
	return reflectToFloat64(v)
}

func reflectToFloat64(v any) (float64, bool) {
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Float32, reflect.Float64:
		return rv.Float(), true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(rv.Int()), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(rv.Uint()), true
	default:
		return 0, false
	}
}
