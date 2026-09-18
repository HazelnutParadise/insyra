package ccl

import (
	"testing"
	"time"
)

// wordCtx holds numbers in A and words in B. MapContext assigns Excel letters
// by sorted name, so the names are the letters.
func wordCtx(t *testing.T) *MapContext {
	t.Helper()
	return mapCtx(t, map[string][]any{
		"A": {10.0, 20.0},
		"B": {"hello", "world"},
	})
}

// Docs/CCL.md has said since v0.3.2 that a non-numeric string cannot be used
// in a numeric comparison and results in an error ("hello" > 5). The evaluator
// returned false, which reads as an answer.
func TestWordComparedWithNumberIsAnError(t *testing.T) {
	ctx := wordCtx(t)
	for _, expr := range []string{"'hello' > 5", "5 < 'hello'", "'hello' >= A", "B <= 5", "10 <= B <= 20"} {
		if got, err := evalCol(t, ctx, expr); err == nil {
			t.Errorf("%s = %v; comparing a word with a number must be an error", expr, got)
		}
	}
}

// Only a word against a number for size changed. Equality, numeric strings,
// nil and every other pair keep the answer they gave on v0.3.2, including two
// words, which have no ordering on this line.
func TestOtherComparisonsKeepTheirAnswers(t *testing.T) {
	ctx := wordCtx(t)
	for _, c := range []struct {
		expr string
		want bool
	}{
		{"'5' > 3", true},
		{"'10' > '9'", true},
		{"'hello' == 5", false},
		{"'hello' != 5", true},
		{"B == A", false},
		{"'abc' < 'abd'", false},
		{"'abc' > 'abd'", false},
		{"'hello' > '5'", false},
		{"nil > 10", false},
		{"nil < 5", false},
		{"true > 'hello'", false},
	} {
		got, err := evalCol(t, ctx, c.expr)
		if err != nil {
			t.Errorf("%s: unexpected error %v", c.expr, err)
			continue
		}
		if got[0] != c.want {
			t.Errorf("%s = %v, want %v", c.expr, got[0], c.want)
		}
	}
}

// dateWordCtx holds date strings in A, empty strings in B, the same dates as
// time.Time in C, and numbers in D.
func dateWordCtx(t *testing.T) *MapContext {
	t.Helper()
	return mapCtx(t, map[string][]any{
		"A": {"2024-01-02", "2024-06-30T12:00:00Z"},
		"B": {"", ""},
		"C": {time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), time.Date(2024, 6, 30, 12, 0, 0, 0, time.UTC)},
		"D": {10.0, 20.0},
	})
}

// A string the evaluator itself reads as a date is not a word: Docs/CCL.md
// says date strings are parsed as dates and that date comparisons work, and
// CSV and Excel loads store date columns as strings. The empty string is not
// a word either. Both returned false on v0.3.2 and must keep doing so, the
// same answer the dates held as time.Time give.
func TestDateStringAndEmptyStringStillCompareFalse(t *testing.T) {
	ctx := dateWordCtx(t)
	for _, expr := range []string{
		"A > 0", "A < 0", "A >= 5", "A <= 5",
		"0 < A", "0 > A",
		"B > 5", "B < 5", "B >= 0", "B <= 0",
		"C > 0", "C < 0",
		"A > D", "B >= D",
	} {
		got, err := evalCol(t, ctx, expr)
		if err != nil {
			t.Errorf("%s: unexpected error %v", expr, err)
			continue
		}
		for i, v := range got {
			if v != false {
				t.Errorf("%s row %d = %v, want false", expr, i, v)
			}
		}
	}
}

// The mixed column from the report: numeric strings compare, the blank one
// returns false, and the whole expression still produces a column.
func TestBlankAmongNumericStringsStillComparesFalse(t *testing.T) {
	ctx := mapCtx(t, map[string][]any{"A": {"10", "", "30"}})
	got, err := evalCol(t, ctx, "A > 5")
	if err != nil {
		t.Fatalf("A > 5: unexpected error %v", err)
	}
	want := []any{true, false, true}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("A > 5 row %d = %v, want %v", i, got[i], want[i])
		}
	}
}
