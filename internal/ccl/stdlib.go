package ccl

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/HazelnutParadise/insyra/internal/utils"
)

// RegisterStandardFunctions registers the standard library of CCL functions.
// Categories:
//   - Logical: IF, AND, OR, CASE
//   - Null/NaN: ISNA, IFNA
//   - String concat / duration helpers (this file)
//   - Math (stdlib_math.go): ABS, ROUND, FLOOR, CEIL, TRUNC, MOD, POW,
//     SQRT, LN, LOG, LOG10, EXP, SIGN
//   - String (stdlib_string.go): LEN, UPPER, LOWER, TRIM/L/RTRIM,
//     LEFT, RIGHT, MID/SUBSTR, REPLACE, FIND, CONTAINS, STARTSWITH,
//     ENDSWITH, REGEX_MATCH, REPEAT
//   - Type conversion (stdlib_typeconv.go): TONUM/VALUE, TOSTR/TEXT,
//     TOBOOL, COALESCE, IFNULL
//   - Date components (stdlib_datetime.go): YEAR, MONTH, DAYOFMONTH,
//     WEEKDAY, DATEDIFF, DATEADD, FORMAT_DATE
//   - Aggregates: SUM, AVG, COUNT, MAX, MIN (this file) and
//     MEDIAN, STDEV/STDEVP, VAR/VARP (stdlib_aggregates.go)
func RegisterStandardFunctions() {
	registerMathFunctions()
	registerStringFunctions()
	registerTypeConversionFunctions()
	registerDateTimeFunctions()
	registerAggregateStatFunctions()

	// Logical Functions
	//
	// IF, AND, OR and CASE are NOT registered here. They control which of their
	// arguments get evaluated, which a registered function cannot do — by the
	// time one is called every argument already has a value — so the evaluator
	// implements them directly (see evaluateWithCallDepth). They used to be
	// registered as well, and those copies were unreachable: their argument
	// checks never ran, which is why AND() with no arguments was true and
	// AND('abc', true) was a silent false.

	// String Functions
	registerFunction("CONCAT", func(args ...any) (any, error) {
		if len(args) < 2 {
			return nil, fmt.Errorf("CONCAT requires at least 2 arguments")
		}
		var sb strings.Builder
		for _, arg := range args {
			// toString, not Fprint: fmt renders nil as "<nil>", which then
			// lands in a cell as data.
			sb.WriteString(toString(arg))
		}
		return sb.String(), nil
	})

	// Null/NaN Checks
	registerFunction("ISNA", func(args ...any) (any, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("ISNA requires 1 argument")
		}
		val := args[0]
		switch v := val.(type) {
		case float64:
			return math.IsNaN(v), nil
		case float32:
			return math.IsNaN(float64(v)), nil
		case string:
			return v == "#N/A", nil
		}
		return false, nil
	})

	registerFunction("IFNA", func(args ...any) (any, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("IFNA requires 2 arguments")
		}
		val := args[0]
		isNA := false
		switch v := val.(type) {
		case float64:
			isNA = math.IsNaN(v)
		case float32:
			isNA = math.IsNaN(float64(v))
		case string:
			isNA = v == "#N/A"
		}

		if isNA {
			return args[1], nil
		}
		return val, nil
	})

	// Aggregate Functions
	registerAggregateFunction("SUM", func(args ...[]any) (any, error) {
		if len(args) == 0 {
			return 0.0, nil
		}
		var sum float64
		exact := newIntegerAggregate("SUM")
		forEachValue(args, func(val any) {
			if f, ok := toFloat64(val); ok && !math.IsNaN(f) {
				sum += f
				exact.add(val)
			}
		})
		if res, ok, err := exact.result(); ok {
			return res, err
		}
		return sum, nil
	})

	registerAggregateFunction("AVG", func(args ...[]any) (any, error) {
		if len(args) == 0 {
			return 0.0, nil
		}
		var sum float64
		var count int
		forEachValue(args, func(val any) {
			if f, ok := toFloat64(val); ok && !math.IsNaN(f) {
				sum += f
				count++
			}
		})
		if count == 0 {
			return 0.0, nil
		}
		return sum / float64(count), nil
	})

	registerAggregateFunction("COUNT", func(args ...[]any) (any, error) {
		var count int
		forEachValue(args, func(val any) {
			if val != nil {
				count++
			}
		})
		return float64(count), nil
	})

	registerAggregateFunction("MAX", func(args ...[]any) (any, error) {
		if len(args) == 0 {
			return nil, nil
		}
		maxVal := -math.MaxFloat64
		found := false
		exact := newIntegerAggregate("MAX")
		forEachValue(args, func(val any) {
			if f, ok := toFloat64(val); ok && !math.IsNaN(f) {
				exact.add(val)
				if f > maxVal {
					maxVal = f
					found = true
				}
			}
		})
		if !found {
			return nil, nil
		}
		if res, ok, err := exact.result(); ok {
			return res, err
		}
		return maxVal, nil
	})

	registerAggregateFunction("MIN", func(args ...[]any) (any, error) {
		if len(args) == 0 {
			return nil, nil
		}
		minVal := math.MaxFloat64
		found := false
		exact := newIntegerAggregate("MIN")
		forEachValue(args, func(val any) {
			if f, ok := toFloat64(val); ok && !math.IsNaN(f) {
				exact.add(val)
				if f < minVal {
					minVal = f
					found = true
				}
			}
		})
		if !found {
			return nil, nil
		}
		if res, ok, err := exact.result(); ok {
			return res, err
		}
		return minVal, nil
	})

	// Duration helpers: convert a time.Duration, a duration string or a
	// number of seconds to a unit.
	registerFunction("DAY", durationIn("DAY", "days", func(d time.Duration) float64 { return d.Hours() / 24.0 },
		"day of the month", "DAYOFMONTH(x)"))
	registerFunction("HOUR", durationIn("HOUR", "hours", time.Duration.Hours,
		"hour", "TONUM(FORMAT_DATE(x, '15'))"))
	registerFunction("MINUTE", durationIn("MINUTE", "minutes", time.Duration.Minutes,
		"minute", "TONUM(FORMAT_DATE(x, '04'))"))
	registerFunction("SECOND", durationIn("SECOND", "seconds", time.Duration.Seconds,
		"second", "TONUM(FORMAT_DATE(x, '05'))"))
}

// durationIn builds DAY, HOUR, MINUTE or SECOND: the argument as a number of
// units. Excel's functions of these names take a part of a date instead, so a
// date is refused with a pointer to the function that does that, rather than
// read as the time since midnight.
func durationIn(name, units string, in func(time.Duration) float64, part, partFunc string) func(args ...any) (any, error) {
	refuseDate := func() error {
		return fmt.Errorf("%s converts a duration to %s and was given a date; for the %s of a date x, use %s",
			name, units, part, partFunc)
	}
	return func(args ...any) (any, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("%s requires 1 argument", name)
		}
		var d time.Duration
		switch x := args[0].(type) {
		case time.Duration:
			d = x
		case time.Time:
			return nil, refuseDate()
		case string:
			pd, err := time.ParseDuration(x)
			if err == nil {
				d = pd
				break
			}
			if _, ok := utils.TryParseTime(x); ok {
				return nil, refuseDate()
			}
			return nil, fmt.Errorf("%s: cannot parse %q as a duration", name, x)
		default:
			f, ok := toFloat64(x)
			if !ok {
				return nil, fmt.Errorf("unsupported type for %s: %T", name, x)
			}
			// A number is a count of seconds.
			if d, ok = durationOf(f, time.Second); !ok {
				return nil, fmt.Errorf("%s: %v seconds is out of range", name, f)
			}
		}
		return in(d), nil
	}
}

// Helper for aggregate functions
func forEachValue(args [][]any, fn func(val any)) {
	var walk func(v any)
	walk = func(v any) {
		if slice, ok := v.([]any); ok {
			for _, item := range slice {
				walk(item)
			}
		} else if v != nil {
			fn(v)
		}
	}
	for _, col := range args {
		for _, val := range col {
			walk(val)
		}
	}
}

// toFloat64 converts a value to float64.
// Exported for use in standard library functions.
func toFloat64(val any) (float64, bool) {
	switch v := val.(type) {
	case float64:
		return v, true
	case int:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case int8:
		return float64(v), true
	case int16:
		return float64(v), true
	case uint:
		return float64(v), true
	case uint8:
		return float64(v), true
	case uint16:
		return float64(v), true
	case uint32:
		return float64(v), true
	case uint64:
		// A uint64 above 2^53 lands on the nearest float64, the way an int64
		// that large already does.
		return float64(v), true
	case float32:
		return float64(v), true
	case time.Duration:
		// A date difference is a number of seconds, so (A - B) > 0 and
		// (A - B) / 86400 behave the way the docs describe.
		return v.Seconds(), true
	case bool:
		if v {
			return 1.0, true
		}
		return 0.0, true
	case string:
		trimmed := strings.TrimSpace(v)
		f, err := strconv.ParseFloat(trimmed, 64)
		return f, err == nil
	case nil:
		return 0.0, true
	default:
		return 0, false
	}
}

// toBool converts a value to boolean.
// Exported for use in standard library functions.
func toBool(val any) (bool, bool) {
	switch v := val.(type) {
	case bool:
		return v, true
	case float64:
		return v != 0, true
	case int:
		return v != 0, true
	case int32:
		return v != 0, true
	case int64:
		return v != 0, true
	case int8:
		return v != 0, true
	case int16:
		return v != 0, true
	case uint:
		return v != 0, true
	case uint8:
		return v != 0, true
	case uint16:
		return v != 0, true
	case uint32:
		return v != 0, true
	case uint64:
		return v != 0, true
	case float32:
		return v != 0, true
	case string:
		lower := strings.ToLower(strings.TrimSpace(v))
		if lower == "true" || lower == "yes" || lower == "1" {
			return true, true
		}
		if lower == "false" || lower == "no" || lower == "0" || lower == "" {
			return false, true
		}
		// For other strings, maybe consider them true if not empty?
		// But "false" check above handles empty string as false.
		// Let's stick to strict parsing for now.
		return false, false
	case nil:
		return false, true
	default:
		return false, false
	}
}
