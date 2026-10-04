package ccl

import (
	"strings"
	"testing"
	"time"
)

// DAY, HOUR, MINUTE and SECOND take a part of a date, as in Excel, Google
// Sheets, DAX and Spark SQL. They used to convert a duration to a number of
// units, so DAY('2024-01-02T06:00:00Z') was 0.25 where every spreadsheet user
// expects 2. Expressing a duration in a unit is DATEDIFF's job, or plain
// arithmetic on the difference, which counts seconds.
func TestDatePartFunctionsLikeExcel(t *testing.T) {
	taipei := time.FixedZone("UTC+8", 8*3600)
	ctx := mapCtx(t, map[string][]any{
		"A": {time.Date(2024, 1, 2, 6, 30, 15, 999_000_000, time.UTC)},
		"B": {"2024-12-31T23:59:58+08:00"},
		"C": {"2024-03-09"},
		"D": {time.Date(2024, 1, 31, 7, 0, 0, 0, taipei)},
	})
	for _, c := range []struct {
		expr string
		want float64
	}{
		{"DAY(A)", 2},
		{"HOUR(A)", 6},
		{"MINUTE(A)", 30},
		{"SECOND(A)", 15}, // a fraction of a second is dropped, as in Excel
		{"DAY(B)", 31},    // read in the date's own time zone
		{"HOUR(B)", 23},
		{"DAY(C)", 9},
		{"HOUR(C)", 0},
		{"DAY(D)", 31},
		{"HOUR(D)", 7},
		{"DAY('2024-01-02T06:00:00Z')", 2},
		// The same answers as DATEPART and DAYOFMONTH.
		{"DATEPART(A, 'day') - DAY(A)", 0},
		{"DAYOFMONTH(A) - DAY(A)", 0},
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
}

// A duration, a duration string or a number is not a date. The error says so
// and names the way to express a duration in a unit.
func TestDatePartFunctionsRefuseADuration(t *testing.T) {
	ctx := mapCtx(t, map[string][]any{
		"A": {time.Date(2024, 1, 3, 12, 0, 0, 0, time.UTC)},
		"B": {time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)},
		"C": {36 * time.Hour},
	})
	for _, c := range []struct {
		expr string
		hint string
	}{
		{"DAY(A - B)", "DATEDIFF(end, start, 'day')"},
		{"HOUR(A - B)", "DATEDIFF(end, start, 'hour')"},
		{"MINUTE(C)", "DATEDIFF(end, start, 'minute')"},
		{"SECOND('1m30s')", "DATEDIFF(end, start, 'second')"},
		{"DAY('36h')", "DATEDIFF(end, start, 'day')"},
		{"HOUR(7200)", "DATEDIFF(end, start, 'hour')"},
		{"DAY(1)", "DATEDIFF(end, start, 'day')"},
	} {
		got, err := evalOne(t, ctx, c.expr)
		if err == nil {
			t.Errorf("%s = %#v, want an error", c.expr, got)
			continue
		}
		if !strings.Contains(err.Error(), "date") || !strings.Contains(err.Error(), c.hint) {
			t.Errorf("%s: the error %q does not say it needs a date and point to %s", c.expr, err, c.hint)
		}
	}
	for _, expr := range []string{"DAY('soon')", "HOUR(true)", "DAY()", "DAY(A, B)"} {
		if got, err := evalOne(t, ctx, expr); err == nil {
			t.Errorf("%s = %#v, want an error", expr, got)
		}
	}

	// What the errors point to gives what DAY(A - B) used to.
	for _, c := range []struct {
		expr string
		want float64
	}{
		{"DATEDIFF(A, B, 'day')", 1.5},
		{"DATEDIFF(A, B, 'hour')", 36},
		{"(A - B) / 86400", 1.5},
		{"(A - B) / 3600", 36},
	} {
		got, err := evalOne(t, ctx, c.expr)
		if err != nil {
			t.Errorf("%s: %v", c.expr, err)
			continue
		}
		if got != c.want {
			t.Errorf("%s = %#v, want %v", c.expr, got, c.want)
		}
	}
}
