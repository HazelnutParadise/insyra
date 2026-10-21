package quant

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"
)

// Shared by QuantileSort and FamaMacBeth, so both put two rows on one date
// under the same rule.
// dateKey groups dates by value. A date is a string (kept exactly,
// surrounding whitespace included), a time.Time (its instant, location
// ignored), a bool, an integer or a float (not NaN), keyed by its Go type and
// exact value, so int 1 and float64 1 are different dates. Any other type has
// no key that cannot be shared by two values and is an error.
func dateKey(d any) (string, error) {
	switch v := d.(type) {
	case string:
		return "s:" + v, nil
	case time.Time:
		return fmt.Sprintf("t:%d.%09d", v.Unix(), v.Nanosecond()), nil
	case bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%T:%d", d, d), nil
	case float32:
		return dateFloatKey("float32", float64(v))
	case float64:
		return dateFloatKey("float64", v)
	}
	return "", fmt.Errorf("a date of type %T is not a supported date type; use a string, time.Time, bool, integer or float", d)
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
