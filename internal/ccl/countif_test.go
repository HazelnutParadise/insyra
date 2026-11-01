package ccl

import (
	"strings"
	"testing"
)

// COUNTIF, SUMIF and AVGIF take the condition as an expression, so anything that
// gives true or false works, a regular expression and several conditions included.
func TestConditionalAggregates(t *testing.T) {
	ctx := mapCtx(t, map[string][]any{
		"A": {10.0, -3.0, 25.0, nil, 7.0},
		"B": {1.0, 2.0, 3.0, 4.0, nil},
		"C": {"apple", "Apple", "banana", "apple pie", nil},
	})
	for _, c := range []struct {
		expr string
		want any
	}{
		{"COUNTIF(A > 0)", 3.0},
		{"COUNTIF(A < 0)", 1.0},
		{"COUNTIF(A > 100)", 0.0},
		{"COUNTIF(C == 'apple')", 1.0},
		{"COUNTIF(C != 'apple')", 4.0}, // an empty cell is not 'apple', as with IF
		{"COUNTIF(A > 0 && B > 1)", 1.0},
		{"COUNTIF(A < 0 || B == 3)", 2.0},
		{"COUNTIF(REGEX_MATCH(C, '^apple'))", 2.0},
		{"COUNTIF(REGEX_MATCH(C, '(?i)^apple$'))", 2.0},
		{"COUNTIF(STARTSWITH(C, 'b'))", 1.0},
		{"countif(A > 0)", 3.0},
		{"COUNTIF(TRUE)", 5.0},
		{"COUNTIF(FALSE)", 0.0},
		{"SUMIF(A > 0, B)", 4.0},
		{"SUMIF(A > 0, A)", 42.0},
		{"SUMIF(REGEX_MATCH(C, '^apple'), B)", 5.0},
		{"SUMIF(A > 100, B)", 0.0},
		{"SUMIF(TRUE, B)", 10.0},
		{"SUMIF(FALSE, B)", 0.0},
		{"AVGIF(A > 0, B)", 2.0},
		{"AVGIF(A > 0, A)", 14.0},
		{"AVGIF(A > 100, B)", 0.0},
		{"COUNTIF(A > 0) / COUNT(A)", 0.75},
	} {
		got, err := evalOne(t, ctx, c.expr)
		if err != nil {
			t.Errorf("%s: %v", c.expr, err)
			continue
		}
		if f, ok := got.(float64); ok {
			if f != c.want {
				t.Errorf("%s = %v, want %v", c.expr, f, c.want)
			}
			continue
		}
		t.Errorf("%s = %#v (%T), want %v", c.expr, got, got, c.want)
	}
}

func TestConditionalAggregatesRefuse(t *testing.T) {
	ctx := mapCtx(t, map[string][]any{"A": {1.0, "word"}, "B": {1.0, 2.0}, "C": {"yes", "maybe"}})
	for _, c := range []struct{ expr, want string }{
		{"COUNTIF()", "COUNTIF requires 1 argument"},
		{"COUNTIF(B > 0, B)", "COUNTIF requires 1 argument"},
		{"SUMIF(B > 0)", "SUMIF requires 2 arguments"},
		{"AVGIF(B > 0)", "AVGIF requires 2 arguments"},
		{"SUMIF(B > 0, B, B)", "SUMIF requires 2 arguments"},
		{"COUNTIF(C)", "maybe"},
		{"COUNTIF(A > 0)", "word"},
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

	// A word compares with nothing, so VALUE turns it into an empty cell first.
	if got, err := evalOne(t, ctx, "COUNTIF(VALUE(A) > 0)"); err != nil || got != 1.0 {
		t.Errorf("COUNTIF(VALUE(A) > 0) = %v, %v; want 1", got, err)
	}
}
