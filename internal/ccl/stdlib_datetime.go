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
// helpers.
func registerDateTimeFunctions() {
	// DAY, HOUR, MINUTE and SECOND take a part of a date, as in Excel, Google
	// Sheets, DAX and Spark SQL. A duration is refused with a pointer to
	// DATEDIFF, which expresses one in a unit.
	for _, p := range []struct{ name, unit string }{
		{"DAY", "day"}, {"HOUR", "hour"}, {"MINUTE", "minute"}, {"SECOND", "second"},
	} {
		registerFunction(p.name, datePartFunction(p.name, p.unit))
	}

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

	// DATEPART(d, unit) returns one part of d, read in d's own time zone:
	// "year", "month", "day", "hour", "minute" or "second" (singular or
	// plural, any case). It is the third of the DATEADD / DATEDIFF / DATEPART
	// family, and gives what YEAR, MONTH, DAY, HOUR, MINUTE and SECOND give.
	registerFunction("DATEPART", func(args ...any) (any, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("DATEPART requires 2 arguments (d, unit)")
		}
		t, ok := toTime(args[0])
		if !ok {
			return nil, fmt.Errorf("DATEPART: cannot convert first arg %T to date", args[0])
		}
		unit, ok := args[1].(string)
		if !ok {
			return nil, fmt.Errorf("DATEPART: unit must be a string, got %T", args[1])
		}
		v, ok := datePart(t, unit)
		if !ok {
			return nil, fmt.Errorf("DATEPART: unknown unit %q (expected year/month/day/hour/minute/second)", unit)
		}
		return v, nil
	})

	// DATEADD(d, n, unit) returns d shifted by n units. Supports the same
	// unit set as DATEDIFF plus "month"/"year", which stop at the last day of
	// a month that is too short, as Excel's EDATE does.
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
			return addMonths(t, 0, int(n)), nil
		case "year", "years":
			return addMonths(t, int(n), 0), nil
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

// addMonths shifts t by years and months, keeping its day unless the target
// month is shorter, in which case the result is that month's last day:
// January 31 plus one month is February 29 in 2024. time.AddDate would carry
// the missing days into the next month and give March 2. Excel's EDATE and
// pandas' DateOffset both stop at the month's end.
func addMonths(t time.Time, years, months int) time.Time {
	// The month is worked out in UTC, where no clock change can move it.
	first := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(years, months, 0)
	// Day 0 of the next month is the last day of this one.
	last := time.Date(first.Year(), first.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()
	hour, minute, sec := t.Clock()
	return time.Date(first.Year(), first.Month(), min(t.Day(), last), hour, minute, sec, t.Nanosecond(), t.Location())
}

// datePart returns the unit part of t, read in t's own time zone, dropping any
// fraction of a second as Excel's SECOND does. ok is false for an unknown unit.
func datePart(t time.Time, unit string) (float64, bool) {
	switch strings.ToLower(unit) {
	case "year", "years":
		return float64(t.Year()), true
	case "month", "months":
		return float64(t.Month()), true
	case "day", "days":
		return float64(t.Day()), true
	case "hour", "hours":
		return float64(t.Hour()), true
	case "minute", "minutes":
		return float64(t.Minute()), true
	case "second", "seconds":
		return float64(t.Second()), true
	}
	return 0, false
}

// datePartFunction builds DAY, HOUR, MINUTE or SECOND, which give one part of
// a date. These names used to convert a duration to a number of units, so a
// duration, a duration string or a number gets an error naming DATEDIFF rather
// than a generic one.
func datePartFunction(name, unit string) func(args ...any) (any, error) {
	refuseDuration := func(what string) error {
		return fmt.Errorf("%s takes a date and was given %s; to express a duration in %ss, use DATEDIFF(end, start, '%s'), or divide the difference of two dates, which counts seconds", name, what, unit, unit)
	}
	return func(args ...any) (any, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("%s requires 1 argument", name)
		}
		if t, ok := toTime(args[0]); ok {
			v, _ := datePart(t, unit)
			return v, nil
		}
		switch x := args[0].(type) {
		case time.Duration:
			return nil, refuseDuration("a duration")
		case string:
			if _, err := time.ParseDuration(x); err == nil {
				return nil, refuseDuration("a duration")
			}
			return nil, fmt.Errorf("%s: cannot read %q as a date", name, x)
		case bool:
			return nil, fmt.Errorf("%s: cannot convert %T to date", name, x)
		}
		if _, ok := toFloat64(args[0]); ok {
			return nil, refuseDuration("a number, which CCL does not read as a date")
		}
		return nil, fmt.Errorf("%s: cannot convert %T to date", name, args[0])
	}
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
