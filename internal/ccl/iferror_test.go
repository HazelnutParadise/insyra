package ccl

import (
	"strings"
	"testing"
)

func TestIfErrorUsesTheFallbackOnlyOnAnError(t *testing.T) {
	ctx := mapCtx(t, map[string][]any{"A": {6.0}, "B": {0.0}, "C": {"word"}, "D": {nil}, "E": {2.0}})
	for _, c := range []struct {
		expr string
		want any
	}{
		{"IFERROR(A / B, 9)", 9.0},
		{"IFERROR(A / E, 9)", 3.0},
		{"IFERROR(C + 1, 'bad')", "bad"},
		{"IFERROR(SQRT(-1), 0)", 0.0},
		{"IFERROR(LOG(0), 0)", 0.0},
		{"IFERROR(DATEDIFF('x', 'y', 'day'), 0)", 0.0},
		{"IFERROR(A.9, 7)", 7.0},
		{"IFERROR(D, 9)", nil},
		{"IFERROR(A, 1 / 0)", 6.0},
		{"IFERROR(1 / 0, IFERROR(A / B, 5))", 5.0},
		{"iferror(A / B, 9)", 9.0},
		{"IFERROR(A / B, 9) + 1", 10.0},
		{"IF(IFERROR(A / B, 0) == 0, 'zero', 'other')", "zero"},
	} {
		got, err := evalOne(t, ctx, c.expr)
		if err != nil {
			t.Errorf("%s: %v", c.expr, err)
			continue
		}
		if want, isNumber := c.want.(float64); isNumber {
			if f, ok := toFloat64(got); !ok || f != want {
				t.Errorf("%s = %#v, want %v", c.expr, got, want)
			}
			continue
		}
		if got != c.want {
			t.Errorf("%s = %#v, want %#v", c.expr, got, c.want)
		}
	}
	for _, c := range []struct{ expr, want string }{
		{"IFERROR(A)", "requires 2 arguments"},
		{"IFERROR(A, 1, 2)", "requires 2 arguments"},
		{"IFERROR(1 / 0, 1 / 0)", "division by zero"},
		// A misspelt name is a mistake in the formula, not in the data.
		{"IFERROR(NOSUCHFN(A), 0)", "undefined function"},
		{"IFERROR(['nope'] + 1, 0)", "nope"},
		{"IFERROR(1 + NOSUCHFN(A), 0)", "undefined function"},
	} {
		got, err := evalOne(t, ctx, c.expr)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s = %#v, %v; want an error with %q", c.expr, got, err, c.want)
		}
	}
}
