package ccl

import (
	"fmt"
	"math"
	"strings"
	"testing"
	"time"
)

// Go leaves int(f) and time.Duration(f) undefined when f is NaN, infinite or
// past the integer range, and the platforms disagree: amd64 turns int(1e300)
// into the most negative int, arm64 saturates to the most positive one. So
// MID('abc', 2, 10^300) gave "" on the Linux and Windows CI runners and "bc"
// on a Mac. Run these tests with GOARCH=amd64 as well as natively.

// portableCtx holds numbers in A, dates in B and NaN in C. MapContext assigns
// Excel letters by sorted name, so the names are the letters.
func portableCtx(t *testing.T) *MapContext {
	jan1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	return mapCtx(t, map[string][]any{
		"A": {1.0, 2.0, 3.0},
		"B": {jan1, jan1, jan1},
		"C": {math.NaN(), math.NaN(), math.NaN()},
	})
}

func TestIntegerArgumentsAgreeAcrossPlatforms(t *testing.T) {
	ctx := portableCtx(t)
	for _, c := range []struct {
		expr    string
		want    any // compared with fmt.Sprint; ignored when wantErr
		wantErr bool
	}{
		// A count or position past the end of the string means "to the end".
		{"LEFT('abc', 10^300)", "abc", false},
		{"RIGHT('abc', 10^300)", "abc", false},
		{"MID('abc', 2, 10^300)", "bc", false},
		{"MID('abc', 10^300, 1)", "", false},
		{"MID('abc', 0-10^300, 2)", "ab", false}, // as MID('abc', 0, 2)
		// NaN is not a count or a digit count.
		{"LEFT('abc', C)", nil, true},
		{"RIGHT('abc', C)", nil, true},
		{"MID('abc', C, 1)", nil, true},
		{"ROUND(2.567, C)", nil, true},
		// A date shift or a duration nothing can hold is refused.
		{"DATEADD(B, 10^300, 'day')", nil, true},
		{"DATEADD(B, 10^300, 'month')", nil, true},
		{"DATEADD(B, 0-10^300, 'year')", nil, true},
		{"DATEADD(B, C, 'day')", nil, true},
		{"DATEADD(B, 10^300, 'hour')", nil, true},
		{"DATEADD(B, 10^300, 'minute')", nil, true},
		{"DATEADD(B, C, 'second')", nil, true},
		{"HOUR(10^300)", nil, true},
		{"DAY(10^300)", nil, true},
		{"MINUTE(10^300)", nil, true},
		{"SECOND(0-10^300)", nil, true},
		{"B + 10^300", nil, true},
		{"B - 10^300", nil, true},
		{"10^300 + B", nil, true},
		{"B + C", nil, true},
		// One day past the largest shift a Duration can hold.
		{"B + 106752", nil, true},
		{"B - 106752", nil, true},
		// A shift, window or repeat count nothing can hold is refused.
		{"LAG(A, 10^300)", nil, true},
		{"LEAD(A, C)", nil, true},
		{"ROLLING_MEAN(A, 10^300)", nil, true},
		{"REPEAT('ab', 10^300)", nil, true},
		{"REPEAT('', C)", nil, true},
		{"REPEAT('ab', 0-1)", nil, true},
		{"REPEAT('ab', 0-1.5)", nil, true},
		{"REPEAT('', 0-1)", nil, true},
		{"REPEAT('ab', 0-10^300)", nil, true},
		// A fraction that truncates to 0 is still no shift or window.
		{"DIFF(A, 0-0.5)", nil, true},
		{"PCT_CHANGE(A, 0-0.5)", nil, true},
		{"ROLLING_MEAN(A, 0-0.5)", nil, true},
		{"ROLLING_SUM(A, 0.5)", nil, true},
		{"REPEAT('abc', 9223372036854775807)", nil, true}, // the bytes overflow int
		// A row range bound that is NaN or past the int32 range is refused.
		{"SUM(A.(0:10^300))", nil, true},
		{"SUM(A.(0-10^300:1))", nil, true},
		{"SUM(A.(0:C))", nil, true},
	} {
		got, err := evalCol(t, ctx, c.expr)
		switch {
		case c.wantErr && err == nil:
			t.Errorf("%s = %v; want an error", c.expr, got)
		case !c.wantErr && err != nil:
			t.Errorf("%s: unexpected error %v", c.expr, err)
		case !c.wantErr && fmt.Sprint(got[0]) != fmt.Sprint(c.want):
			t.Errorf("%s = %v; want %v", c.expr, got[0], c.want)
		}
	}
}

// The guards only refuse what no platform could compute. Every ordinary value
// gives the result it gave before them, including the conversions that
// truncate: a fractional count, digit count, DATEADD day count or range bound
// drops its fraction. A number of days added to a date keeps its fraction,
// because Docs/CCL.md has always said the number counts days.
func TestIntegerArgumentsOrdinaryValuesUnchanged(t *testing.T) {
	ctx := portableCtx(t)
	jan1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, c := range []struct {
		expr string
		want any // compared with fmt.Sprint
	}{
		{"LEFT('abcdef', 2)", "ab"},
		{"LEFT('abc', 2.9)", "ab"},
		{"LEFT('abc', -1)", ""},
		{"LEFT('abc', 3000000000)", "abc"},
		{"RIGHT('abcdef', 2)", "ef"},
		{"RIGHT('abcdef', 0)", ""},
		{"MID('abcdef', 2, 3)", "bcd"},
		{"MID('abc', 0, 2)", "ab"},
		{"ROUND(2.567, 2)", 2.57},
		{"ROUND(2.567, 1.9)", 2.6},
		{"ROUND(1234.5, 0-2)", 1200.0},
		{"ROUND(2.5)", 3.0},
		{"DATEADD(B, 1, 'day')", jan1.AddDate(0, 0, 1)},
		{"DATEADD(B, 1.9, 'day')", jan1.AddDate(0, 0, 1)},
		{"DATEADD(B, 0-1, 'month')", jan1.AddDate(0, -1, 0)},
		{"DATEADD(B, 2, 'year')", jan1.AddDate(2, 0, 0)},
		{"DATEADD(B, 1.5, 'hour')", jan1.Add(90 * time.Minute)},
		{"DATEADD(B, 90, 'minute')", jan1.Add(90 * time.Minute)},
		{"DATEADD(B, 30, 'second')", jan1.Add(30 * time.Second)},
		{"HOUR(7200)", 2.0},
		{"DAY(172800)", 2.0},
		{"MINUTE(90)", 1.5},
		{"SECOND(1.5)", 1.5},
		{"B + 1", jan1.AddDate(0, 0, 1)},
		{"B - 1", jan1.AddDate(0, 0, -1)},
		{"1 + B", jan1.AddDate(0, 0, 1)},
		{"B + 0.5", jan1.Add(12 * time.Hour)},
		{"B + 0.01", jan1.Add(864 * time.Second)},
		// Nothing under an hour is dropped: v0.3.2 moved these 1 hour and 0.
		{"B + 0.0625", jan1.Add(90 * time.Minute)},
		{"B - 0.0625", jan1.Add(-90 * time.Minute)},
		{"0.0625 + B", jan1.Add(90 * time.Minute)},
		{"B + 0.001", jan1.Add(86400 * time.Millisecond)},
		{"B + 106751", jan1.Add(time.Duration(106751*24) * time.Hour)},
		{"B - 106751", jan1.Add(-time.Duration(106751*24) * time.Hour)},
		// Large but deterministic arguments give what they gave on v0.3.2: a
		// shift or window longer than the column is all nil, and a long result
		// is not capped.
		{"LAG(A, 3000000000)", nil},
		{"LEAD(A, 3000000000)", nil},
		{"LAG(A, 0-9223372036854775808)", nil},
		{"ROLLING_MEAN(A, 3000000000)", nil},
		{"LEN(REPEAT('', 100000000))", 0.0},
		{"LEN(REPEAT('ab', 34000000))", 68000000.0},
		{"DATEADD(B, 3000000000, 'day')", jan1.AddDate(0, 0, 3000000000)},
		{"SUM(A.(0:1))", 3.0},
		{"SUM(A.(0.9:1.9))", 3.0},
		// A fraction between -1 and 0 truncates to 0, as int(n) did on v0.3.2.
		{"REPEAT('ab', 0-0.5)", ""},
		{"REPEAT('ab', 0-0.999)", ""},
		{"LEN(REPEAT('', 0-0.5))", 0.0},
		{"REPEAT('ab', 0.5)", ""},
		{"LEFT('abc', 0-0.5)", ""},
		{"RIGHT('abc', 0-0.5)", ""},
		{"MID('abc', 0-0.5, 2)", "ab"},
		{"MID('abc', 1, 0-0.5)", ""},
		{"LAG(A, 0-0.5)", 1.0},
		{"LEAD(A, 0-0.5)", 1.0},
		{"DATEADD(B, 0-0.5, 'day')", jan1},
		{"DATEADD(B, 0-0.999, 'month')", jan1},
		{"DATEADD(B, 0-0.5, 'year')", jan1},
		{"DATEADD(B, 0-0.5, 'hour')", jan1.Add(-30 * time.Minute)},
	} {
		got, err := evalCol(t, ctx, c.expr)
		switch {
		case err != nil:
			t.Errorf("%s: unexpected error %v", c.expr, err)
		case fmt.Sprint(got[0]) != fmt.Sprint(c.want):
			t.Errorf("%s = %v; want %v", c.expr, got[0], c.want)
		}
	}
}

// The out-of-range message quotes the number the expression was written with,
// not the signed shift the evaluator applies. `B - 106752` subtracts 106,752
// days; saying "a shift of -106752 days" sends the reader looking for a minus
// sign that is not in their expression.
func TestDateShiftErrorQuotesTheOperandAsWritten(t *testing.T) {
	ctx := portableCtx(t)
	for _, c := range []struct {
		expr string
		want string
	}{
		{"B + 106752", "a shift of 106752 days is out of range"},
		{"B - 106752", "a shift of 106752 days is out of range"},
		{"106752 + B", "a shift of 106752 days is out of range"},
		// (0-106752) is a negative operand, so the message keeps its sign.
		{"B + (0-106752)", "a shift of -106752 days is out of range"},
		{"B - (0-106752)", "a shift of -106752 days is out of range"},
		// B + 0-106752 parses as (B + 0) - 106752, a subtraction of 106752.
		{"B + 0-106752", "a shift of 106752 days is out of range"},
	} {
		_, err := evalCol(t, ctx, c.expr)
		if err == nil {
			t.Errorf("%s: want an error", c.expr)
			continue
		}
		if !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: error %q does not contain %q", c.expr, err, c.want)
		}
	}
}
