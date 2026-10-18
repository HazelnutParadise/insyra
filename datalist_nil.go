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
	return embedsNilDataList(reflect.ValueOf(dl), 0)
}

func embedsNilDataList(v reflect.Value, depth int) bool {
	if depth > 8 {
		return false
	}
	switch v.Kind() {
	case reflect.Pointer:
		if v.IsNil() {
			return true
		}
		if v.Type() == reflect.TypeFor[*DataList]() {
			return false
		}
		return embedsNilDataList(v.Elem(), depth+1)
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if !v.Type().Field(i).Anonymous {
				continue
			}
			f := v.Field(i)
			if f.Kind() == reflect.Pointer && f.Type() == reflect.TypeFor[*DataList]() && f.IsNil() {
				return true
			}
			if embedsNilDataList(f, depth+1) {
				return true
			}
		}
	}
	return false
}
