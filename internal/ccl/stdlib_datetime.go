package ccl

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/HazelnutParadise/insyra/internal/utils"
)

// toTime coerces a value to time.Time. Accepts time.Time directly or a
// string parseable by utils.TryParseTime.
func toTime(val any) (time.Time, bool) {
	switch x := val.(type) {
	case time.Time:
		return x, true
	case string:
		if t, ok := utils.TryParseTime(x); ok {
			return t, true
		}
	}
	return time.Time{}, false
}

// registerDateTimeFunctions registers date-component extraction and arithmetic
// helpers. These complement the existing DAY/HOUR/MINUTE/SECOND duration
// functions in stdlib.go, which operate on time.Duration values.
func registerDateTimeFunctions() {
	registerFunction("YEAR", func(args ...any) (any, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("YEAR requires 1 argument")
		}
		t, ok := toTime(args[0])
		if !ok {
			return nil, fmt.Errorf("YEAR: cannot convert %T to date", args[0])
		}
		return float64(t.Year()), nil
	})

	registerFunction("MONTH", func(args ...any) (any, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("MONTH requires 1 argument")
		}
		t, ok := toTime(args[0])
		if !ok {
			return nil, fmt.Errorf("MONTH: cannot convert %T to date", args[0])
		}
		return float64(t.Month()), nil
	})

	registerFunction("DAYOFMONTH", func(args ...any) (any, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("DAYOFMONTH requires 1 argument")
		}
		t, ok := toTime(args[0])
		if !ok {
			return nil, fmt.Errorf("DAYOFMONTH: cannot convert %T to date", args[0])
		}
		return float64(t.Day()), nil
	})

	// WEEKDAY returns 0 (Sunday) through 6 (Saturday), matching time.Weekday.
	registerFunction("WEEKDAY", func(args ...any) (any, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("WEEKDAY requires 1 argument")
		}
		t, ok := toTime(args[0])
		if !ok {
			return nil, fmt.Errorf("WEEKDAY: cannot convert %T to date", args[0])
		}
		return float64(t.Weekday()), nil
	})

	// DATEDIFF(d1, d2, unit) returns d1 - d2 expressed in the requested unit.
	// Supported units (case-insensitive): "day"/"days", "hour"/"hours",
	// "minute"/"minutes", "second"/"seconds".
	registerFunction("DATEDIFF", func(args ...any) (any, error) {
		if len(args) != 3 {
			return nil, fmt.Errorf("DATEDIFF requires 3 arguments (d1, d2, unit)")
		}
		t1, ok := toTime(args[0])
		if !ok {
			return nil, fmt.Errorf("DATEDIFF: cannot convert first arg %T to date", args[0])
		}
		t2, ok := toTime(args[1])
		if !ok {
			return nil, fmt.Errorf("DATEDIFF: cannot convert second arg %T to date", args[1])
		}
		unit, ok := args[2].(string)
		if !ok {
			return nil, fmt.Errorf("DATEDIFF: unit must be a string, got %T", args[2])
		}
		diff := t1.Sub(t2)
		switch strings.ToLower(unit) {
		case "day", "days":
			return diff.Hours() / 24.0, nil
		case "hour", "hours":
			return diff.Hours(), nil
		case "minute", "minutes":
			return diff.Minutes(), nil
		case "second", "seconds":
			return diff.Seconds(), nil
		default:
			return nil, fmt.Errorf("DATEDIFF: unknown unit %q (expected day/hour/minute/second)", unit)
		}
	})

	// DATEADD(d, n, unit) returns d shifted by n units. Supports the same
	// unit set as DATEDIFF plus "month"/"year" via time.AddDate.
	registerFunction("DATEADD", func(args ...any) (any, error) {
		if len(args) != 3 {
			return nil, fmt.Errorf("DATEADD requires 3 arguments (d, n, unit)")
		}
		t, ok := toTime(args[0])
		if !ok {
			return nil, fmt.Errorf("DATEADD: cannot convert first arg %T to date", args[0])
		}
		n, ok := toFloat64(args[1])
		if !ok {
			return nil, fmt.Errorf("DATEADD: n must be a number, got %T", args[1])
		}
		unit, ok := args[2].(string)
		if !ok {
			return nil, fmt.Errorf("DATEADD: unit must be a string, got %T", args[2])
		}
		switch strings.ToLower(unit) {
		case "day", "days", "month", "months", "year", "years":
			// AddDate takes an int. Past the int32 range nobody means the
			// shift, and int(n) there differs by platform.
			if math.IsNaN(n) || n > math.MaxInt32 || n < math.MinInt32 {
				return nil, fmt.Errorf("DATEADD: a shift of %v %s is out of range", args[1], unit)
			}
		}
		switch strings.ToLower(unit) {
		case "day", "days":
			return t.AddDate(0, 0, int(n)), nil // a fraction truncates, as it always has
		case "hour", "hours":
			return addDuration(t, n, time.Hour, args[1], unit)
		case "minute", "minutes":
			return addDuration(t, n, time.Minute, args[1], unit)
		case "second", "seconds":
			return addDuration(t, n, time.Second, args[1], unit)
		case "month", "months":
			return t.AddDate(0, int(n), 0), nil
		case "year", "years":
			return t.AddDate(int(n), 0, 0), nil
		default:
			return nil, fmt.Errorf("DATEADD: unknown unit %q", unit)
		}
	})

	// FORMAT_DATE(d, layout) formats a date using a Go reference layout
	// (e.g. "2006-01-02 15:04:05").
	registerFunction("FORMAT_DATE", func(args ...any) (any, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("FORMAT_DATE requires 2 arguments (date, layout)")
		}
		t, ok := toTime(args[0])
		if !ok {
			return nil, fmt.Errorf("FORMAT_DATE: cannot convert %T to date", args[0])
		}
		layout, ok := args[1].(string)
		if !ok {
			return nil, fmt.Errorf("FORMAT_DATE: layout must be a string, got %T", args[1])
		}
		return t.Format(layout), nil
	})
}

// addDuration shifts t by n units of a clock unit, refusing a shift a Duration
// cannot hold instead of converting it differently on each platform.
func addDuration(t time.Time, n float64, unit time.Duration, arg any, unitName string) (any, error) {
	d, ok := durationOf(n, unit)
	if !ok {
		return nil, fmt.Errorf("DATEADD: a shift of %v %s is out of range", arg, unitName)
	}
	return t.Add(d), nil
}
