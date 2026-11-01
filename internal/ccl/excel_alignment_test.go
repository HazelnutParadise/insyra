package ccl

import (
	"math"
	"testing"
	"time"
)

// WEEKDAY, TRIM and COUNT mean what they mean in Excel (#414).
func TestWeekdayCountsSundayAsOne(t *testing.T) {
	ctx := mapCtx(t, map[string][]any{"A": {nil}})
	for _, c := range []struct {
		expr string
		want float64
	}{
		{"WEEKDAY('2024-03-10')", 1}, // Sunday
		{"WEEKDAY('2024-03-11')", 2}, // Monday
		{"WEEKDAY('2024-03-16')", 7}, // Saturday
	} {
		if got, err := evalOne(t, ctx, c.expr); err != nil || got != c.want {
			t.Errorf("%s = %v, %v; want %v", c.expr, got, err, c.want)
		}
	}
}

func TestTrimCollapsesRunsOfSpaces(t *testing.T) {
	ctx := mapCtx(t, map[string][]any{"A": {nil}})
	for _, c := range []struct{ expr, want string }{
		{"TRIM('  a   b  ')", "a b"},
		{"TRIM('a b')", "a b"},
		{"TRIM('a  b   c')", "a b c"},
		{"TRIM('')", ""},
		{"TRIM(A)", ""},
		{"TRIM('資料   分析')", "資料 分析"},
	} {
		if got, err := evalOne(t, ctx, c.expr); err != nil || got != c.want {
			t.Errorf("%s = %q, %v; want %q", c.expr, got, err, c.want)
		}
	}
	// The ends lose any white space; inside, only the space character collapses.
	inner := mapCtx(t, map[string][]any{"A": {"\t a\t\tb\n"}})
	if got, err := evalOne(t, inner, "TRIM(A)"); err != nil || got != "a\t\tb" {
		t.Errorf("TRIM(tabs) = %q, %v; want %q", got, err, "a\t\tb")
	}
}

func TestCountCountsNumbersAndCountACountsValues(t *testing.T) {
	day := time.Date(2024, 3, 9, 0, 0, 0, 0, time.UTC)
	ctx := mapCtx(t, map[string][]any{
		"A": {1, 2.5, int8(3), "4", "text", true, nil, math.NaN(), day, 2 * time.Hour},
	})
	for _, c := range []struct {
		expr string
		want float64
	}{
		{"COUNT(A)", 5}, // 1, 2.5, int8(3), the date and the duration
		{"COUNTA(A)", 9},
	} {
		if got, err := evalOne(t, ctx, c.expr); err != nil || got != c.want {
			t.Errorf("%s = %v, %v; want %v", c.expr, got, err, c.want)
		}
	}
}

// The streaming form has to give what the whole-column form gives.
func TestStreamingCountMatchesAggregate(t *testing.T) {
	values := []any{1, 2.5, "4", "text", true, nil, math.NaN(), time.Now()}
	for name, want := range map[string]float64{"COUNT": 3, "COUNTA": 7} {
		s, ok := NewStreamingAggregate(name)
		if !ok {
			t.Fatalf("%s has no streaming form", name)
		}
		s.Add(values[:3])
		s.Add(values[3:])
		got, err := s.Result()
		if err != nil || got != want {
			t.Errorf("streaming %s = %v, %v; want %v", name, got, err, want)
		}
	}
}
