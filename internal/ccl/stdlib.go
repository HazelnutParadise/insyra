package ccl

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// tryParseTime attempts to parse common date/time string formats and returns
// the parsed time and true on success. Implemented locally to avoid cross-package
// dependency on other helpers.
func tryParseTime(str string) (time.Time, bool) {
	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02",
		"2006-01-02T15:04:05Z07:00",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, str); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// RegisterStandardFunctions registers the standard library of CCL functions.
// This includes logical functions (IF, AND, OR), string functions (CONCAT),
// and aggregate functions (SUM, AVG, COUNT, MAX, MIN).
func RegisterStandardFunctions() {
	// Logical Functions
	registerFunction("IF", func(args ...any) (any, error) {
		if len(args) != 3 {
			return nil, fmt.Errorf("IF requires 3 arguments")
		}

		cond, ok := toBool(args[0])
		if !ok {
			// Try to parse if it's not directly a bool
			// This mimics the behavior in insyra/ccl.go but using our internal helper
			// Note: toBool handles more cases now
			return nil, fmt.Errorf("first argument to IF cannot be converted to boolean: %T", args[0])
		}

		if cond {
			return args[1], nil
		}
		return args[2], nil
	})

	registerFunction("AND", func(args ...any) (any, error) {
		if len(args) < 2 {
			return nil, fmt.Errorf("AND requires at least 2 arguments")
		}
		for _, arg := range args {
			if cond, ok := toBool(arg); !ok || !cond {
				return false, nil
			}
		}
		return true, nil
	})

	registerFunction("OR", func(args ...any) (any, error) {
		if len(args) < 2 {
			return nil, fmt.Errorf("OR requires at least 2 arguments")
		}
		for _, arg := range args {
			if cond, ok := toBool(arg); ok && cond {
				return true, nil
			}
		}
		return false, nil
	})

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
			sb.WriteString(fmt.Sprintf("%v", arg))
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
			if f, ok := toFloat64(val); ok {
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
			if f, ok := toFloat64(val); ok {
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
			} else if t, ok := tryParseTime(x); ok {
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
			} else if t, ok := tryParseTime(x); ok {
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
			} else if t, ok := tryParseTime(x); ok {
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
			} else if t, ok := tryParseTime(x); ok {
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
