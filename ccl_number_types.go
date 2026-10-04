package insyra

// unifyCCLNumbers gives a column CCL computed one kind of number, the way a
// pandas column has one dtype, so whether a literal was written 0 or 0.0 does
// not show in the result: COALESCE(TONUM(A), 0) over [19.5, "abc", 30] holds
// three float64s, not two and an int64. When the column's numbers include a
// float, every integer a float64 holds exactly becomes a float64; an integer
// past 2^53 keeps its digits, because a float64 would change it. A column whose
// numbers are all integers is left as it is, and values that are not numbers
// (text, booleans, nil, dates, durations, decimals) are never touched. col is
// changed in place.
func unifyCCLNumbers(col []any) {
	hasFloat := false
	for _, v := range col {
		switch v.(type) {
		case float64, float32:
			hasFloat = true
		}
		if hasFloat {
			break
		}
	}
	if !hasFloat {
		return
	}
	for i, v := range col {
		if f, ok := exactFloatOfInt(v); ok {
			col[i] = f
		}
	}
}

// float64ExactIntegers is the largest magnitude up to which a float64 holds
// every integer.
const float64ExactIntegers = 1 << 53

// exactFloatOfInt returns v as a float64 when v is a Go integer a float64
// holds exactly.
func exactFloatOfInt(v any) (float64, bool) {
	var i int64
	switch x := v.(type) {
	case int:
		i = int64(x)
	case int8:
		i = int64(x)
	case int16:
		i = int64(x)
	case int32:
		i = int64(x)
	case int64:
		i = x
	case uint8:
		i = int64(x)
	case uint16:
		i = int64(x)
	case uint32:
		i = int64(x)
	case uint:
		if uint64(x) > float64ExactIntegers {
			return 0, false
		}
		i = int64(x)
	case uint64:
		if x > float64ExactIntegers {
			return 0, false
		}
		i = int64(x)
	default:
		return 0, false
	}
	if i > float64ExactIntegers || i < -float64ExactIntegers {
		return 0, false
	}
	return float64(i), true
}
