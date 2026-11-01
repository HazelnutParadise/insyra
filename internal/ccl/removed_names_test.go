package ccl

import (
	"strings"
	"testing"
)

// One function, one name: the shorter name stays, and a formula using the other
// is told what to write.
func TestRemovedNamesSayWhatToUse(t *testing.T) {
	ctx := mapCtx(t, map[string][]any{"A": {"2024-03-09"}})
	for _, c := range []struct{ expr, want string }{
		{"TONUM(A)", "use VALUE"},
		{"TOSTR(A)", "use TEXT"},
		{"TOSTR(A, '%s')", "use TEXT"},
		{"SUBSTR(A, 1, 2)", "use MID"},
		{"DAYOFMONTH(A)", "use DAY"},
	} {
		got, err := evalOne(t, ctx, c.expr)
		if err == nil {
			t.Errorf("%s = %#v, want an error", c.expr, got)
			continue
		}
		if !strings.Contains(err.Error(), "removed") || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: the error %q does not say it was removed and %s", c.expr, err, c.want)
		}
	}
}
