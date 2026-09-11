package ccl

import (
	"fmt"
	"math"
	"testing"
	"time"
)

// Go leaves int(f) and time.Duration(f) undefined when f is NaN, infinite or
// past the integer range, and the platforms disagree: amd64 turns int(1e300)
// into the most negative int, arm64 saturates to the most positive one. So
// MID('abc', 2, 10^300) gave "" on the Linux and Windows CI runners and "bc"
// on a Mac. Run this test with GOARCH=amd64 as well as natively.
func TestIntegerArgumentsAgreeAcrossPlatforms(t *testing.T) {
	// MapContext assigns Excel letters by sorted name, so the names are the
	// letters: A holds numbers, B dates, C NaN.
	jan1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	ctx := mapCtx(t, map[string][]any{
		"A": {1.0, 2.0, 3.0},
		"B": {jan1, jan1, jan1},
		"C": {math.NaN(), math.NaN(), math.NaN()},
	})
	day2 := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)
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
		{"ROUND(2.567, C)", nil, true},
		{"ROUND(2.567, 2)", 2.57, false},
		// A date shift or a duration nothing can hold is refused.
		{"DATEADD(B, 1, 'day')", day2, false},
		{"DATEADD(B, 10^300, 'day')", nil, true},
		{"DATEADD(B, 10^300, 'month')", nil, true},
		{"DATEADD(B, 10^300, 'year')", nil, true},
		{"DATEADD(B, 10^300, 'hour')", nil, true},
		{"DATEADD(B, C, 'second')", nil, true},
		{"HOUR(7200)", 2.0, false},
		{"HOUR(10^300)", nil, true},
		{"DAY(10^300)", nil, true},
		{"MINUTE(10^300)", nil, true},
		{"SECOND(0-10^300)", nil, true},
		{"B + 1", day2, false},
		{"B + 10^300", nil, true},
		{"B - 10^300", nil, true},
		{"10^300 + B", nil, true},
		// A row range bound past the int32 range is refused, like a row index.
		{"SUM(A.(0:10^300))", nil, true},
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
