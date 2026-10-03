package py

import (
	"cmp"
	"encoding"
	"errors"
	"fmt"
	"maps"
	"math"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"
	"unicode"

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
		// Binding into a value `*insyra.DataTable` would copy internal lock-like fields
		// (e.g. sync/atomic.Int64 -> sync/atomic.noCopy) which is unsafe and flagged by govet.
		// Require callers to pass a pointer-to-pointer (**insyra.DataTable) or an interface
		// (*insyra.IDataTable) instead so we don't copy lock-containing structs.
		return fmt.Errorf("binding to *insyra.DataTable (value target) is not supported; pass a **insyra.DataTable or *insyra.IDataTable instead")
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
		// Binding into a value `*insyra.DataList` would copy internal lock-like fields
		// (e.g. sync/atomic.Int64 -> sync/atomic.noCopy) which is unsafe and flagged by govet.
		// Require callers to pass a pointer-to-pointer (**insyra.DataList) or an interface
		// (*insyra.IDataList) instead so we don't copy lock-containing structs.
		return fmt.Errorf("binding to *insyra.DataList (value target) is not supported; pass a **insyra.DataList or *insyra.IDataList instead")
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

	// A table or a list in out's type, whether an isr wrapper's or one below
	// the top level, would come out of JSON empty, and an empty interface
	// would round a large integer, so such a type is decoded part by part. So
	// is a result holding a number go-json would wrap around in an integer.
	if rv := reflect.ValueOf(out); rv.Kind() == reflect.Pointer && !rv.IsNil() && (decodedPartByPart(rv.Elem().Type()) || holdsWideNumber(result)) {
		return decodeInto(rv.Elem(), result)
	}

	jsonData, err := json.Marshal(result)
	if err != nil {
		// JSON has no NaN or infinity, which Python sends for a float that is
		// one, so such a result is set part by part instead.
		if rv := reflect.ValueOf(out); rv.Kind() == reflect.Pointer && !rv.IsNil() && isNonFinite(err) {
			return decodeInto(rv.Elem(), result)
		}
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

	// The empty path has already named its columns; naming them again would see
	// each name as taken and add a suffix.
	if len(colNames) > 0 && len(rows) > 0 {
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

	return insyra.ReadSlice2D(rows)
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

var (
	dataTablePtrType    = reflect.TypeOf((*insyra.DataTable)(nil))
	dataListPtrType     = reflect.TypeOf((*insyra.DataList)(nil))
	iDataTableType      = reflect.TypeOf((*insyra.IDataTable)(nil)).Elem()
	iDataListType       = reflect.TypeOf((*insyra.IDataList)(nil)).Elem()
	jsonUnmarshalerType = reflect.TypeOf((*json.Unmarshaler)(nil)).Elem()
	textUnmarshalerType = reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()
)

// decodeInto runs assignResult, and returns a panic from reflection on a type
// nobody foresaw as an error, because the library never panics.
func decodeInto(dst reflect.Value, src any) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("cannot decode the result into %s: %v", dst.Type(), r)
		}
	}()
	return assignResult(dst, src)
}

// isTableOrList reports whether t is one of the types a DataFrame or a Series
// is decoded into.
func isTableOrList(t reflect.Type) bool {
	return t == dataTablePtrType || t == dataListPtrType || t == iDataTableType || t == iDataListType
}

// decodesItself reports whether encoding/json would hand a value of type t to
// its own UnmarshalJSON or UnmarshalText.
func decodesItself(t reflect.Type) bool {
	pt := reflect.PointerTo(t)
	return pt.Implements(jsonUnmarshalerType) || pt.Implements(textUnmarshalerType)
}

var holdsCache sync.Map // reflect.Type -> bool

// decodedPartByPart reports whether t is, or holds where encoding/json would
// decode it (a struct field, a map value, a slice or array element, or behind
// a pointer), a table, a list or an empty interface. JSON cannot decode a table
// or a list, and would give an empty interface a float64 for an integer a
// float64 rounds, so a type with any of them is decoded by assignResult.
func decodedPartByPart(t reflect.Type) bool {
	if holds, ok := holdsCache.Load(t); ok {
		return holds.(bool)
	}
	holds := decodedPartByPartIn(t, map[reflect.Type]bool{})
	holdsCache.Store(t, holds)
	return holds
}

func decodedPartByPartIn(t reflect.Type, seen map[reflect.Type]bool) bool {
	if isTableOrList(t) || (t.Kind() == reflect.Interface && t.NumMethod() == 0) {
		return true
	}
	if seen[t] || decodesItself(t) {
		return false
	}
	seen[t] = true
	switch t.Kind() {
	case reflect.Pointer, reflect.Slice, reflect.Array:
		return decodedPartByPartIn(t.Elem(), seen)
	case reflect.Map:
		return validMapKey(t.Key()) && decodedPartByPartIn(t.Elem(), seen)
	case reflect.Struct:
		fields := cachedFields(t)
		if len(fields.whole) > 0 {
			return true
		}
		for _, f := range fields.list {
			if decodedPartByPartIn(f.typ, seen) {
				return true
			}
		}
	}
	return false
}

// validMapKey reports whether encoding/json decodes an object into a map with
// keys of type kt: a type with UnmarshalText, or a string or integer kind.
func validMapKey(kt reflect.Type) bool {
	switch kt.Kind() {
	case reflect.String,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return true
	}
	return reflect.PointerTo(kt).Implements(textUnmarshalerType)
}

// assignResult decodes src, a value decoded from Python's JSON, into dst the
// way encoding/json would, except that a table or a list is decoded by the
// table and list decoders. dst must be addressable.
func assignResult(dst reflect.Value, src any) error {
	t := dst.Type()
	if isTableOrList(t) {
		return assignTableOrList(dst, src)
	}
	if !dst.CanSet() && t.Kind() != reflect.Struct {
		return fmt.Errorf("cannot set a value of type %s", t)
	}
	if t.Kind() >= reflect.Int && t.Kind() <= reflect.Uintptr && !decodesItself(t) {
		if handled, err := setInteger(dst, src); handled {
			return err
		}
	}
	// A struct reached through an embedded unexported field cannot be set as
	// a whole, but its exported fields can, so it is decoded field by field.
	// So is a value holding a NaN or an infinity, which JSON cannot carry,
	// unless its type decodes itself.
	if dst.CanSet() && !decodedPartByPart(t) && !holdsWideNumber(src) {
		err := assignJSON(dst, src)
		if err == nil || !isNonFinite(err) || decodesItself(t) {
			return err
		}
	}
	switch t.Kind() {
	case reflect.Pointer:
		if src == nil {
			dst.Set(reflect.Zero(t))
			return nil
		}
		if dst.IsNil() {
			dst.Set(reflect.New(t.Elem()))
		}
		return assignResult(dst.Elem(), src)
	case reflect.Slice:
		if src == nil {
			dst.Set(reflect.Zero(t))
			return nil
		}
		items, ok := src.([]any)
		if !ok {
			return fmt.Errorf("expected an array for %s, got %T", t, src)
		}
		// The elements already there are decoded into, as encoding/json does.
		out := reflect.MakeSlice(t, len(items), len(items))
		reflect.Copy(out, dst)
		for i, item := range items {
			if err := assignResult(out.Index(i), item); err != nil {
				return fmt.Errorf("element %d: %w", i, err)
			}
		}
		dst.Set(out)
		return nil
	case reflect.Array:
		if src == nil {
			return nil
		}
		items, ok := src.([]any)
		if !ok {
			return fmt.Errorf("expected an array for %s, got %T", t, src)
		}
		for i := 0; i < dst.Len(); i++ {
			if i >= len(items) {
				dst.Index(i).SetZero()
				continue
			}
			if err := assignResult(dst.Index(i), items[i]); err != nil {
				return fmt.Errorf("element %d: %w", i, err)
			}
		}
		return nil
	case reflect.Map:
		if src == nil {
			dst.Set(reflect.Zero(t))
			return nil
		}
		entries, ok := src.(map[string]any)
		if !ok {
			return fmt.Errorf("expected an object for %s, got %T", t, src)
		}
		if dst.IsNil() {
			dst.Set(reflect.MakeMapWithSize(t, len(entries)))
		}
		for _, k := range slices.Sorted(maps.Keys(entries)) {
			key, err := mapKey(t.Key(), k)
			if err != nil {
				return err
			}
			val := reflect.New(t.Elem()).Elem()
			if err := assignResult(val, entries[k]); err != nil {
				return fmt.Errorf("key %q: %w", k, err)
			}
			dst.SetMapIndex(key, val)
		}
		return nil
	case reflect.Struct:
		if src == nil {
			return nil
		}
		fields := cachedFields(t)
		entries, isObject := src.(map[string]any)
		// An untagged embedded table or list is decoded from the whole value,
		// the way encoding/json decodes an untagged embedded struct from the
		// object around it; the isr types are such a struct. Beside other
		// fields it takes only what Python sent as a table or a list, since an
		// ordinary object is the struct's own.
		if len(fields.whole) > 0 && (len(fields.list) == 0 || !isObject || isPayload(entries)) {
			for _, w := range fields.whole {
				field, err := fieldByIndex(dst, w.index)
				if err != nil {
					return err
				}
				if err := assignTableOrList(field, src); err != nil {
					return err
				}
			}
			return nil
		}
		if !isObject {
			return fmt.Errorf("expected an object for %s, got %T", t, src)
		}
		return assignStruct(dst, entries)
	case reflect.Float32, reflect.Float64:
		if f, ok := src.(float64); ok {
			dst.SetFloat(f)
			return nil
		}
	case reflect.Interface:
		if t.NumMethod() == 0 {
			// An interface holding a non-nil pointer is decoded into what the
			// pointer points to, as encoding/json does.
			if e := dst.Elem(); src != nil && e.Kind() == reflect.Pointer && !e.IsNil() {
				return assignResult(e.Elem(), src)
			}
			if src == nil {
				dst.SetZero()
			} else {
				dst.Set(reflect.ValueOf(src))
			}
			return nil
		}
	}
	if f, ok := src.(float64); ok && (math.IsNaN(f) || math.IsInf(f, 0)) {
		return fmt.Errorf("cannot decode %v into %s", f, t)
	}
	return assignJSON(dst, src)
}

// isPayload reports whether an object is a table or a list as insyra.Return
// sends one, marked with its type.
func isPayload(entries map[string]any) bool {
	_, ok := entries[pyReturnTypeKey]
	return ok
}

// assignTableOrList decodes src into dst, a table or a list, replacing what
// dst held.
func assignTableOrList(dst reflect.Value, src any) error {
	t := dst.Type()
	if !dst.CanSet() {
		return fmt.Errorf("cannot set a value of type %s", t)
	}
	var decoded reflect.Value
	if t == dataTablePtrType || t == iDataTableType {
		dt, err := decodeDataTable(src)
		if err != nil {
			return err
		}
		if dt != nil {
			decoded = reflect.ValueOf(dt)
		}
	} else {
		dl, err := decodeDataList(src)
		if err != nil {
			return err
		}
		if dl != nil {
			decoded = reflect.ValueOf(dl)
		}
	}
	if !decoded.IsValid() {
		dst.SetZero()
		return nil
	}
	dst.Set(decoded)
	return nil
}

// assignStruct decodes an object into the fields of a struct. Keys are taken
// in sorted order, the order JSON decoding saw them in, since Python's own
// order is gone by the time the result is a map; each finds its field by an
// exact name first and then without regard to case, as in encoding/json.
func assignStruct(dst reflect.Value, entries map[string]any) error {
	fields := cachedFields(dst.Type())
	for _, key := range slices.Sorted(maps.Keys(entries)) {
		i, ok := fields.exact[key]
		if !ok {
			i, ok = fields.folded[foldName(key)]
		}
		if !ok {
			continue
		}
		f := fields.list[i]
		field, err := fieldByIndex(dst, f.index)
		if err != nil {
			return err
		}
		if f.quoted {
			err = assignQuoted(field, entries[key])
		} else {
			err = assignResult(field, entries[key])
		}
		if err != nil {
			return fmt.Errorf("field %s: %w", f.name, err)
		}
	}
	return nil
}

// fieldByIndex returns the field of the struct v at index, allocating the
// embedded pointers on the way, as encoding/json does once a key reaches
// them. A nil embedded pointer to an unexported struct cannot be set, which
// is an error there too.
func fieldByIndex(v reflect.Value, index []int) (reflect.Value, error) {
	for _, i := range index {
		if v.Kind() == reflect.Pointer {
			if v.IsNil() {
				if !v.CanSet() {
					return reflect.Value{}, fmt.Errorf("cannot set embedded pointer to unexported struct: %v", v.Type().Elem())
				}
				v.Set(reflect.New(v.Type().Elem()))
			}
			v = v.Elem()
		}
		v = v.Field(i)
	}
	return v, nil
}

// assignQuoted decodes a field tagged ",string", whose value encoding/json
// requires to be a string holding the field's JSON.
func assignQuoted(dst reflect.Value, src any) error {
	switch s := src.(type) {
	case nil:
		return assignJSON(dst, nil)
	case string:
		// Decoding into a copy leaves the field as it was when the value is
		// wrong, where go-json would have written part of it. A pointer starts
		// nil, so a quoted null sets it to nil, as in encoding/json.
		tmp := reflect.New(dst.Type())
		if dst.Kind() != reflect.Pointer {
			tmp.Elem().Set(dst)
		}
		if err := json.Unmarshal([]byte(s), tmp.Interface()); err != nil {
			return fmt.Errorf("invalid use of ,string struct tag, trying to unmarshal %q into %v", s, dst.Type())
		}
		dst.Set(tmp.Elem())
		return nil
	default:
		return fmt.Errorf("invalid use of ,string struct tag, trying to unmarshal unquoted value into %v", dst.Type())
	}
}

// mapKey turns an object key into a key of type kt as encoding/json does: a
// type with UnmarshalText decodes it, a string kind takes it as it is, and an
// integer kind parses it.
func mapKey(kt reflect.Type, key string) (reflect.Value, error) {
	if reflect.PointerTo(kt).Implements(textUnmarshalerType) {
		kv := reflect.New(kt)
		if err := kv.Interface().(encoding.TextUnmarshaler).UnmarshalText([]byte(key)); err != nil {
			return reflect.Value{}, fmt.Errorf("key %q: %w", key, err)
		}
		return kv.Elem(), nil
	}
	kv := reflect.New(kt).Elem()
	switch kt.Kind() {
	case reflect.String:
		kv.SetString(key)
		return kv, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(key, 10, 64)
		if err != nil || kt.OverflowInt(n) {
			return reflect.Value{}, fmt.Errorf("cannot use the key %q as a %s", key, kt)
		}
		kv.SetInt(n)
		return kv, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		n, err := strconv.ParseUint(key, 10, 64)
		if err != nil || kt.OverflowUint(n) {
			return reflect.Value{}, fmt.Errorf("cannot use the key %q as a %s", key, kt)
		}
		kv.SetUint(n)
		return kv, nil
	}
	return reflect.Value{}, fmt.Errorf("cannot use a key of type %s", kt)
}

// assignJSON decodes src into dst through JSON.
func assignJSON(dst reflect.Value, src any) error {
	data, err := json.Marshal(src)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}
	if err := json.Unmarshal(data, dst.Addr().Interface()); err != nil {
		return fmt.Errorf("failed to unmarshal result: %w", err)
	}
	return nil
}

// isNonFinite reports whether err is JSON refusing a NaN or an infinity.
func isNonFinite(err error) bool {
	var unsupported *json.UnsupportedValueError
	return errors.As(err, &unsupported)
}

// jsonField is a struct field encoding/json decodes.
type jsonField struct {
	name   string
	tagged bool
	index  []int
	typ    reflect.Type
	quoted bool
}

// jsonFields lists a struct type's fields in index order, with each field's
// position by its exact name and by its name folded for case, and the
// untagged embedded tables and lists, which take the whole value.
type jsonFields struct {
	list   []jsonField
	exact  map[string]int
	folded map[string]int
	whole  []jsonField
}

var fieldCache sync.Map // reflect.Type -> *jsonFields

func cachedFields(t reflect.Type) *jsonFields {
	if fs, ok := fieldCache.Load(t); ok {
		return fs.(*jsonFields)
	}
	fs, _ := fieldCache.LoadOrStore(t, structFields(t))
	return fs.(*jsonFields)
}

// structFields lists the fields of the struct type t that encoding/json
// decodes, by its rules: an embedded struct's fields are promoted, a field
// hides one of the same name deeper down, two of one name at one depth hide
// each other unless only one is tagged, and a field tagged "-" is skipped. It
// follows typeFields in encoding/json's encode.go, except that an untagged
// embedded *insyra.DataTable or *insyra.DataList is not a struct to look
// into: it takes the whole value. The shallowest ones hide any deeper down,
// and two of one type at that depth hide each other. An embedded IDataTable
// or IDataList is a field named after its type, as an embedded interface is
// in encoding/json.
func structFields(t reflect.Type) *jsonFields {
	current := []jsonField{}
	next := []jsonField{{typ: t}}
	var count, nextCount map[reflect.Type]int
	visited := map[reflect.Type]bool{}
	var fields []jsonField
	var whole []jsonField

	for len(next) > 0 {
		current, next = next, current[:0]
		count, nextCount = nextCount, map[reflect.Type]int{}

		for _, f := range current {
			if visited[f.typ] {
				continue
			}
			visited[f.typ] = true

			for i := 0; i < f.typ.NumField(); i++ {
				sf := f.typ.Field(i)
				if sf.Anonymous {
					et := sf.Type
					if et.Kind() == reflect.Pointer {
						et = et.Elem()
					}
					if !sf.IsExported() && et.Kind() != reflect.Struct {
						continue
					}
				} else if !sf.IsExported() {
					continue
				}
				tag := sf.Tag.Get("json")
				if tag == "-" {
					continue
				}
				name, opts, _ := strings.Cut(tag, ",")
				if !isValidTag(name) {
					name = ""
				}
				index := make([]int, len(f.index)+1)
				copy(index, f.index)
				index[len(f.index)] = i

				ft := sf.Type
				if ft.Name() == "" && ft.Kind() == reflect.Pointer {
					ft = ft.Elem()
				}
				quoted := false
				if hasTagOption(opts, "string") {
					switch ft.Kind() {
					case reflect.Bool,
						reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
						reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
						reflect.Float32, reflect.Float64,
						reflect.String:
						quoted = true
					}
				}

				if sf.Anonymous && name == "" && (sf.Type == dataTablePtrType || sf.Type == dataListPtrType) {
					if len(whole) == 0 || len(whole[0].index) == len(index) {
						whole = append(whole, jsonField{index: index, typ: sf.Type})
						if count[f.typ] > 1 {
							whole = append(whole, whole[len(whole)-1])
						}
					}
					continue
				}
				if name != "" || !sf.Anonymous || ft.Kind() != reflect.Struct {
					tagged := name != ""
					if name == "" {
						name = sf.Name
					}
					fields = append(fields, jsonField{name: name, tagged: tagged, index: index, typ: sf.Type, quoted: quoted})
					if count[f.typ] > 1 {
						// A second copy makes the pass below see the name
						// twice, so both instances hide each other.
						fields = append(fields, fields[len(fields)-1])
					}
					continue
				}

				nextCount[ft]++
				if nextCount[ft] == 1 {
					next = append(next, jsonField{name: ft.Name(), index: index, typ: ft})
				}
			}
		}
	}

	slices.SortFunc(fields, func(a, b jsonField) int {
		if c := strings.Compare(a.name, b.name); c != 0 {
			return c
		}
		if c := cmp.Compare(len(a.index), len(b.index)); c != 0 {
			return c
		}
		if a.tagged != b.tagged {
			if a.tagged {
				return -1
			}
			return 1
		}
		return slices.Compare(a.index, b.index)
	})

	// For each name keep the dominant field: the shallowest, and at equal
	// depth the only tagged one. Two at the same depth that are both tagged,
	// or both untagged, hide each other.
	out := fields[:0]
	for advance, i := 0, 0; i < len(fields); i += advance {
		fi := fields[i]
		for advance = 1; i+advance < len(fields); advance++ {
			if fields[i+advance].name != fi.name {
				break
			}
		}
		if advance > 1 && len(fi.index) == len(fields[i+1].index) && fi.tagged == fields[i+1].tagged {
			continue
		}
		out = append(out, fi)
	}
	slices.SortFunc(out, func(a, b jsonField) int {
		return slices.Compare(a.index, b.index)
	})

	var kept []jsonField
	for _, w := range whole {
		n := 0
		for _, o := range whole {
			if o.typ == w.typ {
				n++
			}
		}
		if n == 1 {
			kept = append(kept, w)
		}
	}

	fs := &jsonFields{list: out, exact: make(map[string]int, len(out)), folded: make(map[string]int, len(out)), whole: kept}
	for i, f := range out {
		fs.exact[f.name] = i
		// The first field to fold to a name takes it, as in encoding/json.
		if _, ok := fs.folded[foldName(f.name)]; !ok {
			fs.folded[foldName(f.name)] = i
		}
	}
	return fs
}

// isValidTag reports whether s can name a field, as in encoding/json.
func isValidTag(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		switch {
		case strings.ContainsRune("!#$%&()*+-./:;<=>?@[]^_{|}~ ", c):
			// Backslash and quote characters are reserved; other
			// punctuation may appear in a name.
		case !unicode.IsLetter(c) && !unicode.IsDigit(c):
			return false
		}
	}
	return true
}

// hasTagOption reports whether the comma-separated options of a json tag
// include option.
func hasTagOption(opts, option string) bool {
	for opts != "" {
		var opt string
		opt, opts, _ = strings.Cut(opts, ",")
		if opt == option {
			return true
		}
	}
	return false
}

// foldName returns a form of name that two names share exactly when
// strings.EqualFold holds for them, as encoding/json's foldName does.
func foldName(name string) string {
	return strings.Map(func(r rune) rune { return unicode.ToUpper(unicode.ToLower(r)) }, name)
}

// holdsWideNumber reports whether v holds a number go-json would wrap around
// when decoding it into an integer, since it checks neither a nineteen- nor a
// twenty-digit number: a uint64, or a float64 of 2^63 or more in magnitude.
func holdsWideNumber(v any) bool {
	switch x := v.(type) {
	case uint64:
		return true
	case float64:
		return x >= 1<<63 || x <= -(1<<63)
	case []any:
		for _, e := range x {
			if holdsWideNumber(e) {
				return true
			}
		}
	case map[string]any:
		for _, e := range x {
			if holdsWideNumber(e) {
				return true
			}
		}
	}
	return false
}

// setInteger sets the integer dst from a decoded number, with the range check
// go-json leaves out. It reports false for a value that is not a number, which
// JSON then decodes. A number dst cannot hold, or one with a fraction, is an
// error.
func setInteger(dst reflect.Value, src any) (bool, error) {
	t := dst.Type()
	signed := t.Kind() <= reflect.Int64
	switch x := src.(type) {
	case int64:
		if signed && !dst.OverflowInt(x) {
			dst.SetInt(x)
			return true, nil
		}
		if !signed && x >= 0 && !dst.OverflowUint(uint64(x)) {
			dst.SetUint(uint64(x))
			return true, nil
		}
	case uint64:
		if signed && x <= math.MaxInt64 && !dst.OverflowInt(int64(x)) {
			dst.SetInt(int64(x))
			return true, nil
		}
		if !signed && !dst.OverflowUint(x) {
			dst.SetUint(x)
			return true, nil
		}
	case float64:
		if x != math.Trunc(x) {
			return true, fmt.Errorf("cannot decode %v into %s", x, t)
		}
		if signed && x >= -(1<<63) && x < 1<<63 && !dst.OverflowInt(int64(x)) {
			dst.SetInt(int64(x))
			return true, nil
		}
		if !signed && x >= 0 && x < 1<<64 && !dst.OverflowUint(uint64(x)) {
			dst.SetUint(uint64(x))
			return true, nil
		}
	default:
		return false, nil
	}
	return true, fmt.Errorf("cannot decode %v into %s", src, t)
}
