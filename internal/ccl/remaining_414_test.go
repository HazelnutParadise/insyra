package ccl

import (
	"strings"
	"testing"
)

// The multiple cases come from Microsoft's CEILING and FLOOR pages, then the cases
// a float64 gets wrong if it is divided rather than read as written.
func TestCeilFloorMultiple(t *testing.T) {
	ctx := mapCtx(t, map[string][]any{"A": {12.0}})
	for _, c := range []struct {
		expr string
		want float64
	}{
		{"CEIL(2.5, 1)", 3},
		{"CEIL(-2.5, 2)", -2},
		{"CEIL(1.5, 0.1)", 1.5},
		{"CEIL(0.234, 0.01)", 0.24},
		{"FLOOR(3.7, 2)", 2},
		{"FLOOR(1.58, 0.1)", 1.5},
		{"FLOOR(0.234, 0.01)", 0.23},
		{"FLOOR(-2.5, 2)", -4},
		{"CEIL(A, 5)", 15},
		{"FLOOR(A, 5)", 10},
		{"CEIL(A, 4)", 12},
		{"FLOOR(A, 4)", 12},
		{"CEIL(1.1, 0.1)", 1.1},
		{"FLOOR(0.3, 0.1)", 0.3},
		{"CEIL(-1, 5)", 0},
		{"FLOOR(0, 5)", 0},
		{"CEIL(2.1)", 3},
		{"FLOOR(2.9)", 2},
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
	for _, c := range []struct{ expr, want string }{
		{"CEIL(1, 0)", "positive"},
		{"CEIL(1, -2)", "positive"},
		{"FLOOR(2.5, -2)", "positive"},
		{"FLOOR(1, 'x')", "positive"},
		{"CEIL(1, 2, 3)", "CEIL requires 1 or 2 arguments"},
		{"FLOOR()", "FLOOR requires 1 or 2 arguments"},
		{"CEIL('x', 2)", "CEIL"},
	} {
		got, err := evalOne(t, ctx, c.expr)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s = %#v, %v; want an error with %q", c.expr, got, err, c.want)
		}
	}
}

// A name from Excel that CCL spells shorter says which name to write, and stays an
// error, including under IFERROR.
func TestLongNamesPointToTheShortOnes(t *testing.T) {
	ctx := mapCtx(t, map[string][]any{"A": {1.0}})
	for _, c := range []struct{ expr, want string }{
		{"AVERAGE(A)", "use AVG"},
		{"AVERAGEIF(A > 0, A)", "use AVGIF"},
		{"POWER(A, 2)", "use POW"},
		{"CEILING(A)", "use CEIL"},
		{"REPT('a', 2)", "use REPEAT"},
		{"ISNUMBER(A)", "use ISNUM"},
		{"ISLOGICAL(A)", "use ISBOOL"},
		{"CONCATENATE('a', 'b')", "use CONCAT"},
		{"SUBSTITUTE('a', 'a', 'b')", "use REPLACE"},
		{"average(A)", "use AVG"},
		{"IFERROR(AVERAGE(A), 0)", "use AVG"},
		{"NOSUCHFN(A)", "undefined function: NOSUCHFN"},
	} {
		got, err := evalOne(t, ctx, c.expr)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s = %#v, %v; want an error with %q", c.expr, got, err, c.want)
		}
	}
}

func TestRegexTest(t *testing.T) {
	ctx := mapCtx(t, map[string][]any{"A": {"Apple pie"}})
	for _, c := range []struct {
		expr string
		want bool
	}{
		{"REGEXTEST(A, '^Apple')", true},
		{"REGEXTEST(A, '^apple')", false},
		{"REGEXTEST(A, '(?i)^apple')", true},
		{"REGEXTEST(A, 'pie$')", true},
	} {
		got, err := evalOne(t, ctx, c.expr)
		if err != nil {
			t.Errorf("%s: %v", c.expr, err)
			continue
		}
		if got != c.want {
			t.Errorf("%s = %v, want %v", c.expr, got, c.want)
		}
	}
	if got, err := evalOne(t, ctx, "REGEXTEST(A, '[')"); err == nil {
		t.Errorf("a bad pattern gave %v", got)
	}
}
