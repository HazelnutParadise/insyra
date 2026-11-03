package arrowipc

import (
	"fmt"
	"strings"

	"github.com/TimLai666/go-decimal/decimal"
	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
)

// Keep these mappings aligned with parquet/internal.go's supportedArrowType
// and getVal. Unsupported fields fail at schema selection, even if all null.
func supportedCellType(dt arrow.DataType) bool {
	switch dt.ID() {
	case arrow.INT8, arrow.INT16, arrow.INT32, arrow.INT64,
		arrow.UINT8, arrow.UINT16, arrow.UINT32, arrow.UINT64,
		arrow.FLOAT32, arrow.FLOAT64, arrow.BOOL, arrow.STRING, arrow.LARGE_STRING,
		arrow.BINARY, arrow.LARGE_BINARY, arrow.FIXED_SIZE_BINARY,
		arrow.TIMESTAMP, arrow.DATE32, arrow.DATE64, arrow.DURATION,
		arrow.NULL:
		return true
	case arrow.DECIMAL128:
		t := dt.(*arrow.Decimal128Type)
		return t.Precision >= 1 && t.Precision <= 38 && t.Scale >= 0 && t.Scale <= t.Precision
	case arrow.DECIMAL256:
		t := dt.(*arrow.Decimal256Type)
		return t.Precision >= 1 && t.Precision <= 76 && t.Scale >= 0 && t.Scale <= t.Precision
	case arrow.DICTIONARY:
		return supportedCellType(dt.(*arrow.DictionaryType).ValueType)
	default:
		return false
	}
}

func cellValue(arr arrow.Array, i int) (any, error) {
	if arr.IsNull(i) {
		return nil, nil
	}
	switch a := arr.(type) {
	case *array.Int8:
		return a.Value(i), nil
	case *array.Int16:
		return a.Value(i), nil
	case *array.Int32:
		return a.Value(i), nil
	case *array.Int64:
		return a.Value(i), nil
	case *array.Uint8:
		return a.Value(i), nil
	case *array.Uint16:
		return a.Value(i), nil
	case *array.Uint32:
		return a.Value(i), nil
	case *array.Uint64:
		return a.Value(i), nil
	case *array.Float32:
		return a.Value(i), nil
	case *array.Float64:
		return a.Value(i), nil
	case *array.Boolean:
		return a.Value(i), nil
	case *array.String:
		return strings.Clone(a.Value(i)), nil
	case *array.LargeString:
		return strings.Clone(a.Value(i)), nil
	case *array.Binary:
		return append([]byte(nil), a.Value(i)...), nil
	case *array.LargeBinary:
		return append([]byte(nil), a.Value(i)...), nil
	case *array.FixedSizeBinary:
		return append([]byte(nil), a.Value(i)...), nil
	case *array.Timestamp:
		return a.Value(i).ToTime(a.DataType().(*arrow.TimestampType).Unit), nil
	case *array.Date32:
		return a.Value(i).ToTime().UTC(), nil
	case *array.Date64:
		return a.Value(i).ToTime().UTC(), nil
	case *array.Duration:
		return int64(a.Value(i)), nil
	case *array.Decimal128:
		return decimal.NewFromScaledInt(a.Value(i).BigInt(), a.DataType().(*arrow.Decimal128Type).Scale), nil
	case *array.Decimal256:
		return decimal.NewFromScaledInt(a.Value(i).BigInt(), a.DataType().(*arrow.Decimal256Type).Scale), nil
	case *array.Null:
		return nil, nil
	case *array.Dictionary:
		index := a.GetValueIndex(i)
		if index < 0 || index >= a.Dictionary().Len() {
			return nil, fmt.Errorf("arrowipc: dictionary index %d out of range", index)
		}
		return cellValue(a.Dictionary(), index)
	default:
		return nil, fmt.Errorf("arrowipc: unsupported Arrow type %v", arr.DataType())
	}
}

func checkExtension(field arrow.Field) error {
	if name, ok := field.Metadata.GetValue("ARROW:extension:name"); ok {
		return fmt.Errorf("arrowipc: unsupported selected column %q extension type %q (storage %s)", field.Name, name, field.Type)
	}
	return nil
}
