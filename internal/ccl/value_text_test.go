package ccl

import "testing"

// VALUE reads a number the way a spreadsheet displays it.
func TestValueReadsDisplayedNumbers(t *testing.T) {
	ctx := mapCtx(t, map[string][]any{"A": {"12.5%"}})
	for _, c := range []struct {
		expr string
		want any
	}{
		{"VALUE('10%')", 0.1},
		{"VALUE('7%')", 0.07},
		{"VALUE('12.3%')", 0.123},
		{"VALUE('-10%')", -0.1},
		{"VALUE('10 %')", 0.1},
		{"VALUE(' 10% ')", 0.1},
		{"VALUE('.5%')", 0.005},
		{"VALUE('1,234')", 1234.0},
		{"VALUE('1,234.5')", 1234.5},
		{"VALUE('-1,234,567.25')", -1234567.25},
		{"VALUE('1,234%')", 12.34},
		{"VALUE('12.5')", 12.5},
		{"VALUE('.5')", 0.5},
		{"VALUE('12')", 12.0},
		{"VALUE('1e3')", 1000.0},
		{"VALUE(A)", 0.125},
		{"VALUE('1,23')", nil},
		{"VALUE('12,345,67')", nil},
		{"VALUE('1234,567')", nil},
		{"VALUE(',123')", nil},
		{"VALUE('1,234,')", nil},
		{"VALUE('%')", nil},
		{"VALUE('10%%')", nil},
		{"VALUE('.%')", nil},
		{"VALUE('Inf%')", nil},
		{"VALUE('NaN%')", nil},
		{"VALUE('1e3%')", nil},
		{"VALUE('abc')", nil},
		{"VALUE('$12')", nil},
	} {
		got, err := evalOne(t, ctx, c.expr)
		if err != nil {
			t.Errorf("%s: %v", c.expr, err)
			continue
		}
		if got != c.want {
			t.Errorf("%s = %#v, want %#v", c.expr, got, c.want)
		}
	}
}
