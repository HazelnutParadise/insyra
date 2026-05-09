package ccl

import (
	"math"
	"strings"
	"testing"
	"time"
)

// Each test calls RegisterStandardFunctions to ensure the new helpers are
// wired in alongside the existing ones. Registration is idempotent because
// it overwrites map entries.
func init() {
	RegisterStandardFunctions()
}

// callFn invokes a registered scalar function by name and returns the result.
// Wraps callFunction to reset funcCallDepth between tests.
func callFn(t *testing.T, name string, args ...any) (any, error) {
	t.Helper()
	ResetFuncCallDepth()
	return callFunction(name, args)
}

func callAgg(t *testing.T, name string, cols ...[]any) (any, error) {
	t.Helper()
	return callAggregateFunction(name, cols)
}

func wantFloat(t *testing.T, got any, want float64) {
	t.Helper()
	f, ok := got.(float64)
	if !ok {
		t.Fatalf("expected float64, got %T (%v)", got, got)
	}
	if math.IsNaN(want) {
		if !math.IsNaN(f) {
			t.Fatalf("expected NaN, got %v", f)
		}
		return
	}
	const eps = 1e-9
	if math.Abs(f-want) > eps {
		t.Fatalf("expected %v, got %v", want, f)
	}
}

func wantString(t *testing.T, got any, want string) {
	t.Helper()
	s, ok := got.(string)
	if !ok {
		t.Fatalf("expected string, got %T (%v)", got, got)
	}
	if s != want {
		t.Fatalf("expected %q, got %q", want, s)
	}
}

func wantBool(t *testing.T, got any, want bool) {
	t.Helper()
	b, ok := got.(bool)
	if !ok {
		t.Fatalf("expected bool, got %T (%v)", got, got)
	}
	if b != want {
		t.Fatalf("expected %v, got %v", want, b)
	}
}

// ---------------------------------------------------------------- Math ----

func TestMathFunctions(t *testing.T) {
	cases := []struct {
		name string
		args []any
		want float64
	}{
		{"ABS", []any{-3.5}, 3.5},
		{"ABS", []any{2.0}, 2.0},
		{"ROUND", []any{3.14159, 2.0}, 3.14},
		{"ROUND", []any{2.5}, 3.0},
		{"ROUND", []any{-1.6}, -2.0},
		{"FLOOR", []any{3.7}, 3.0},
		{"FLOOR", []any{-1.2}, -2.0},
		{"CEIL", []any{3.2}, 4.0},
		{"CEIL", []any{-1.8}, -1.0},
		{"TRUNC", []any{3.9}, 3.0},
		{"TRUNC", []any{-3.9}, -3.0},
		{"MOD", []any{10.0, 3.0}, 1.0},
		{"POW", []any{2.0, 10.0}, 1024.0},
		{"SQRT", []any{16.0}, 4.0},
		{"LN", []any{math.E}, 1.0},
		{"LOG", []any{1000.0}, 3.0},        // base-10 default
		{"LOG", []any{8.0, 2.0}, 3.0},      // explicit base
		{"LOG10", []any{100.0}, 2.0},
		{"EXP", []any{0.0}, 1.0},
		{"SIGN", []any{-5.0}, -1.0},
		{"SIGN", []any{0.0}, 0.0},
		{"SIGN", []any{42.0}, 1.0},
	}
	for _, c := range cases {
		got, err := callFn(t, c.name, c.args...)
		if err != nil {
			t.Fatalf("%s%v: unexpected error: %v", c.name, c.args, err)
		}
		wantFloat(t, got, c.want)
	}
}

func TestMathErrors(t *testing.T) {
	cases := []struct {
		name string
		args []any
		msg  string
	}{
		{"ABS", []any{}, "ABS requires"},
		{"ABS", []any{"oops"}, "cannot convert"},
		{"SQRT", []any{-1.0}, "negative"},
		{"LN", []any{0.0}, "non-positive"},
		{"LOG", []any{-1.0}, "non-positive"},
		{"LOG", []any{10.0, 1.0}, "invalid base"},
		{"MOD", []any{1.0, 0.0}, "division by zero"},
	}
	for _, c := range cases {
		_, err := callFn(t, c.name, c.args...)
		if err == nil {
			t.Fatalf("%s%v: expected error containing %q, got nil", c.name, c.args, c.msg)
		}
		if !strings.Contains(err.Error(), c.msg) {
			t.Fatalf("%s%v: expected error containing %q, got %v", c.name, c.args, c.msg, err)
		}
	}
}

// -------------------------------------------------------------- String ----

func TestStringFunctions(t *testing.T) {
	got, err := callFn(t, "LEN", "héllo")
	if err != nil {
		t.Fatal(err)
	}
	wantFloat(t, got, 5)

	got, _ = callFn(t, "UPPER", "hello")
	wantString(t, got, "HELLO")

	got, _ = callFn(t, "LOWER", "WoRLD")
	wantString(t, got, "world")

	got, _ = callFn(t, "TRIM", "  hi  ")
	wantString(t, got, "hi")

	got, _ = callFn(t, "LTRIM", "  hi  ")
	wantString(t, got, "hi  ")

	got, _ = callFn(t, "RTRIM", "  hi  ")
	wantString(t, got, "  hi")

	got, _ = callFn(t, "LEFT", "abcdef", 3.0)
	wantString(t, got, "abc")

	got, _ = callFn(t, "RIGHT", "abcdef", 2.0)
	wantString(t, got, "ef")

	got, _ = callFn(t, "MID", "abcdef", 2.0, 3.0) // 1-based start
	wantString(t, got, "bcd")

	got, _ = callFn(t, "SUBSTR", "abcdef", 2.0, 3.0)
	wantString(t, got, "bcd")

	got, _ = callFn(t, "REPLACE", "foo bar foo", "foo", "baz")
	wantString(t, got, "baz bar baz")

	got, _ = callFn(t, "FIND", "bar", "foo bar baz")
	wantFloat(t, got, 5) // 1-based, "bar" starts at position 5

	got, _ = callFn(t, "FIND", "qux", "foo bar baz")
	wantFloat(t, got, 0) // not found -> 0

	got, _ = callFn(t, "CONTAINS", "hello world", "world")
	wantBool(t, got, true)

	got, _ = callFn(t, "STARTSWITH", "hello world", "hello")
	wantBool(t, got, true)

	got, _ = callFn(t, "ENDSWITH", "hello world", "world")
	wantBool(t, got, true)

	got, _ = callFn(t, "REGEX_MATCH", "abc123", `\d+`)
	wantBool(t, got, true)

	got, _ = callFn(t, "REGEX_MATCH", "abc", `\d+`)
	wantBool(t, got, false)

	got, _ = callFn(t, "REPEAT", "ab", 3.0)
	wantString(t, got, "ababab")
}

func TestStringEdgeCases(t *testing.T) {
	// nil input -> empty string semantics
	got, err := callFn(t, "LEN", nil)
	if err != nil {
		t.Fatalf("LEN(nil) errored: %v", err)
	}
	wantFloat(t, got, 0)

	// Slicing past the end clamps cleanly.
	got, _ = callFn(t, "LEFT", "ab", 10.0)
	wantString(t, got, "ab")

	got, _ = callFn(t, "RIGHT", "ab", 10.0)
	wantString(t, got, "ab")

	got, _ = callFn(t, "MID", "abcdef", 100.0, 5.0)
	wantString(t, got, "")

	// Negative count for REPEAT errors out.
	if _, err := callFn(t, "REPEAT", "x", -1.0); err == nil {
		t.Fatal("REPEAT with negative count should error")
	}

	// Bad regex pattern errors.
	if _, err := callFn(t, "REGEX_MATCH", "abc", "(["); err == nil {
		t.Fatal("REGEX_MATCH with bad pattern should error")
	}
}

// ---------------------------------------------------- Type conversion ----

func TestTypeConversionFunctions(t *testing.T) {
	got, _ := callFn(t, "TONUM", "42.5")
	wantFloat(t, got, 42.5)

	got, _ = callFn(t, "TONUM", "not a number")
	if got != nil {
		t.Fatalf("TONUM('not a number') should return nil, got %v", got)
	}

	got, _ = callFn(t, "VALUE", "  17  ")
	wantFloat(t, got, 17)

	got, _ = callFn(t, "TOSTR", 3.14)
	wantString(t, got, "3.14")

	got, _ = callFn(t, "TOSTR", 3.14159, "%.2f")
	wantString(t, got, "3.14")

	got, _ = callFn(t, "TEXT", 42, "%05d")
	wantString(t, got, "00042")

	got, _ = callFn(t, "TOBOOL", "yes")
	wantBool(t, got, true)

	got, _ = callFn(t, "TOBOOL", "no")
	wantBool(t, got, false)

	// COALESCE
	got, _ = callFn(t, "COALESCE", nil, nil, "first", "second")
	wantString(t, got, "first")

	// COALESCE skips NaN
	got, _ = callFn(t, "COALESCE", math.NaN(), nil, 7.0)
	wantFloat(t, got, 7)

	// COALESCE all nil -> nil
	got, _ = callFn(t, "COALESCE", nil, nil)
	if got != nil {
		t.Fatalf("COALESCE(nil, nil) should return nil, got %v", got)
	}

	// IFNULL
	got, _ = callFn(t, "IFNULL", nil, "fallback")
	wantString(t, got, "fallback")

	got, _ = callFn(t, "IFNULL", "value", "fallback")
	wantString(t, got, "value")
}

// ----------------------------------------------------------- DateTime ----

func TestDateTimeFunctions(t *testing.T) {
	d := time.Date(2024, 7, 15, 10, 30, 0, 0, time.UTC) // Monday

	got, _ := callFn(t, "YEAR", d)
	wantFloat(t, got, 2024)

	got, _ = callFn(t, "MONTH", d)
	wantFloat(t, got, 7)

	got, _ = callFn(t, "DAYOFMONTH", d)
	wantFloat(t, got, 15)

	got, _ = callFn(t, "WEEKDAY", d)
	wantFloat(t, got, float64(time.Monday)) // 1

	// String input gets parsed automatically.
	got, _ = callFn(t, "YEAR", "2024-07-15")
	wantFloat(t, got, 2024)

	// DATEDIFF: d1 - d2 in chosen unit.
	d1 := time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC)
	d2 := time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC)
	got, _ = callFn(t, "DATEDIFF", d1, d2, "day")
	wantFloat(t, got, 5)

	got, _ = callFn(t, "DATEDIFF", d1, d2, "hour")
	wantFloat(t, got, 120)

	// DATEADD: shift forward.
	got, err := callFn(t, "DATEADD", d, 10.0, "day")
	if err != nil {
		t.Fatalf("DATEADD error: %v", err)
	}
	added, ok := got.(time.Time)
	if !ok {
		t.Fatalf("DATEADD: expected time.Time, got %T", got)
	}
	if added.Day() != 25 {
		t.Fatalf("DATEADD +10 days: expected day 25, got %v", added)
	}

	got, err = callFn(t, "DATEADD", d, 1.0, "month")
	if err != nil {
		t.Fatalf("DATEADD error: %v", err)
	}
	added = got.(time.Time)
	if int(added.Month()) != 8 {
		t.Fatalf("DATEADD +1 month: expected August, got %v", added.Month())
	}

	// FORMAT_DATE
	got, _ = callFn(t, "FORMAT_DATE", d, "2006-01-02")
	wantString(t, got, "2024-07-15")
}

func TestDateTimeErrors(t *testing.T) {
	if _, err := callFn(t, "YEAR", "not-a-date"); err == nil {
		t.Fatal("YEAR with bad string should error")
	}
	if _, err := callFn(t, "DATEDIFF", time.Now(), time.Now(), "fortnights"); err == nil {
		t.Fatal("DATEDIFF with unknown unit should error")
	}
	if _, err := callFn(t, "DATEADD", time.Now(), 1.0, "fortnights"); err == nil {
		t.Fatal("DATEADD with unknown unit should error")
	}
}

// --------------------------------------------------- Statistical aggs ----

func TestAggregateStatFunctions(t *testing.T) {
	// Single column.
	col := []any{1.0, 2.0, 3.0, 4.0, 5.0}

	got, _ := callAgg(t, "MEDIAN", col)
	wantFloat(t, got, 3)

	got, _ = callAgg(t, "MEDIAN", []any{1.0, 2.0, 3.0, 4.0})
	wantFloat(t, got, 2.5)

	// Sample variance of {1..5} = 2.5; population variance = 2.0
	got, _ = callAgg(t, "VAR", col)
	wantFloat(t, got, 2.5)

	got, _ = callAgg(t, "VARP", col)
	wantFloat(t, got, 2.0)

	got, _ = callAgg(t, "STDEV", col)
	wantFloat(t, got, math.Sqrt(2.5))

	got, _ = callAgg(t, "STDEVP", col)
	wantFloat(t, got, math.Sqrt(2.0))

	// nil values are skipped, mirroring SUM/AVG semantics.
	got, _ = callAgg(t, "MEDIAN", []any{1.0, nil, 2.0, nil, 3.0})
	wantFloat(t, got, 2)

	// Empty input -> nil.
	if got, _ := callAgg(t, "MEDIAN", []any{}); got != nil {
		t.Fatalf("MEDIAN on empty -> nil expected, got %v", got)
	}

	// VAR requires >=2 values.
	if _, err := callAgg(t, "VAR", []any{1.0}); err == nil {
		t.Fatal("VAR with 1 value should error")
	}

	// VARP works with single value.
	got, _ = callAgg(t, "VARP", []any{5.0})
	wantFloat(t, got, 0)
}
