package ccl

import (
	"testing"
	"time"
)

// DATEPART(d, unit) takes one part of a date, the way Excel's YEAR, MONTH,
// DAY, HOUR, MINUTE and SECOND do. CCL's HOUR, MINUTE and SECOND convert
// durations, so before this the hour of a date was TONUM(FORMAT_DATE(d, '15')).
func TestDatePart(t *testing.T) {
	taipei := time.FixedZone("UTC+8", 8*3600)
	ctx := mapCtx(t, map[string][]any{
		"A": {"2024-03-09T06:07:08Z"},
		"B": {time.Date(2024, 12, 31, 23, 59, 58, 999_000_000, taipei)},
	})
	for _, c := range []struct {
		expr string
		want float64
	}{
		{"DATEPART(A, 'year')", 2024},
		{"DATEPART(A, 'month')", 3},
		{"DATEPART(A, 'day')", 9},
		{"DATEPART(A, 'hour')", 6},
		{"DATEPART(A, 'minute')", 7},
		{"DATEPART(A, 'second')", 8},
		{"DATEPART(A, 'HOUR')", 6},
		{"DATEPART(A, 'hours')", 6},
		{"DATEPART('2024-03-09', 'hour')", 0},
		// The part is read in the date's own time zone, and a fraction of a
		// second is dropped, as Excel's SECOND drops it.
		{"DATEPART(B, 'day')", 31},
		{"DATEPART(B, 'hour')", 23},
		{"DATEPART(B, 'second')", 58},
	} {
		got, err := evalOne(t, ctx, c.expr)
		if err != nil {
			t.Errorf("%s: %v", c.expr, err)
			continue
		}
		if got != c.want {
			t.Errorf("%s = %#v (%T), want %v", c.expr, got, got, c.want)
		}
	}
	for _, expr := range []string{
		"DATEPART(A, 'week')",
		"DATEPART(A, 1)",
		"DATEPART('soon', 'hour')",
		"DATEPART(A)",
		"DATEPART(A, 'hour', 1)",
	} {
		if got, err := evalOne(t, ctx, expr); err == nil {
			t.Errorf("%s = %#v, want an error", expr, got)
		}
	}
}
