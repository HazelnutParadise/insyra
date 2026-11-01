package ccl

import (
	"testing"
	"time"
)

// DAY, HOUR, MINUTE and SECOND read a date, and still convert a duration until 0.4.
func TestDatePartFunctionsReadADate(t *testing.T) {
	ctx := mapCtx(t, map[string][]any{
		"A": {"2024-03-09T06:07:08Z"},
		"B": {time.Date(2024, 12, 31, 23, 59, 58, 0, time.UTC)},
		"C": {90 * time.Minute},
	})
	for _, c := range []struct {
		expr string
		want float64
	}{
		{"DAY(A)", 9},
		{"HOUR(A)", 6},
		{"MINUTE(A)", 7},
		{"SECOND(A)", 8},
		{"DAY(B)", 31},
		{"HOUR(B)", 23},
		{"MINUTE(B)", 59},
		{"SECOND(B)", 58},
		{"DAY('2024-01-02')", 2},
		{"HOUR(C)", 1.5},
		{"MINUTE(C)", 90},
		{"HOUR('36h')", 36},
		{"DAY('36h')", 1.5},
		{"HOUR(7200)", 2},
		{"DAYOFMONTH(A)", 9},
		{"DATEPART(A, 'hour')", 6},
	} {
		got, err := evalOne(t, ctx, c.expr)
		if err != nil {
			t.Errorf("%s: %v", c.expr, err)
			continue
		}
		if f, ok := toFloat64(got); !ok || f != c.want {
			t.Errorf("%s = %#v, want %v", c.expr, got, c.want)
		}
	}
}
