package ccl

import (
	"strings"
	"testing"
)

// DATEPART repeated YEAR, MONTH, DAY, HOUR, MINUTE and SECOND once those read
// dates, so it was removed. A formula still using it is told what to write.
func TestDatePartIsRemovedWithAPointer(t *testing.T) {
	ctx := mapCtx(t, map[string][]any{"A": {"2024-03-09T06:07:08Z"}})
	for _, expr := range []string{"DATEPART(A, 'hour')", "DATEPART(A)", "DATEPART()"} {
		got, err := evalOne(t, ctx, expr)
		if err == nil {
			t.Errorf("%s = %#v, want an error", expr, got)
			continue
		}
		if !strings.Contains(err.Error(), "removed") || !strings.Contains(err.Error(), "HOUR(d)") {
			t.Errorf("%s: the error %q does not say it was removed and what to use", expr, err)
		}
	}
	// What the error points to gives what it gave.
	for _, c := range []struct {
		expr string
		want float64
	}{{"YEAR(A)", 2024}, {"MONTH(A)", 3}, {"DAY(A)", 9}, {"HOUR(A)", 6}, {"MINUTE(A)", 7}, {"SECOND(A)", 8}} {
		got, err := evalOne(t, ctx, c.expr)
		if err != nil || got != c.want {
			t.Errorf("%s = %#v, %v; want %v", c.expr, got, err, c.want)
		}
	}
}
