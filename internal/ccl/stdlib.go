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
		forEachValue(args, func(val any) {
			if f, ok := toFloat64(val); ok && !math.IsNaN(f) {
				sum += f
			}
		})
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
		forEachValue(args, func(val any) {
			if f, ok := toFloat64(val); ok {
				if f > maxVal {
					maxVal = f
					found = true
				}
			}
		})
		if !found {
			return nil, nil
		}
		return maxVal, nil
	})

	registerAggregateFunction("MIN", func(args ...[]any) (any, error) {
		if len(args) == 0 {
			return nil, nil
		}
		minVal := math.MaxFloat64
		found := false
		forEachValue(args, func(val any) {
			if f, ok := toFloat64(val); ok {
				if f < minVal {
					minVal = f
					found = true
				}
			}
		})
		if !found {
			return nil, nil
		}
		return minVal, nil
	})

	// Duration helpers: convert time.Duration (or parsable duration string / numeric seconds) to units
	registerFunction("DAY", func(args ...any) (any, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("DAY requires 1 argument")
		}
		val := args[0]
		var d time.Duration
		switch x := val.(type) {
		case time.Duration:
			d = x
		case string:
			if pd, err := time.ParseDuration(x); err == nil {
				d = pd
			} else if t, ok := utils.TryParseTime(x); ok {
				d = time.Duration(t.Sub(time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())))
			} else {
				return nil, fmt.Errorf("cannot parse duration or date from string: %s", x)
			}
		default:
			if f, ok := toFloat64(val); ok {
				// treat numeric as seconds
				d = time.Duration(f * float64(time.Second))
			} else {
				return nil, fmt.Errorf("unsupported type for DAY: %T", val)
			}
		}
		return d.Hours() / 24.0, nil
	})

	registerFunction("HOUR", func(args ...any) (any, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("HOUR requires 1 argument")
		}
		val := args[0]
		var d time.Duration
		switch x := val.(type) {
		case time.Duration:
			d = x
		case string:
			if pd, err := time.ParseDuration(x); err == nil {
				d = pd
			} else if t, ok := utils.TryParseTime(x); ok {
				d = time.Duration(t.Sub(time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())))
			} else {
				return nil, fmt.Errorf("cannot parse duration or date from string: %s", x)
			}
		default:
			if f, ok := toFloat64(val); ok {
				d = time.Duration(f * float64(time.Second))
			} else {
				return nil, fmt.Errorf("unsupported type for HOUR: %T", val)
			}
		}
		return d.Hours(), nil
	})

	registerFunction("MINUTE", func(args ...any) (any, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("MINUTE requires 1 argument")
		}
		val := args[0]
		var d time.Duration
		switch x := val.(type) {
		case time.Duration:
			d = x
		case string:
			if pd, err := time.ParseDuration(x); err == nil {
				d = pd
			} else if t, ok := utils.TryParseTime(x); ok {
				d = time.Duration(t.Sub(time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())))
			} else {
				return nil, fmt.Errorf("cannot parse duration or date from string: %s", x)
			}
		default:
			if f, ok := toFloat64(val); ok {
				d = time.Duration(f * float64(time.Second))
			} else {
				return nil, fmt.Errorf("unsupported type for MINUTE: %T", val)
			}
		}
		return d.Minutes(), nil
	})

	registerFunction("SECOND", func(args ...any) (any, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("SECOND requires 1 argument")
		}
		val := args[0]
		var d time.Duration
		switch x := val.(type) {
		case time.Duration:
			d = x
		case string:
			if pd, err := time.ParseDuration(x); err == nil {
				d = pd
			} else if t, ok := utils.TryParseTime(x); ok {
				d = time.Duration(t.Sub(time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())))
			} else {
				return nil, fmt.Errorf("cannot parse duration or date from string: %s", x)
			}
		default:
			if f, ok := toFloat64(val); ok {
				d = time.Duration(f * float64(time.Second))
			} else {
				return nil, fmt.Errorf("unsupported type for SECOND: %T", val)
			}
		}
		return d.Seconds(), nil
	})
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
