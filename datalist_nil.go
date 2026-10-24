package insyra

import "reflect"

// IsNilList reports whether dl has no DataList behind it: a nil interface, a
// nil *DataList, or a value that embeds a nil *DataList such as isr.DL, which
// satisfies IDataList but panics on its first method call. Code that accepts an
// IDataList uses it to refuse such a value before calling a method on it.
func IsNilList(dl IDataList) bool {
	if dl == nil {
		return true
	}
	return embedsNil(reflect.ValueOf(dl), reflect.TypeFor[*DataList](), 0)
}

// IsNilTable reports whether dt has no DataTable behind it, the way IsNilList
// does for a list: a nil interface, a nil *DataTable, or a value that embeds a
// nil *DataTable.
func IsNilTable(dt IDataTable) bool {
	if dt == nil {
		return true
	}
	return embedsNil(reflect.ValueOf(dt), reflect.TypeFor[*DataTable](), 0)
}

// embedsNil reports whether v is, or embeds, a nil pointer of type target.
func embedsNil(v reflect.Value, target reflect.Type, depth int) bool {
	if depth > 8 {
		return false
	}
	switch v.Kind() {
	case reflect.Pointer:
		if v.IsNil() {
			return true
		}
		if v.Type() == target {
			return false
		}
		return embedsNil(v.Elem(), target, depth+1)
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if !v.Type().Field(i).Anonymous {
				continue
			}
			f := v.Field(i)
			if f.Kind() == reflect.Pointer && f.Type() == target && f.IsNil() {
				return true
			}
			if embedsNil(f, target, depth+1) {
				return true
			}
		}
	}
	return false
}
