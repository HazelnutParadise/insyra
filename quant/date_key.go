package quant

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"time"
)

// dateKey groups dates by value, shared by QuantileSort and FamaMacBeth so both
// put two rows on one date under the same rule. A date is a time.Time (its
// instant, location ignored) or a value whose underlying kind is string (kept
// exactly, whitespace included), bool, an integer or a float (not NaN). The key
// carries the concrete type, so int 1, float64 1 and a defined type over int 1
// are different dates. Anything else (slices, maps, structs, pointers,
// interfaces, nil) has no key that two different values cannot share and is an
// error.
func dateKey(d any) (string, error) {
	if t, ok := d.(time.Time); ok {
		return fmt.Sprintf("t:%d.%09d", t.Unix(), t.Nanosecond()), nil
	}
	v := reflect.ValueOf(d)
	if !v.IsValid() {
		return "", errors.New("a date of type <nil> is not a supported date type")
	}
	name := v.Type().String()
	switch v.Kind() {
	case reflect.String:
		return name + ":" + v.String(), nil
	case reflect.Bool:
		return name + ":" + strconv.FormatBool(v.Bool()), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return name + ":" + strconv.FormatInt(v.Int(), 10), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return name + ":" + strconv.FormatUint(v.Uint(), 10), nil
	case reflect.Float32, reflect.Float64:
		return dateFloatKey(name, v.Float())
	}
	return "", fmt.Errorf("a date of type %T is not a supported date type; use a string, time.Time, bool, integer or float, or a defined type whose underlying type is one of these", d)
}

func dateFloatKey(name string, v float64) (string, error) {
	if math.IsNaN(v) {
		return "", errors.New("a date is NaN")
	}
	if v == 0 {
		v = 0 // -0 and 0 are the same date
	}
	return name + ":" + strconv.FormatFloat(v, 'g', -1, 64), nil
}
