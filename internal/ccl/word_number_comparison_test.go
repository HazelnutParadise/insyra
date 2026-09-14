package ccl

import "testing"

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
