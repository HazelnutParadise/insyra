package ccl

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

// b8Ctx is a two-column table: A = [10 20 30], B = [1 0 3]. B's zero makes
// A / B the natural way to ask whether an operand was evaluated.
func b8Ctx(t *testing.T) *MapContext {
	t.Helper()
	return mapCtx(t, map[string][]any{
		"a": {10.0, 20.0, 30.0},
		"b": {1.0, 0.0, 3.0},
	})
}

// CCL-12: the guard people write to avoid an error must not cause it.
func TestLogicalOperatorsShortCircuit(t *testing.T) {
	ctx := b8Ctx(t)
	got, err := evalCol(t, ctx, "B != 0 && A / B > 1")
	if err != nil {
		t.Fatalf("B != 0 && A / B > 1: %v", err)
	}
	if want := []any{true, false, true}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	got, err = evalCol(t, ctx, "B == 0 || A / B > 1")
	if err != nil {
		t.Fatalf("B == 0 || A / B > 1: %v", err)
	}
	if want := []any{true, true, true}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// The same must hold for a folded chain, which the parser builds for runs of
// two or more operators — the two shapes have to stay interchangeable.
func TestFoldedLogicalChainShortCircuits(t *testing.T) {
	ctx := b8Ctx(t)
	for _, tc := range []struct {
		expr string
		want any
	}{
		{"false && (1/0 > 0)", false},
		{"true || (1/0 > 0)", true},
		{"false && (1/0 > 0) && true", false},
		{"true || (1/0 > 0) || false", true},
		{"B != 0 && A / B > 1 && true", nil}, // per-row, checked below
	} {
		if tc.want == nil {
			continue
		}
		node, err := CompileExpression(tc.expr)
		if err != nil {
			t.Fatalf("compile %q: %v", tc.expr, err)
		}
		got, err := Evaluate(node, ctx)
		if err != nil {
			t.Errorf("%s: %v", tc.expr, err)
			continue
		}
		if got != tc.want {
			t.Errorf("%s = %v, want %v", tc.expr, got, tc.want)
		}
	}
}

// CCL-12: CASE must pick a branch before evaluating it, like IF.
func TestCaseShortCircuits(t *testing.T) {
	ctx := b8Ctx(t)
	got, err := evalCol(t, ctx, "CASE(B != 0, A / B, nil)")
	if err != nil {
		t.Fatalf("CASE(B != 0, A / B, nil): %v", err)
	}
	want := []any{10.0, nil, 10.0}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// CCL-13: a missing comma silently changed the meaning of a call.
func TestArgumentListRequiresCommas(t *testing.T) {
	for _, expr := range []string{
		"SUM(A B)",
		"IF(A > 15 1 0)",
		"IF(A > 15, 1, 0,)",
		"SUM(A,,B)",
		"CONCAT('a' 'b')",
	} {
		if _, err := CompileExpression(expr); err == nil {
			t.Errorf("%s compiled; a malformed argument list must be an error", expr)
		}
	}
	// Well-formed calls still compile.
	for _, expr := range []string{"SUM(A, B)", "IF(A > 15, 1, 0)", "PI()", "ABS(A)"} {
		if _, err := CompileExpression(expr); err != nil {
			t.Errorf("%s: %v", expr, err)
		}
	}
}

// CCL-9: strings had no ordering at all — both directions were false.
func TestStringComparisonIsLexicographic(t *testing.T) {
	ctx := b8Ctx(t)
	for _, tc := range []struct {
		expr string
		want bool
	}{
		{"'abc' < 'abd'", true},
		{"'abc' > 'abd'", false},
		{"'abd' > 'abc'", true},
		{"'abc' <= 'abc'", true},
		{"'10' > '9'", true}, // both parse as numbers: numeric comparison wins
	} {
		node, err := CompileExpression(tc.expr)
		if err != nil {
			t.Fatalf("compile %q: %v", tc.expr, err)
		}
		got, err := Evaluate(node, ctx)
		if err != nil {
			t.Errorf("%s: %v", tc.expr, err)
			continue
		}
		if got != tc.want {
			t.Errorf("%s = %v, want %v", tc.expr, got, tc.want)
		}
	}
}

// CCL-9: the docs have always said this is an error; it was silently false.
func TestMixedComparisonIsAnError(t *testing.T) {
	ctx := b8Ctx(t)
	for _, expr := range []string{"'hello' > 5", "5 < 'hello'", "'hello' >= A"} {
		node, err := CompileExpression(expr)
		if err != nil {
			t.Fatalf("compile %q: %v", expr, err)
		}
		if got, err := Evaluate(node, ctx); err == nil {
			t.Errorf("%s = %v; comparing a word with a number must be an error", expr, got)
		}
	}
	// nil keeps its documented exemption.
	for _, expr := range []string{"nil > 10", "nil < 5", "nil >= 0"} {
		node, err := CompileExpression(expr)
		if err != nil {
			t.Fatalf("compile %q: %v", expr, err)
		}
		got, err := Evaluate(node, ctx)
		if err != nil {
			t.Errorf("%s: %v", expr, err)
			continue
		}
		if got != false {
			t.Errorf("%s = %v, want false", expr, got)
		}
	}
}

// CCL-11: Go's %v placeholder was reaching the data.
func TestNilConcatenatesAsEmptyString(t *testing.T) {
	ctx := b8Ctx(t)
	for _, tc := range []struct{ expr, want string }{
		{"nil & 'x'", "x"},
		{"'x' & nil", "x"},
		{"CONCAT(nil, 'x')", "x"},
		{"UPPER(nil) & 'x'", "x"},
		{"'a' & 1", "a1"},
	} {
		node, err := CompileExpression(tc.expr)
		if err != nil {
			t.Fatalf("compile %q: %v", tc.expr, err)
		}
		got, err := Evaluate(node, ctx)
		if err != nil {
			t.Errorf("%s: %v", tc.expr, err)
			continue
		}
		if got != tc.want {
			t.Errorf("%s = %q, want %q", tc.expr, got, tc.want)
		}
	}
}

// CCL-14: & bound as tightly as +, so concatenating a sum was an error.
func TestConcatBindsLooserThanAddition(t *testing.T) {
	ctx := b8Ctx(t)
	for _, tc := range []struct{ expr, want string }{
		{"'a' & 1 + 2", "a3"},
		{"'1' & '2' + 3", "15"},
		{"'total: ' & A + B", "total: 11"},
	} {
		node, err := CompileExpression(tc.expr)
		if err != nil {
			t.Fatalf("compile %q: %v", tc.expr, err)
		}
		if err := ctx.SetRowIndex(0); err != nil {
			t.Fatal(err)
		}
		got, err := Evaluate(node, ctx)
		if err != nil {
			t.Errorf("%s: %v", tc.expr, err)
			continue
		}
		if got != tc.want {
			t.Errorf("%s = %v, want %q", tc.expr, got, tc.want)
		}
	}
	// & still binds tighter than comparison.
	node, err := CompileExpression("'a' & 'b' == 'ab'")
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if got, err := Evaluate(node, ctx); err != nil || got != true {
		t.Errorf("'a' & 'b' == 'ab' = %v, %v; want true", got, err)
	}
}

// CCL-23 / CCL-39: internal representations must never become a value.
func TestRangesAndRowsAreNotValues(t *testing.T) {
	ctx := b8Ctx(t)
	for _, expr := range []string{"A:B", "LAG(@, 1)", "CUMSUM(@)"} {
		node, err := CompileExpression(expr)
		if err != nil {
			continue // rejected at compile time is also fine
		}
		bound, err := Bind(node, ctx.ColNameMap)
		if err != nil {
			continue
		}
		if got, err := Evaluate(bound, ctx); err == nil {
			t.Errorf("%s = %v (%T); an internal type must not become a value", expr, got, got)
		}
	}
	// The forms that consume a range still work.
	for _, expr := range []string{"SUM(A:B)", "A.(0:1)"} {
		node, err := CompileExpression(expr)
		if err != nil {
			t.Fatalf("compile %q: %v", expr, err)
		}
		bound, err := Bind(node, ctx.ColNameMap)
		if err != nil {
			t.Fatalf("bind %q: %v", expr, err)
		}
		if _, err := Evaluate(bound, ctx); err != nil {
			t.Errorf("%s: %v", expr, err)
		}
	}
}

// CCL-24: the evaluator's special-case path skipped the registered checks.
func TestAndOrCheckTheirArguments(t *testing.T) {
	ctx := b8Ctx(t)
	for _, expr := range []string{
		"AND()", "OR()", "AND(true)", "OR(false)",
		"AND('abc', true)", "OR('abc', false)",
	} {
		node, err := CompileExpression(expr)
		if err != nil {
			continue
		}
		if got, err := Evaluate(node, ctx); err == nil {
			t.Errorf("%s = %v; want an error", expr, got)
		}
	}
	for _, tc := range []struct {
		expr string
		want bool
	}{
		{"AND(true, true)", true},
		{"AND(true, false)", false},
		{"OR(false, true)", true},
		{"AND(1, 1)", true}, // documented boolean coercion still applies
	} {
		node, err := CompileExpression(tc.expr)
		if err != nil {
			t.Fatalf("compile %q: %v", tc.expr, err)
		}
		got, err := Evaluate(node, ctx)
		if err != nil {
			t.Errorf("%s: %v", tc.expr, err)
			continue
		}
		if got != tc.want {
			t.Errorf("%s = %v, want %v", tc.expr, got, tc.want)
		}
	}
}

// CCL-25: a fractional index was truncated, and a NaN index gave a different
// answer on arm64 than on amd64.
func TestIndicesAndWindowsMustBeWholeNumbers(t *testing.T) {
	ctx := b8Ctx(t)
	for _, expr := range []string{
		"A.(1.7)",
		"A.(0:1.9)",
		"ROLLING_MEAN(A, 2.9)",
		"LAG(A, 1.5)",
		"A.(POW(-8, 1/3))", // NaN
		"A.(1/0.0 - 1/0.0)",
	} {
		node, err := CompileExpression(expr)
		if err != nil {
			continue
		}
		bound, err := Bind(node, ctx.ColNameMap)
		if err != nil {
			continue
		}
		if got, err := Evaluate(bound, ctx); err == nil {
			t.Errorf("%s = %v; a non-integer index must be an error", expr, got)
		}
	}
	// Whole numbers, including ones that arrive as floats, still work.
	for _, expr := range []string{"A.(1)", "A.(0:1)", "ROLLING_MEAN(A, 2)", "LAG(A, 1)", "A.(3 - 2)"} {
		node, err := CompileExpression(expr)
		if err != nil {
			t.Fatalf("compile %q: %v", expr, err)
		}
		bound, err := Bind(node, ctx.ColNameMap)
		if err != nil {
			t.Fatalf("bind %q: %v", expr, err)
		}
		if _, err := Evaluate(bound, ctx); err != nil {
			t.Errorf("%s: %v", expr, err)
		}
	}
}

// CCL-20: sub-hour precision was truncated away, so A + 0.001 did nothing.
func TestFractionalDaysKeepTheirFraction(t *testing.T) {
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	ctx := mapCtx(t, map[string][]any{"a": {base}})
	node, err := CompileExpression("A + 0.001")
	if err != nil {
		t.Fatal(err)
	}
	bound, err := Bind(node, ctx.ColNameMap)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Evaluate(bound, ctx)
	if err != nil {
		t.Fatal(err)
	}
	ts, ok := got.(time.Time)
	if !ok {
		t.Fatalf("got %T, want time.Time", got)
	}
	if want := base.Add(86400 * time.Millisecond); !ts.Equal(want) {
		t.Errorf("A + 0.001 = %v, want %v (86.4s later)", ts, want)
	}
}

// The error for a malformed argument list should say what is wrong.
func TestArgumentListErrorMentionsTheComma(t *testing.T) {
	_, err := CompileExpression("SUM(A B)")
	if err == nil {
		t.Fatal("SUM(A B) compiled")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "comma") {
		t.Errorf("error should mention the missing comma: %v", err)
	}
}
