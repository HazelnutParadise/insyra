package ccl

import (
	"math"
	"reflect"
	"strings"
	"testing"
)

// perfCtx builds a column with the shapes that expose an optimisation which
// changes arithmetic order or parsing: mixed magnitudes, a blank, a NaN, text
// and a date.
func perfCtx(t *testing.T, rows int) *MapContext {
	t.Helper()
	a := make([]any, rows)
	s := make([]any, rows)
	for i := range a {
		switch {
		case i%17 == 0:
			a[i] = nil
		case i%23 == 0:
			a[i] = math.NaN()
		case i%7 == 0:
			a[i] = float64(i) * 1e8
		default:
			a[i] = float64(i%50) + 0.125
		}
		s[i] = "item-" + string(rune('a'+i%5))
	}
	return mapCtx(t, map[string][]any{"a": a, "s": s})
}

// evalRows evaluates expr for every row, optionally folding row-invariant
// aggregates first. Comparing the two is the point: folding must be invisible.
func evalRows(t *testing.T, ctx *MapContext, expr string, fold bool) []any {
	t.Helper()
	node, err := CompileExpression(expr)
	if err != nil {
		t.Fatalf("compile %q: %v", expr, err)
	}
	bound, err := Bind(node, ctx.ColNameMap)
	if err != nil {
		t.Fatalf("bind %q: %v", expr, err)
	}
	if fold {
		bound = FoldRowInvariantAggregates(bound, ctx)
	}
	out := make([]any, ctx.Rows)
	for i := range ctx.Rows {
		if err := ctx.SetRowIndex(i); err != nil {
			t.Fatal(err)
		}
		v, err := Evaluate(bound, ctx)
		if err != nil {
			t.Fatalf("%s row %d: %v", expr, i, err)
		}
		out[i] = v
	}
	return out
}

// CCL-18: an aggregate inside a per-row expression was recomputed for every
// row. Computing it once must produce exactly what the loop produced.
func TestFoldingDoesNotChangeResults(t *testing.T) {
	ctx := perfCtx(t, 200)
	for _, expr := range []string{
		"A / SUM(A)",
		"(A - AVG(A)) / STDEV(A)",
		"A - MIN(A)",
		"A / MAX(A)",
		"SUM(A) + COUNT(A)",
		"A * SUM(A) / AVG(A)",
		"IF(A > AVG(A), 'high', 'low')",
		"MEDIAN(A)",
		"SUM(A) > 0 && A > 0",
		"A + SUM(A) + A * AVG(A)",
	} {
		t.Run(expr, func(t *testing.T) {
			unfolded := evalRows(t, ctx, expr, false)
			folded := evalRows(t, ctx, expr, true)
			if len(unfolded) != len(folded) {
				t.Fatalf("length %d vs %d", len(unfolded), len(folded))
			}
			for i := range unfolded {
				if !sameValue(unfolded[i], folded[i]) {
					t.Fatalf("row %d: folded %v, unfolded %v", i, folded[i], unfolded[i])
				}
			}
		})
	}
}

// sameValue compares two cell values, treating two NaNs as equal and comparing
// floats by their bits so a change in the last place cannot slip through.
func sameValue(a, b any) bool {
	af, aok := a.(float64)
	bf, bok := b.(float64)
	if aok && bok {
		if math.IsNaN(af) && math.IsNaN(bf) {
			return true
		}
		return math.Float64bits(af) == math.Float64bits(bf)
	}
	return reflect.DeepEqual(a, b)
}

// An aggregate that reads the current row is not row-invariant and must keep
// being evaluated per row.
func TestAggregateReadingTheRowIsNotFolded(t *testing.T) {
	ctx := perfCtx(t, 20)
	const expr = "SUM(A.(0:#))"
	unfolded := evalRows(t, ctx, expr, false)
	folded := evalRows(t, ctx, expr, true)

	distinct := map[float64]struct{}{}
	for _, v := range unfolded {
		if f, ok := v.(float64); ok {
			distinct[f] = struct{}{}
		}
	}
	if len(distinct) < 2 {
		t.Fatalf("%s should differ per row; got %v", expr, unfolded)
	}
	for i := range unfolded {
		if !sameValue(unfolded[i], folded[i]) {
			t.Fatalf("row %d: folding changed a row-dependent aggregate: %v vs %v", i, folded[i], unfolded[i])
		}
	}
}

// An aggregate that cannot be evaluated must be left alone, so the error still
// arrives from the row loop where it always did.
func TestFoldingLeavesAFailingAggregateAlone(t *testing.T) {
	ctx := perfCtx(t, 5)
	node, err := CompileExpression("A / SUM(B)")
	if err != nil {
		t.Fatal(err)
	}
	// B is not bound, so binding fails; fold the unbound tree instead and
	// check it survives untouched rather than swallowing the problem.
	folded := FoldRowInvariantAggregates(node, ctx)
	if _, err := Evaluate(folded, ctx); err == nil {
		t.Error("an aggregate over a missing column should still fail")
	}
}

// CCL-38: the rolling window converts its column once now. Every window must
// still receive the same values in the same order, so the sums are identical.
func TestRollingMatchesNaiveReference(t *testing.T) {
	rows := 300
	ctx := perfCtx(t, rows)
	col := ctx.Data["a"]

	for _, window := range []int{1, 2, 7, 50} {
		got, err := seqRollingSum([][]any{col, {window}}...)
		if err != nil {
			t.Fatalf("window %d: %v", window, err)
		}
		want := naiveRollingSum(col, window)
		for i := range want {
			if !sameValue(got[i], want[i]) {
				t.Fatalf("window %d row %d: got %v, want %v", window, i, got[i], want[i])
			}
		}
	}
}

// naiveRollingSum is the shape the implementation had before it hoisted the
// conversion: convert inside the window, one fresh slice per window.
func naiveRollingSum(col []any, window int) []any {
	out := make([]any, len(col))
	for i := range col {
		lo := max(i-window+1, 0)
		var vals []float64
		for j := lo; j <= i; j++ {
			f, ok := toFloat64(col[j])
			if !ok || math.IsNaN(f) {
				continue
			}
			vals = append(vals, f)
		}
		if len(vals) < window {
			out[i] = nil
			continue
		}
		var s float64
		for _, v := range vals {
			s += v
		}
		out[i] = s
	}
	return out
}

// CCL-38: a string is only offered to the date parsers when it could be one.
// Dates must still work and text must still behave as before.
func TestDateFastPathKeepsBehaviour(t *testing.T) {
	ctx := mapCtx(t, map[string][]any{
		"a": {"2024-01-02", "2024-01-02T03:04:05Z", "item-a", "", "x2024-01-02"},
	})
	node, err := CompileExpression("A + 1")
	if err != nil {
		t.Fatal(err)
	}
	bound, err := Bind(node, ctx.ColNameMap)
	if err != nil {
		t.Fatal(err)
	}
	at := func(row int) (any, error) {
		if err := ctx.SetRowIndex(row); err != nil {
			t.Fatal(err)
		}
		return Evaluate(bound, ctx)
	}

	// The two date strings still parse and shift by a day.
	for _, row := range []int{0, 1} {
		v, err := at(row)
		if err != nil {
			t.Errorf("row %d: %v", row, err)
			continue
		}
		if _, ok := v.(interface{ Year() int }); !ok {
			t.Errorf("row %d: a date string must still be read as a date, got %T", row, v)
		}
	}

	// Text is neither a date nor a number, so adding to it still fails —
	// including "x2024-01-02", which the fast path skips on its first byte
	// and which the parsers rejected before.
	for _, row := range []int{2, 3, 4} {
		if _, err := at(row); err == nil {
			t.Errorf("row %d: adding to text should still fail", row)
		}
	}
}

// CCL-38: patterns are compiled once and cached; the cache must not confuse
// two patterns, and an invalid one must still be reported.
func TestRegexCacheKeepsPatternsApart(t *testing.T) {
	ctx := mapCtx(t, map[string][]any{"a": {"item-1", "item-22"}})
	for _, tc := range []struct {
		expr string
		want []any
	}{
		{"REGEX_MATCH(A, '^item-[0-9]$')", []any{true, false}},
		{"REGEX_MATCH(A, '^item-[0-9]+$')", []any{true, true}},
		{"REGEX_MATCH(A, 'nope')", []any{false, false}},
	} {
		got := evalRows(t, ctx, tc.expr, true)
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s = %v, want %v", tc.expr, got, tc.want)
		}
	}

	node, err := CompileExpression("REGEX_MATCH(A, '[')")
	if err != nil {
		t.Fatal(err)
	}
	bound, err := Bind(node, ctx.ColNameMap)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Evaluate(bound, ctx); err == nil {
		t.Error("an invalid pattern must still be reported")
	} else if !strings.Contains(err.Error(), "pattern") {
		t.Errorf("the error should name the pattern: %v", err)
	}
}

// Without this the comparison test above would pass just as happily if folding
// did nothing at all.
func TestFoldingActuallyFolds(t *testing.T) {
	ctx := perfCtx(t, 20)
	for _, tc := range []struct {
		expr       string
		wantFolded bool
	}{
		{"A / SUM(A)", true},
		{"(A - AVG(A)) / STDEV(A)", true},
		{"SUM(A.(0:#))", false}, // reads the current row
		{"A / 2", false},        // nothing to fold
	} {
		node, err := CompileExpression(tc.expr)
		if err != nil {
			t.Fatalf("compile %q: %v", tc.expr, err)
		}
		bound, err := Bind(node, ctx.ColNameMap)
		if err != nil {
			t.Fatalf("bind %q: %v", tc.expr, err)
		}
		folded := FoldRowInvariantAggregates(bound, ctx)
		changed := countAggregates(folded) < countAggregates(bound)
		if changed != tc.wantFolded {
			t.Errorf("%s: folded = %v, want %v (aggregates %d -> %d)",
				tc.expr, changed, tc.wantFolded,
				countAggregates(bound), countAggregates(folded))
		}
	}
}

// countAggregates counts aggregate calls left in a tree.
func countAggregates(n cclNode) int {
	total := 0
	switch t := n.(type) {
	case *funcCallNode:
		if _, isAgg := lookupAggregateFunction(strings.ToUpper(t.name)); isAgg {
			total++
		}
		for _, arg := range t.args {
			total += countAggregates(arg)
		}
	case *cclBinaryOpNode:
		total += countAggregates(t.left) + countAggregates(t.right)
	case *cclFoldChainNode:
		total += countAggregates(t.init)
		for _, operand := range t.operands {
			total += countAggregates(operand)
		}
	case *cclChainedComparisonNode:
		for _, v := range t.values {
			total += countAggregates(v)
		}
	}
	return total
}
