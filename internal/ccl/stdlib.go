package ccl

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/HazelnutParadise/insyra/internal/utils"
)

// warnCountChanges warns when COUNT counts a value it will not count from 0.4:
// text, a bool or NaN.
func warnCountChanges(val any) {
	counted := true
	switch val.(type) {
	case nil, string, bool:
		counted = false
	case time.Time:
	default:
		f, ok := toFloat64(val)
		counted = ok && !math.IsNaN(f)
	}
	if !counted {
		warnOnce("COUNT", "COUNT counts text here, and from 0.4 it counts numbers and dates only, as in Excel; COUNTA counts every value that is not nil in both")
	}
}

// isConditionalAggregate reports whether name takes a condition first.
func isConditionalAggregate(name string) bool {
	switch strings.ToUpper(name) {
	case "COUNTIF", "SUMIF", "AVGIF":
		return true
	}
	return false
}

// conditionMet reads one row of a condition. An empty row does not meet it.
func conditionMet(name string, cond any, row int) (bool, error) {
	if cond == nil {
		return false, nil
	}
	met, ok := toBool(cond)
	if !ok {
		return false, fmt.Errorf("%s: the condition at row %d is %v (%T), which is not true or false", name, row, cond, cond)
	}
	return met, nil
}

// valuesWhere gives the values of a column on the rows where the condition holds,
// for SUMIF and AVGIF.
func valuesWhere(name string, args [][]any) ([]any, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("%s requires 2 arguments, a condition and a column, such as %s(A > 0, B)", name, name)
	}
	conds, values := args[0], args[1]
	if len(conds) != len(values) {
		return nil, fmt.Errorf("%s: the condition has %d rows and the column %d", name, len(conds), len(values))
	}
	var kept []any
	for row, v := range values {
		met, err := conditionMet(name, conds[row], row)
		if err != nil {
			return nil, err
		}
		if met {
			kept = append(kept, v)
		}
	}
	return kept, nil
}

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
//     BOOL, COALESCE, IFNULL
//   - Date components (stdlib_datetime.go): YEAR, MONTH, DAYOFMONTH,
//     WEEKDAY, DATEPART, DATEDIFF, DATEADD, FORMAT_DATE,
//     and DATE, TODAY, NOW (clock.go)
//   - Type checks (stdlib_typecheck.go): ISBLANK, ISNUM, ISTEXT, ISBOOL,
//     ISDATE; ISERROR and IFERROR are in the evaluator
//   - Aggregates: SUM, AVG, COUNT, COUNTA, COUNTIF, SUMIF, AVGIF, MAX, MIN (this file) and
//     MEDIAN, STDEV/STDEVP, VAR/VARP (stdlib_aggregates.go)
func RegisterStandardFunctions() {
	registerTypeCheckFunctions()
	registerMathFunctions()
	registerStringFunctions()
	registerTypeConversionFunctions()
	registerDateTimeFunctions()
	registerAggregateStatFunctions()

	// Logical Functions
	//
	// IF, AND and OR are NOT registered here. The evaluator implements them
	// directly (see evaluateWithCallDepth) so they can skip arguments they do
	// not need, and it handles them before any registered function is looked
	// up. The copies that used to be registered were unreachable: their
	// argument checks never ran. CASE has no such special case, so its
	// registration below is the one that runs.
	registerFunction("CASE", func(args ...any) (any, error) {
		if len(args) < 3 {
			return nil, fmt.Errorf("CASE requires at least 3 arguments")
		}
		if len(args)%2 != 1 {
			return nil, fmt.Errorf("CASE requires an odd number of arguments")
		}

		for i := 0; i < len(args)-1; i += 2 {
			if cond, ok := toBool(args[i]); ok {
				if cond {
					return args[i+1], nil
				}
			} else {
				return nil, fmt.Errorf("condition at position %d cannot be evaluated as boolean", i)
			}
		}
		return args[len(args)-1], nil
	})

	// String Functions
	registerFunction("CONCAT", func(args ...any) (any, error) {
		if len(args) < 2 {
			return nil, fmt.Errorf("CONCAT requires at least 2 arguments")
		}
		var sb strings.Builder
		for _, arg := range args {
			sb.WriteString(textOf(arg))
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
	sumAggregate := func(args ...[]any) (any, error) {
		if len(args) == 0 {
			return 0.0, nil
		}
		var sum float64
		forEachValue(args, func(val any) {
			if f, ok := toFloat64(val); ok {
				sum += f
			}
		})
		return sum, nil
	}
	registerAggregateFunction("SUM", sumAggregate)

	avgAggregate := func(args ...[]any) (any, error) {
		if len(args) == 0 {
			return 0.0, nil
		}
		var sum float64
		var count int
		forEachValue(args, func(val any) {
			if f, ok := toFloat64(val); ok {
				sum += f
				count++
			}
		})
		if count == 0 {
			return 0.0, nil
		}
		return sum / float64(count), nil
	}
	registerAggregateFunction("AVG", avgAggregate)

	registerAggregateFunction("COUNT", func(args ...[]any) (any, error) {
		var count int
		forEachValue(args, func(val any) {
			if val != nil {
				warnCountChanges(val)
				count++
			}
		})
		return float64(count), nil
	})

	// COUNTA counts every value that is not nil, as COUNT does here. From 0.4
	// COUNT counts numbers and dates only, so COUNTA gives this answer on both.
	registerAggregateFunction("COUNTA", func(args ...[]any) (any, error) {
		var count int
		forEachValue(args, func(val any) {
			if val != nil {
				count++
			}
		})
		return float64(count), nil
	})

	// COUNTIF(condition), SUMIF(condition, values) and AVGIF(condition, values)
	// take the condition as an expression, not as text to parse: COUNTIF(A > 0).
	registerAggregateFunction("COUNTIF", func(args ...[]any) (any, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("COUNTIF requires 1 argument, a condition such as A > 0")
		}
		var count int
		for row, cond := range args[0] {
			met, err := conditionMet("COUNTIF", cond, row)
			if err != nil {
				return nil, err
			}
			if met {
				count++
			}
		}
		return float64(count), nil
	})

	registerAggregateFunction("SUMIF", func(args ...[]any) (any, error) {
		values, err := valuesWhere("SUMIF", args)
		if err != nil {
			return nil, err
		}
		return sumAggregate(values)
	})

	registerAggregateFunction("AVGIF", func(args ...[]any) (any, error) {
		values, err := valuesWhere("AVGIF", args)
		if err != nil {
			return nil, err
		}
		return avgAggregate(values)
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
	registerDurationPart("DAY", func(args ...any) (any, error) {
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
				if d, ok = durationOf(f, time.Second); !ok {
					return nil, fmt.Errorf("DAY: %v seconds is out of range", f)
				}
			} else {
				return nil, fmt.Errorf("unsupported type for DAY: %T", val)
			}
		}
		return d.Hours() / 24.0, nil
	})

	registerDurationPart("HOUR", func(args ...any) (any, error) {
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
				if d, ok = durationOf(f, time.Second); !ok {
					return nil, fmt.Errorf("HOUR: %v seconds is out of range", f)
				}
			} else {
				return nil, fmt.Errorf("unsupported type for HOUR: %T", val)
			}
		}
		return d.Hours(), nil
	})

	registerDurationPart("MINUTE", func(args ...any) (any, error) {
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
				if d, ok = durationOf(f, time.Second); !ok {
					return nil, fmt.Errorf("MINUTE: %v seconds is out of range", f)
				}
			} else {
				return nil, fmt.Errorf("unsupported type for MINUTE: %T", val)
			}
		}
		return d.Minutes(), nil
	})

	registerDurationPart("SECOND", func(args ...any) (any, error) {
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
				if d, ok = durationOf(f, time.Second); !ok {
					return nil, fmt.Errorf("SECOND: %v seconds is out of range", f)
				}
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
