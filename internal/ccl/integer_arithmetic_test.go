package ccl

import (
	"math"
	"strings"
	"testing"
)

// CCL-21 (#358): every number went through float64, so an integer column came
// out of A * 1 as float64 and an integer past 2^53 lost its last digits:
// int64(9007199254740993) + 0 was 9007199254740992. Integers now stay int64
// through +, -, *, %, unary minus, SUM, MIN, MAX and MOD, and compare exactly.

const big = int64(1<<53 + 1) // 9007199254740993, the first integer a float64 cannot hold

// evalOne evaluates expr on the first row of ctx.
func evalOne(t *testing.T, ctx *MapContext, expr string) (any, error) {
	t.Helper()
	got, err := evalCol(t, ctx, expr)
	if err != nil {
		return nil, err
	}
	return got[0], nil
}

func TestIntegerLiteralsAreInt64(t *testing.T) {
	ctx := mapCtx(t, map[string][]any{"A": {int64(1)}})
	for _, c := range []struct {
		expr string
		want any
	}{
		{"1", int64(1)},
		{"0", int64(0)},
		{"-5", int64(-5)},
		{"9223372036854775807", int64(math.MaxInt64)},
		{"-9223372036854775808", int64(math.MinInt64)},
		{"9007199254740993", big},
		// Past int64 a whole-number literal is read as a float64, as before.
		{"9223372036854775808", float64(1 << 63)},
		{"1.0", 1.0},
		{"1.5", 1.5},
		{"1e3", 1000.0},
		{".5", 0.5},
		{"1 + 2", int64(3)},
		{"7 - 10", int64(-3)},
		{"6 * 7", int64(42)},
		{"7 % 3", int64(1)},
		{"-7 % 3", int64(-1)},
		{"1 + 2.0", 3.0},
		{"7 / 2", 3.5},
		{"6 / 3", 2.0},
		{"2 ^ 3", 8.0},
	} {
		got, err := evalOne(t, ctx, c.expr)
		if err != nil {
			t.Errorf("%s: %v", c.expr, err)
			continue
		}
		if got != c.want {
			t.Errorf("%s = %#v (%T), want %#v (%T)", c.expr, got, got, c.want, c.want)
		}
	}
}

func TestIntegerColumnsStayExact(t *testing.T) {
	ctx := mapCtx(t, map[string][]any{
		"A": {big},
		"B": {int64(1 << 53)},
		"C": {int16(3)},
		"D": {uint8(4)},
		"E": {uint64(math.MaxUint64)},
		"F": {uint64(7)},
		"G": {2.5},
		"H": {int32(-6)},
	})
	for _, c := range []struct {
		expr string
		want any
	}{
		{"A", big},
		{"A + 0", big},
		{"A * 1", big},
		{"A - 1", int64(1 << 53)},
		{"A + A", 2 * big},
		{"A % 10", int64(3)},
		{"-A", -big},
		{"0 - A", -big},
		{"C + D", int64(7)},
		{"C * H", int64(-18)},
		{"F + 1", int64(8)},
		{"A == B", false},
		{"A != B", true},
		{"A > B", true},
		{"B < A", true},
		{"A >= A", true},
		{"A == 9007199254740993", true},
		{"A == 9007199254740992", false},
		{"B < A < A + 1", true},
		// Anything that is not two integers goes through float64, as before.
		{"A / 1", float64(big)},
		{"A ^ 1", float64(big)},
		{"A + 0.0", float64(big)},
		{"C + G", 5.5},
		{"E + 0", float64(math.MaxUint64)},
		{"'3' + C", 6.0},
		{"true + C", 4.0},
		{"C == 3.0", true},
		// nil counts as 0 in arithmetic; next to an integer it is an integer 0.
		{"nil + C", int64(3)},
		{"C - nil", int64(3)},
		{"nil * 2", int64(0)},
		{"nil + G", 2.5},
		{"nil + nil", 0.0},
		{"nil == 0", false},
		{"nil < C", false},
	} {
		got, err := evalOne(t, ctx, c.expr)
		if err != nil {
			t.Errorf("%s: %v", c.expr, err)
			continue
		}
		if got != c.want {
			t.Errorf("%s = %#v (%T), want %#v (%T)", c.expr, got, got, c.want, c.want)
		}
	}
}

// An integer result that int64 cannot hold is an error, never a wrapped
// number and never a float64 that quietly lost digits.
func TestIntegerOverflowIsAnError(t *testing.T) {
	ctx := mapCtx(t, map[string][]any{
		"A": {int64(math.MaxInt64)},
		"B": {int64(math.MinInt64)},
		"C": {int64(0)},
	})
	for _, expr := range []string{
		"A + 1",
		"1 + A",
		"B - 1",
		"A - -1",
		"A * 2",
		"B * -1",
		"-1 * B",
		"-B",
		"0 - B",
		"A + A",
		"SUM(A, 1)",
		"SUM(B, -1)",
	} {
		got, err := evalOne(t, ctx, expr)
		if err == nil {
			t.Errorf("%s = %#v, want an overflow error", expr, got)
			continue
		}
		if !strings.Contains(err.Error(), "overflow") {
			t.Errorf("%s: the error %q does not say the result overflows", expr, err)
		}
	}
	for _, expr := range []string{"A % C", "MOD(A, C)"} {
		if got, err := evalOne(t, ctx, expr); err == nil {
			t.Errorf("%s = %#v, want a modulo-by-zero error", expr, got)
		}
	}
	// The edges themselves are fine.
	for _, c := range []struct {
		expr string
		want int64
	}{
		{"A - 1 + 1", math.MaxInt64},
		{"B + 1 - 1", math.MinInt64},
		{"B % -1", 0},
		{"A * -1", -math.MaxInt64},
	} {
		got, err := evalOne(t, ctx, c.expr)
		if err != nil {
			t.Errorf("%s: %v", c.expr, err)
			continue
		}
		if got != c.want {
			t.Errorf("%s = %#v (%T), want int64 %d", c.expr, got, got, c.want)
		}
	}
}

func TestRowIndexIsInt64(t *testing.T) {
	ctx := mapCtx(t, map[string][]any{"A": {10.0, 20.0, 30.0}})
	got, err := evalCol(t, ctx, "#")
	if err != nil {
		t.Fatal(err)
	}
	for i, v := range got {
		if v != int64(i) {
			t.Errorf("# on row %d = %#v (%T), want int64 %d", i, v, v, i)
		}
	}
	got, err = evalCol(t, ctx, "IF(# > 0, A.(# - 1), nil)")
	if err != nil {
		t.Fatal(err)
	}
	if got[0] != nil || got[1] != 10.0 || got[2] != 20.0 {
		t.Errorf("A.(# - 1) = %#v", got)
	}
}

func TestSumMinMaxModKeepIntegers(t *testing.T) {
	ctx := mapCtx(t, map[string][]any{
		"A": {big, int64(1 << 53), nil, int64(2)},
		"B": {int64(1), 2.5, nil, int64(3)},
		"C": {int64(5), math.NaN(), nil, int64(7)},
		"D": {"1", "2", nil, "3"},
		"E": {nil, nil, nil, nil},
		"F": {int64(math.MaxInt64), int64(1), nil, 0.5},
	})
	for _, c := range []struct {
		expr string
		want any
	}{
		{"SUM(A)", int64(1<<54 + 3)},
		{"MAX(A)", big},
		{"MIN(A)", int64(2)},
		{"SUM(B)", 6.5},
		{"MAX(B)", 3.0},
		{"MIN(B)", 1.0},
		// NaN is skipped, as aggregates always skipped it, so it does not
		// make the values mixed.
		{"SUM(C)", int64(12)},
		{"MAX(C)", int64(7)},
		{"MIN(C)", int64(5)},
		{"SUM(D)", 6.0},
		{"MAX(D)", 3.0},
		{"SUM(E)", 0.0},
		{"MAX(E)", nil},
		{"SUM(1, 2, 3)", int64(6)},
		{"SUM(C, B)", 18.5},
		// An int64 sum that overflows is no error when a float64 joins it,
		// because the result is a float64 anyway.
		{"SUM(F)", float64(1 << 63)},
		{"MOD(7, 3)", int64(1)},
		{"MOD(-7, 3)", int64(-1)},
		{"MOD(A, 10)", int64(3)},
		{"MOD(7.5, 2)", 1.5},
		{"MOD(nil, 3)", int64(0)},
		{"nil % 3", int64(0)},
		{"COUNT(A)", 3.0},
		{"AVG(C)", 6.0},
	} {
		got, err := evalOne(t, ctx, c.expr)
		if err != nil {
			t.Errorf("%s: %v", c.expr, err)
			continue
		}
		if got != c.want {
			t.Errorf("%s = %#v (%T), want %#v (%T)", c.expr, got, got, c.want, c.want)
		}
	}
}

// The streaming aggregates parquet uses answer exactly what the functions
// answer, type included, fed a value at a time.
func TestStreamingAggregatesKeepIntegers(t *testing.T) {
	cols := [][]any{
		{big, int64(1 << 53), nil, int64(2)},
		{int64(1), 2.5, nil, int64(3)},
		{int64(5), math.NaN(), nil, int64(7)},
		{int64(math.MaxInt64), int64(1), nil, 0.5},
		{int64(math.MaxInt64), int64(1)},
		{"1", "2"},
		{},
	}
	for _, name := range []string{"SUM", "MIN", "MAX"} {
		for _, col := range cols {
			want, wantErr := callAggregateFunction(name, [][]any{col})
			s, ok := NewStreamingAggregate(name)
			if !ok {
				t.Fatalf("no streaming %s", name)
			}
			for _, v := range col {
				s.Add([]any{v})
			}
			got, gotErr := s.Result()
			if (wantErr == nil) != (gotErr == nil) {
				t.Errorf("%s(%v): function error %v, streaming error %v", name, col, wantErr, gotErr)
				continue
			}
			if got != want {
				t.Errorf("%s(%v): streaming %#v (%T), function %#v (%T)", name, col, got, got, want, want)
			}
		}
	}
}

// An integer under a float verb is formatted as the float64 it equals; under
// an integer verb it is formatted as itself.
func TestTOSTRFormatsIntegers(t *testing.T) {
	ctx := mapCtx(t, map[string][]any{"A": {int64(42)}})
	for _, c := range []struct {
		expr string
		want string
	}{
		{"TOSTR(50, '%.1f')", "50.0"},
		{"TOSTR(A, '%.2f')", "42.00"},
		{"TOSTR(A, '%d items')", "42 items"},
		{"TOSTR(A * 2, '%05d')", "00084"},
		{"TOSTR(A)", "42"},
	} {
		got, err := evalOne(t, ctx, c.expr)
		if err != nil {
			t.Errorf("%s: %v", c.expr, err)
			continue
		}
		if got != c.want {
			t.Errorf("%s = %q, want %q", c.expr, got, c.want)
		}
	}
	if got, err := evalOne(t, ctx, "TOSTR(A, '%s')"); err == nil {
		t.Errorf("TOSTR(A, '%%s') = %q, want an error", got)
	}
}
