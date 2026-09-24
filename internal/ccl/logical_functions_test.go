package ccl

import (
	"strings"
	"testing"
)

// logicalCtx holds a column with a zero in it, so a guarded division has a row
// that would fail if the guard were not honoured.
func logicalCtx(t *testing.T) *MapContext {
	t.Helper()
	return mapCtx(t, map[string][]any{
		"A": {10.0, 20.0, 30.0},
		"B": {0.0, 5.0, 10.0},
	})
}

// AND() and OR() read an argument they could not convert as false, so
// AND('abc', TRUE) answered false — the same answer a real comparison gives —
// while 'abc' && TRUE reported the word. Docs/CCL.md calls && "equivalent to
// AND()", so the two must agree on what counts as a boolean.
func TestLogicalFunctionsReadArgumentsLikeTheOperators(t *testing.T) {
	ctx := logicalCtx(t)

	for _, tc := range []struct {
		fn   string
		op   string
		want any // nil: both forms must report an error
	}{
		{"AND('abc', TRUE)", "'abc' && TRUE", nil},
		{"AND(TRUE, 'abc')", "TRUE && 'abc'", nil},
		{"OR('abc', FALSE)", "'abc' || FALSE", nil},
		{"OR(FALSE, 'abc')", "FALSE || 'abc'", nil},
		// Not only a word: any value the conversion cannot read, such as the
		// whole row.
		{"AND(@, TRUE)", "@ && TRUE", nil},
		// Values the conversion does read stay readable, and answer the same
		// through the function as through the operator.
		{"AND(1, TRUE)", "1 && TRUE", true},
		{"AND(nil, TRUE)", "nil && TRUE", false},
		{"AND('yes', TRUE)", "'yes' && TRUE", true},
		{"OR(1, FALSE)", "1 || FALSE", true},
		{"OR(nil, FALSE)", "nil || FALSE", false},
	} {
		fnVal, fnErr := evalCol(t, ctx, tc.fn)
		opVal, opErr := evalCol(t, ctx, tc.op)

		if tc.want == nil {
			if fnErr == nil {
				t.Errorf("%s = %v, want an error like %s", tc.fn, fnVal, tc.op)
			}
			if opErr == nil {
				t.Errorf("%s = %v, want an error", tc.op, opVal)
			}
			continue
		}
		if fnErr != nil {
			t.Errorf("%s: %v", tc.fn, fnErr)
			continue
		}
		if opErr != nil {
			t.Errorf("%s: %v", tc.op, opErr)
			continue
		}
		if fnVal[0] != tc.want {
			t.Errorf("%s = %v, want %v", tc.fn, fnVal[0], tc.want)
		}
		if opVal[0] != fnVal[0] {
			t.Errorf("%s = %v but %s = %v", tc.fn, fnVal[0], tc.op, opVal[0])
		}
	}
}

// The message must say which argument, so a long condition can be fixed
// without bisecting it.
func TestLogicalFunctionErrorNamesTheArgument(t *testing.T) {
	ctx := logicalCtx(t)

	for expr, want := range map[string]string{
		"AND('abc', TRUE)":   "argument 1 to AND cannot be converted to boolean: abc",
		"AND(TRUE, 'abc')":   "argument 2 to AND cannot be converted to boolean: abc",
		"OR('abc', FALSE)":   "argument 1 to OR cannot be converted to boolean: abc",
		"OR(FALSE, 'abc')":   "argument 2 to OR cannot be converted to boolean: abc",
		"AND(TRUE, A > 'x')": "invalid operands",
	} {
		got, err := evalCol(t, ctx, expr)
		if err == nil {
			t.Errorf("%s = %v, want an error", expr, got)
			continue
		}
		if !strings.Contains(err.Error(), want) {
			t.Errorf("%s error = %q, want it to contain %q", expr, err.Error(), want)
		}
	}
}

// What the argument check does not touch: the argument count, and the
// short-circuit. An argument the short-circuit skips is never evaluated, so it
// cannot report anything either.
func TestLogicalFunctionsKeepTheirArityAndShortCircuit(t *testing.T) {
	ctx := logicalCtx(t)

	for expr, want := range map[string]any{
		"AND()":      true,
		"AND(TRUE)":  true,
		"AND(FALSE)": false,
		"OR()":       false,
		"OR(TRUE)":   true,
		"OR(FALSE)":  false,
		// The skipped argument is a word the check would otherwise report.
		"AND(FALSE, 'abc')": false,
		"OR(TRUE, 'abc')":   true,
	} {
		got, err := evalCol(t, ctx, expr)
		if err != nil {
			t.Errorf("%s: %v", expr, err)
			continue
		}
		if got[0] != want {
			t.Errorf("%s = %v, want %v", expr, got[0], want)
		}
	}

	// A guard written as a function keeps working on the row whose divisor is
	// zero, where the same guard written with && still fails.
	for expr, want := range map[string][]any{
		"AND(B != 0, A / B > 1)": {false, true, true},
		"OR(B == 0, A / B > 1)":  {true, true, true},
	} {
		got, err := evalCol(t, ctx, expr)
		if err != nil {
			t.Errorf("%s: %v", expr, err)
			continue
		}
		if len(got) != len(want) {
			t.Errorf("%s returned %d rows, want %d", expr, len(got), len(want))
			continue
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("%s row %d = %v, want %v", expr, i, got[i], want[i])
			}
		}
	}
}
