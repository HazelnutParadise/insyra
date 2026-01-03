package py

import (
	"fmt"
	"reflect"

	"github.com/HazelnutParadise/Go-Utils/conv"
	"github.com/HazelnutParadise/insyra"
	json "github.com/goccy/go-json"
)

func bindPyResult(out any, result any) error {
	if out == nil {
		return nil
	}

	switch target := out.(type) {
	case **insyra.DataTable:
		dt, err := decodeDataTable(result)
		if err != nil {
			return err
		}
		*target = dt
		return nil
	case *insyra.DataTable:
		dt, err := decodeDataTable(result)
		if err != nil {
			return err
		}
		if dt != nil {
			*target = *dt
		}
		return nil
	case *insyra.IDataTable:
		dt, err := decodeDataTable(result)
		if err != nil {
			return err
		}
		// If decode returns nil, explicitly set the interface to an untyped nil
		// so callers see a true nil value (not a typed nil pointer in the interface).
		if dt == nil {
			*target = nil
		} else {
			*target = dt
		}
		return nil
	case **insyra.DataList:
		dl, err := decodeDataList(result)
		if err != nil {
			return err
		}
		*target = dl
		return nil
	case *insyra.DataList:
		dl, err := decodeDataList(result)
		if err != nil {
			return err
		}
		if dl != nil {
			*target = *dl
		}
		return nil
	case *insyra.IDataList:
		dl, err := decodeDataList(result)
		if err != nil {
			return err
		}
		// If decode returns nil, explicitly set the interface to an untyped nil
		// so callers see a true nil value (not a typed nil pointer in the interface).
		if dl == nil {
			*target = nil
		} else {
			*target = dl
		}
		return nil
	}

	// Try binding into isr wrapper types (dl / dt) by reflection so we don't need to
	// refer to their unexported type names. The wrappers embed an exported field
	// named "DataList" or "DataTable", which we can set via reflection.
	if func() bool {
		// Try DataList
		if dl, err := decodeDataList(result); err == nil {
			rv := reflect.ValueOf(out)
			for depth := 0; depth < 3 && rv.IsValid(); depth++ {
				if rv.Kind() != reflect.Ptr {
					break
				}
				// If we have a pointer-to-pointer and the inner pointer is nil, allocate it
				if rv.Elem().Kind() == reflect.Ptr && rv.Elem().IsNil() {
					rv.Elem().Set(reflect.New(rv.Elem().Type().Elem()))
				}
				if rv.Elem().Kind() == reflect.Struct {
					fld := rv.Elem().FieldByName("DataList")
					if fld.IsValid() && fld.CanSet() && fld.Type() == reflect.TypeOf((*insyra.DataList)(nil)) {
						if dl == nil {
							fld.Set(reflect.Zero(fld.Type()))
						} else {
							fld.Set(reflect.ValueOf(dl))
						}
						return true
					}
				}
				rv = rv.Elem()
			}
		}
		// Try DataTable
		if dt, err := decodeDataTable(result); err == nil {
			rv := reflect.ValueOf(out)
			for depth := 0; depth < 3 && rv.IsValid(); depth++ {
				if rv.Kind() != reflect.Ptr {
					break
				}
				if rv.Elem().Kind() == reflect.Ptr && rv.Elem().IsNil() {
					rv.Elem().Set(reflect.New(rv.Elem().Type().Elem()))
				}
				if rv.Elem().Kind() == reflect.Struct {
					fld := rv.Elem().FieldByName("DataTable")
					if fld.IsValid() && fld.CanSet() && fld.Type() == reflect.TypeOf((*insyra.DataTable)(nil)) {
						if dt == nil {
							fld.Set(reflect.Zero(fld.Type()))
						} else {
							fld.Set(reflect.ValueOf(dt))
						}
						return true
					}
				}
				rv = rv.Elem()
			}
		}
		return false
	}() {
		return nil
	}

	jsonData, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}
	if err := json.Unmarshal(jsonData, out); err != nil {
		return fmt.Errorf("failed to unmarshal result to struct: %w", err)
	}
	return nil
}

func decodeDataList(result any) (*insyra.DataList, error) {
	if result == nil {
		return nil, nil
	}

	if payload, ok := result.(map[string]any); ok {
		return dataListFromPayload(payload)
	}

	values, err := normalizeSliceAny(result)
	if err != nil {
		return nil, fmt.Errorf("unsupported datalist result: %T", result)
	}
	return newDataListFromSlice(values, ""), nil
}

func dataListFromPayload(payload map[string]any) (*insyra.DataList, error) {
	typeValue := ""
	if rawType, ok := payload[pyReturnTypeKey]; ok && rawType != nil {
		typeValue = conv.ToString(rawType)
	}

	if typeValue != "" &&
		typeValue != pyReturnTypeDataList &&
		typeValue != pyReturnTypeSeries {
		return nil, fmt.Errorf("unsupported datalist payload type: %s", typeValue)
	}

	dataRaw, ok := payload[pyReturnDataKey]
	if !ok {
		return nil, fmt.Errorf("missing datalist payload data")
	}
	values, err := normalizeSliceAny(dataRaw)
	if err != nil {
		return nil, err
	}

	name := ""
	if rawName, ok := payload[pyReturnNameKey]; ok && rawName != nil {
		name = conv.ToString(rawName)
	}

	return newDataListFromSlice(values, name), nil
}

func newDataListFromSlice(values []any, name string) *insyra.DataList {
	dl := insyra.NewDataList()
	if name != "" {
		dl.SetName(name)
	}
	if len(values) > 0 {
		dl.Append(values...)
	}
	return dl
}

func decodeDataTable(result any) (*insyra.DataTable, error) {
	if result == nil {
		return nil, nil
	}

	switch v := result.(type) {
	case map[string]any:
		return dataTableFromMap(v)
	case []any:
		return dataTableFromSlice(v)
	case [][]any:
		return dataTableFromRows(v, nil)
	case []map[string]any:
		return insyra.ReadJSON(v)
	default:
		return nil, fmt.Errorf("unsupported datatable result: %T", result)
	}
}

func dataTableFromMap(payload map[string]any) (*insyra.DataTable, error) {
	typeValue := ""
	if rawType, ok := payload[pyReturnTypeKey]; ok && rawType != nil {
		typeValue = conv.ToString(rawType)
	}

	if typeValue != "" &&
		typeValue != pyReturnTypeDataTable &&
		typeValue != pyReturnTypeDataFrame {
		return nil, fmt.Errorf("unsupported datatable payload type: %s", typeValue)
	}

	if typeValue != "" {
		return dataTableFromPayload(payload)
	}
	if _, ok := payload[pyReturnDataKey]; ok {
		return dataTableFromPayload(payload)
	}

	return insyra.ReadJSON(payload)
}

func dataTableFromSlice(values []any) (*insyra.DataTable, error) {
	if len(values) == 0 {
		return insyra.NewDataTable(), nil
	}

	if _, ok := values[0].(map[string]any); ok {
		records := make([]map[string]any, 0, len(values))
		for i, item := range values {
			row, ok := item.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("row %d is not an object, got %T", i, item)
			}
			records = append(records, row)
		}
		return insyra.ReadJSON(records)
	}

	rows, err := normalize2DSliceAny(values)
	if err != nil {
		return nil, err
	}
	return dataTableFromRows(rows, nil)
}

func dataTableFromPayload(payload map[string]any) (*insyra.DataTable, error) {
	dataRaw, ok := payload[pyReturnDataKey]
	if !ok {
		return nil, fmt.Errorf("missing datatable payload data")
	}

	rows, err := normalize2DSliceAny(dataRaw)
	if err != nil {
		return nil, err
	}

	var colNames []string
	if rawCols, ok := payload[pyReturnColumnsKey]; ok {
		colNames, err = normalizeStringSlice(rawCols)
		if err != nil {
			return nil, err
		}
	}

	var rowNames []string
	if rawIndex, ok := payload[pyReturnIndexKey]; ok {
		rowNames, err = normalizeStringSlice(rawIndex)
		if err != nil {
			return nil, err
		}
	}

	dt, err := dataTableFromRows(rows, colNames)
	if err != nil {
		return nil, err
	}

	if len(colNames) > 0 {
		dt.SetColNames(colNames)
	}
	if len(rowNames) > 0 {
		dt.SetRowNames(rowNames)
	}
	if rawName, ok := payload[pyReturnNameKey]; ok && rawName != nil {
		dt.SetName(conv.ToString(rawName))
	}

	return dt, nil
}

func dataTableFromRows(rows [][]any, colNames []string) (*insyra.DataTable, error) {
	if len(rows) == 0 {
		dt := insyra.NewDataTable()
		if len(colNames) == 0 {
			return dt, nil
		}

		emptyCols := make([]*insyra.DataList, len(colNames))
		for i, name := range colNames {
			dl := insyra.NewDataList()
			dl.SetName(name)
			emptyCols[i] = dl
		}
		dt.AppendCols(emptyCols...)
		return dt, nil
	}

	return insyra.Slice2DToDataTable(rows)
}

func normalizeSliceAny(raw any) ([]any, error) {
	if raw == nil {
		return nil, nil
	}

	switch v := raw.(type) {
	case []any:
		return v, nil
	case []string:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = item
		}
		return out, nil
	case []int:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = item
		}
		return out, nil
	case []int64:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = item
		}
		return out, nil
	case []int32:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = item
		}
		return out, nil
	case []int16:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = item
		}
		return out, nil
	case []int8:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = item
		}
		return out, nil
	case []uint:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = item
		}
		return out, nil
	case []uint64:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = item
		}
		return out, nil
	case []uint32:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = item
		}
		return out, nil
	case []uint16:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = item
		}
		return out, nil
	case []uint8:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = item
		}
		return out, nil
	case []float64:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = item
		}
		return out, nil
	case []float32:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = item
		}
		return out, nil
	case []bool:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = item
		}
		return out, nil
	default:
		return nil, fmt.Errorf("expected array, got %T", raw)
	}
}

func normalize2DSliceAny(raw any) ([][]any, error) {
	if raw == nil {
		return nil, nil
	}

	switch v := raw.(type) {
	case [][]any:
		return v, nil
	case []any:
		rows := make([][]any, len(v))
		for i, item := range v {
			row, err := normalizeSliceAny(item)
			if err != nil {
				return nil, fmt.Errorf("row %d: %w", i, err)
			}
			rows[i] = row
		}
		return rows, nil
	default:
		return nil, fmt.Errorf("expected 2D array, got %T", raw)
	}
}

func normalizeStringSlice(raw any) ([]string, error) {
	if raw == nil {
		return nil, nil
	}

	switch v := raw.(type) {
	case []string:
		return v, nil
	case []any:
		out := make([]string, len(v))
		for i, item := range v {
			if item == nil {
				out[i] = ""
				continue
			}
			out[i] = conv.ToString(item)
		}
		return out, nil
	default:
		return nil, fmt.Errorf("expected string array, got %T", raw)
	}
}
