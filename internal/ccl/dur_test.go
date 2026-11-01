package ccl

import (
	"math"
	"strings"
	"testing"
	"time"
)

// DUR expresses a duration in a unit, which DAY, HOUR, MINUTE and SECOND used
// to do before they took a part of a date. It also reads a duration string,
// which nothing else in CCL does.
func TestDUR(t *testing.T) {
	ctx := mapCtx(t, map[string][]any{
		"A": {time.Date(2024, 1, 3, 12, 0, 0, 0, time.UTC)},
		"B": {time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)},
		"C": {36 * time.Hour},
		"D": {"1h30m"},
	})
	for _, c := range []struct {
		expr string
		want float64
	}{
		{"DUR('36h', 'day')", 1.5},
		{"DUR('36h', 'days')", 1.5},
		{"DUR('36h', 'DAY')", 1.5},
		{"DUR('1h30m', 'minute')", 90},
		{"DUR('90s', 'second')", 90},
		{"DUR('90s', 'minute')", 1.5},
		{"DUR('2d', 'hour')", 48},
		{"DUR('1.5d', 'hour')", 36},
		{"DUR('1w', 'day')", 7},
		{"DUR('1w2d', 'day')", 9},
		{"DUR('1d12h', 'hour')", 36},
		{"DUR('-2d', 'hour')", -48},
		{"DUR('500ms', 'second')", 0.5},
		{"DUR(C, 'hour')", 36},
		{"DUR(D, 'minute')", 90},
		{"DUR(A - B, 'hour')", 36},
		{"DUR(A - B, 'day') - DATEDIFF(A, B, 'day')", 0},
	} {
		got, err := evalOne(t, ctx, c.expr)
		if err != nil {
			t.Errorf("%s: %v", c.expr, err)
			continue
		}
		f, ok := got.(float64)
		if !ok || math.Abs(f-c.want) > 1e-9 {
			t.Errorf("%s = %#v (%T), want %v", c.expr, got, got, c.want)
		}
	}
}

// One empty cell gives an empty result rather than failing the whole column.
func TestDURLeavesAnEmptyCellEmpty(t *testing.T) {
	ctx := mapCtx(t, map[string][]any{"A": {nil}})
	got, err := evalOne(t, ctx, "DUR(A, 'day')")
	if err != nil || got != nil {
		t.Errorf("DUR(nil) = %#v, %v; want nil and no error", got, err)
	}
}

func TestDURRefusesWhatItCannotRead(t *testing.T) {
	ctx := mapCtx(t, map[string][]any{"A": {nil}})
	for _, c := range []struct{ expr, want string }{
		{"DUR(36, 'hour')", "number"},
		{"DUR('soon', 'hour')", "soon"},
		{"DUR('', 'hour')", "DUR"},
		{"DUR('2x', 'hour')", "2x"},
		{"DUR(true, 'hour')", "DUR"},
		{"DUR('36h', 'month')", "month"},
		{"DUR('36h', 1)", "unit"},
		{"DUR('36h')", "2 arguments"},
		{"DUR('999999999w', 'day')", "999999999w"},
		{"DUR('2024-01-02', 'day')", "2024-01-02"},
	} {
		got, err := evalOne(t, ctx, c.expr)
		if err == nil {
			t.Errorf("%s = %#v, want an error", c.expr, got)
			continue
		}
		if !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: the error %q does not mention %q", c.expr, err, c.want)
		}
	}
}

// DAY used to say an empty cell was a number.
func TestDatePartFunctionsSayACellIsEmpty(t *testing.T) {
	ctx := mapCtx(t, map[string][]any{"A": {nil}})
	for _, name := range []string{"DAY", "HOUR", "MINUTE", "SECOND"} {
		_, err := evalOne(t, ctx, name+"(A)")
		if err == nil {
			t.Errorf("%s(nil) gave no error", name)
			continue
		}
		if strings.Contains(err.Error(), "number") || !strings.Contains(err.Error(), "empty") {
			t.Errorf("%s(nil): the error %q should say the value is empty", name, err)
		}
	}
}

// The error for a duration string now points to something that reads one.
func TestDatePartFunctionsPointToDUR(t *testing.T) {
	ctx := mapCtx(t, map[string][]any{"A": {nil}})
	_, err := evalOne(t, ctx, "DAY('36h')")
	if err == nil || !strings.Contains(err.Error(), "DUR(x, 'day')") {
		t.Errorf("DAY('36h'): the error %v does not point to DUR", err)
	}
}
