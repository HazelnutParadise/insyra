package ccl

import (
	"strings"
	"testing"
)

// The cases from Microsoft's pages for INT, ROUNDUP and ROUNDDOWN, then the cases
// a float64 gets wrong if it is multiplied rather than read as written.
func TestRoundFamily(t *testing.T) {
	ctx := mapCtx(t, map[string][]any{"A": {3.14159}})
	for _, c := range []struct {
		expr string
		want float64
	}{
		{"INT(8.9)", 8},
		{"INT(-8.9)", -9},
		{"INT(5)", 5},
		{"ROUNDUP(3.2, 0)", 4},
		{"ROUNDUP(76.9, 0)", 77},
		{"ROUNDUP(3.14159, 3)", 3.142},
		{"ROUNDUP(-3.14159, 1)", -3.2},
		{"ROUNDUP(31415.92654, -2)", 31500},
		{"ROUNDDOWN(3.2, 0)", 3},
		{"ROUNDDOWN(76.9, 0)", 76},
		{"ROUNDDOWN(3.14159, 3)", 3.141},
		{"ROUNDDOWN(-3.14159, 1)", -3.1},
		{"ROUNDDOWN(31415.92654, -2)", 31400},
		{"ROUNDUP(A, 2)", 3.15},
		{"ROUNDDOWN(A, 2)", 3.14},
		{"ROUNDUP(1.1, 1)", 1.1},
		{"ROUNDDOWN(2.3, 1)", 2.3},
		{"ROUNDDOWN(1.005, 2)", 1},
		{"ROUNDUP(1.001, 2)", 1.01},
		{"ROUNDUP(9.991, 2)", 10},
		{"ROUNDUP(5, -2)", 100},
		{"ROUNDUP(0.001, 2)", 0.01},
		{"ROUNDDOWN(5, -2)", 0},
		{"ROUNDUP(0, 2)", 0},
		{"ROUNDDOWN(-0.4, 0)", 0},
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
		{"ROUNDUP(1.5)", "requires 2 arguments"},
		{"ROUNDDOWN(1.5)", "requires 2 arguments"},
		{"ROUNDUP('word', 1)", "ROUNDUP"},
		{"ROUNDDOWN(1.5, 'word')", "ROUNDDOWN"},
		{"INT('word')", "INT"},
		{"INT()", "INT"},
	} {
		got, err := evalOne(t, ctx, c.expr)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s = %#v, %v; want an error with %q", c.expr, got, err, c.want)
		}
	}
}
