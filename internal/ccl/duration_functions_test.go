package ccl

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// CCL-22 (#359): DAY, HOUR, MINUTE and SECOND convert a duration to a unit,
// while Excel's functions of the same names take a part of a date. A date
// reaching them used to be read as the time since midnight, so
// DAY('2024-01-02T06:00:00Z') was 0.25 where an Excel user expects 2.
func TestDurationFunctionsRefuseADate(t *testing.T) {
	jan2 := time.Date(2024, 1, 2, 6, 30, 15, 0, time.UTC)
	ctx := mapCtx(t, map[string][]any{
		"A": {jan2},
		"B": {"2024-01-02T06:30:15Z"},
		"C": {"2024-01-02"},
	})
	for _, c := range []struct {
		expr string
		hint []string // what the error must point to
	}{
		{"DAY('2024-01-02T06:00:00Z')", []string{"DAYOFMONTH("}},
		{"DAY(C)", []string{"DAYOFMONTH("}},
		{"DAY(A)", []string{"DAYOFMONTH("}},
		{"HOUR(B)", []string{"FORMAT_DATE(", "'15'"}},
		{"HOUR(A)", []string{"FORMAT_DATE(", "'15'"}},
		{"MINUTE(B)", []string{"FORMAT_DATE(", "'04'"}},
		{"MINUTE(C)", []string{"FORMAT_DATE(", "'04'"}},
		{"SECOND(B)", []string{"FORMAT_DATE(", "'05'"}},
		{"SECOND(A)", []string{"FORMAT_DATE(", "'05'"}},
	} {
		got, err := evalCol(t, ctx, c.expr)
		if err == nil {
			t.Errorf("%s = %v, want an error", c.expr, got)
			continue
		}
		if !strings.Contains(err.Error(), "duration") {
			t.Errorf("%s: the error %q does not say the function takes a duration", c.expr, err)
		}
		for _, h := range c.hint {
			if !strings.Contains(err.Error(), h) {
				t.Errorf("%s: the error %q does not mention %s", c.expr, err, h)
			}
		}
	}
}

// The functions the errors point to give the part Excel would.
func TestDatePartReplacementsGiveTheExcelPart(t *testing.T) {
	ctx := mapCtx(t, map[string][]any{"A": {"2024-01-02T06:30:15Z"}})
	for _, c := range []struct {
		expr string
		want float64
	}{
		{"DAYOFMONTH(A)", 2},
		{"TONUM(FORMAT_DATE(A, '15'))", 6},
		{"TONUM(FORMAT_DATE(A, '04'))", 30},
		{"TONUM(FORMAT_DATE(A, '05'))", 15},
	} {
		got, err := evalCol(t, ctx, c.expr)
		if err != nil {
			t.Errorf("%s: %v", c.expr, err)
			continue
		}
		if got[0] != c.want {
			t.Errorf("%s = %v, want %v", c.expr, got[0], c.want)
		}
	}
}

// What the functions are for still works: a duration, a duration string, a
// number of seconds, and the difference of two dates.
func TestDurationFunctionsStillConvertDurations(t *testing.T) {
	ctx := mapCtx(t, map[string][]any{
		"A": {time.Date(2024, 1, 3, 12, 0, 0, 0, time.UTC)},
		"B": {time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)},
		"C": {36 * time.Hour},
	})
	for _, c := range []struct {
		expr string
		want float64
	}{
		{"DAY(A - B)", 1.5},
		{"HOUR(A - B)", 36},
		{"DAY(C)", 1.5},
		{"DAY('36h')", 1.5},
		{"MINUTE('90s')", 1.5},
		{"SECOND('1m30s')", 90},
		{"HOUR(7200)", 2},
		{"DAY(86400)", 1},
	} {
		got, err := evalCol(t, ctx, c.expr)
		if err != nil {
			t.Errorf("%s: %v", c.expr, err)
			continue
		}
		if fmt.Sprint(got[0]) != fmt.Sprint(c.want) {
			t.Errorf("%s = %v, want %v", c.expr, got[0], c.want)
		}
	}

	// A string that is neither is still an error.
	if _, err := evalCol(t, ctx, "DAY('soon')"); err == nil {
		t.Error("DAY('soon') succeeded")
	}
}
