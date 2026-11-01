package ccl

import (
	"strings"
	"testing"
	"time"
)

func TestDateFunction(t *testing.T) {
	ctx := mapCtx(t, map[string][]any{"A": {2024.0}, "B": {3.0}, "C": {9.0}, "D": {nil}, "E": {"word"}})
	date := func(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }
	for _, c := range []struct {
		expr string
		want any
	}{
		{"DATE(2024, 3, 9)", date(2024, 3, 9)},
		{"DATE(A, B, C)", date(2024, 3, 9)},
		{"DATE(2024, 13, 1)", date(2025, 1, 1)},
		{"DATE(2024, 1, 0)", date(2023, 12, 31)},
		{"DATE(2024, 3, 9.9)", date(2024, 3, 9)},
		{"DATE('2024', '3', '9')", date(2024, 3, 9)},
		{"DATE(D, B, C)", nil},
		{"FORMAT_DATE(DATE(2024, 3, 9), '2006/01/02')", "2024/03/09"},
		{"DATEDIFF(DATE(2024, 3, 9), '2024-03-01', 'day')", 8.0},
		{"YEAR(DATE(2024, 3, 9))", 2024.0},
	} {
		got, err := evalOne(t, ctx, c.expr)
		if err != nil {
			t.Errorf("%s: %v", c.expr, err)
			continue
		}
		if want, ok := c.want.(float64); ok {
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
		{"DATE(2024, 3)", "requires 3 arguments"},
		{"DATE(E, 1, 1)", "year"},
		{"DATE(2024, TRUE, 1)", "month"},
		{"DATE(2024, 1, 1e9)", "day"},
		{"DATE(99999, 1, 1)", "1 to 9999"},
		{"TODAY(1)", "no arguments"},
		{"NOW(1)", "no arguments"},
	} {
		got, err := evalOne(t, ctx, c.expr)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s = %#v, %v; want an error with %q", c.expr, got, err, c.want)
		}
	}
}

func TestClockReadsTheLocalDate(t *testing.T) {
	ctx := mapCtx(t, map[string][]any{"A": {1.0}})
	before := time.Now()
	got, err := evalOne(t, ctx, "HOUR(NOW())")
	if err != nil {
		t.Fatal(err)
	}
	after := time.Now()
	if f, _ := toFloat64(got); int(f) != before.Hour() && int(f) != after.Hour() {
		t.Errorf("HOUR(NOW()) = %v, the local hour is %d", got, before.Hour())
	}
	got, err = evalOne(t, ctx, "DATEDIFF(NOW(), TODAY(), 'day')")
	if err != nil {
		t.Fatal(err)
	}
	if f, _ := toFloat64(got); f < 0 || f >= 1 {
		t.Errorf("NOW is %v days after TODAY", got)
	}
	got, err = evalOne(t, ctx, "HOUR(TODAY())")
	if err != nil {
		t.Fatal(err)
	}
	if f, _ := toFloat64(got); f != 0 {
		t.Errorf("TODAY has hour %v", got)
	}
}

func TestFreezeClockReachesEveryCall(t *testing.T) {
	now := time.Date(2024, 3, 9, 14, 5, 0, 0, time.UTC)
	ctx := mapCtx(t, map[string][]any{"A": {1.0, 2.0}})
	for _, c := range []struct {
		expr string
		want any
	}{
		{"NOW()", now},
		{"TODAY()", time.Date(2024, 3, 9, 0, 0, 0, 0, time.UTC)},
		{"FORMAT_DATE(NOW(), '15:04')", "14:05"},
		{"IF(A > 0, YEAR(NOW()) == 2024, FALSE)", true},
		{"YEAR(NOW()) == 2024 && YEAR(TODAY()) == 2024", true},
		{"2023 < YEAR(TODAY()) < 2025", true},
		{"MAX(YEAR(TODAY())) == 2024", true},
	} {
		node, err := CompileExpression(c.expr)
		if err != nil {
			t.Fatal(err)
		}
		got, err := Evaluate(FreezeClock(node, now), ctx)
		if err != nil {
			t.Errorf("%s: %v", c.expr, err)
			continue
		}
		if got != c.want {
			t.Errorf("%s = %#v, want %#v", c.expr, got, c.want)
		}
	}
}
