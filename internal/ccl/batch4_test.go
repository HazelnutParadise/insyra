package ccl

import (
	"strings"
	"sync"
	"testing"
)

// mapCtx builds a MapContext and fails the test on error.
func mapCtx(t *testing.T, data map[string][]any) *MapContext {
	t.Helper()
	ctx, err := NewMapContext(data)
	if err != nil {
		t.Fatal(err)
	}
	return ctx
}

// evalCol compiles expr, binds it against ctx's column names, and evaluates
// it for every row, returning the per-row values.
func evalCol(t *testing.T, ctx *MapContext, expr string) ([]any, error) {
	t.Helper()
	node, err := CompileExpression(expr)
	if err != nil {
		return nil, err
	}
	bound, err := Bind(node, ctx.ColNameMap)
	if err != nil {
		return nil, err
	}
	if !IsRowDependent(GetExpressionNode(bound)) {
		v, err := Evaluate(bound, ctx)
		if err != nil {
			return nil, err
		}
		if col, ok := v.([]any); ok {
			return col, nil
		}
		return []any{v}, nil
	}
	out := make([]any, ctx.Rows)
	for i := 0; i < ctx.Rows; i++ {
		if err := ctx.SetRowIndex(i); err != nil {
			return nil, err
		}
		v, err := Evaluate(bound, ctx)
		if err != nil {
			return nil, err
		}
		out[i] = v
	}
	return out, nil
}

// CCL-4: MapContext column order must not depend on map iteration.
func TestMapContextDeterministicOrder(t *testing.T) {
	for i := 0; i < 50; i++ {
		ctx := mapCtx(t, map[string][]any{"z": {3}, "a": {1}, "m": {2}})
		if got := strings.Join(ctx.ColNames, ","); got != "a,m,z" {
			t.Fatalf("iteration %d: column order %q", i, got)
		}
	}
}

// CCL-5: a sequence function fed by another sequence function keeps the
// full column instead of collapsing to one element.
func TestNestedSequenceFunctions(t *testing.T) {
	ctx := mapCtx(t, map[string][]any{"A": {10.0, 20.0, 30.0}})
	got, err := evalCol(t, ctx, "LAG(LAG(A,1),1)")
	if err != nil {
		t.Fatal(err)
	}
	seqApproxEqual(t, got, []any{nil, nil, 10.0}, 0)

	// CUMSUM counts nil as 0 (CCL-wide nil→0 coercion), so the shifted
	// column accumulates from the second row.
	got, err = evalCol(t, ctx, "CUMSUM(LAG(A,1))")
	if err != nil {
		t.Fatal(err)
	}
	seqApproxEqual(t, got, []any{0.0, 10.0, 30.0}, 0)
}

// CCL-6: absurd shift, length, and repeat counts become errors, never panics.
func TestHugeArgumentsDoNotPanic(t *testing.T) {
	ctx := mapCtx(t, map[string][]any{"A": {1.0, 2.0, 3.0}})
	for expr, wantErr := range map[string]bool{
		"LEAD(A, 10^300)":         true,
		"LAG(A, 0-10^300)":        true,
		"REPEAT('x', 10^300)":     true,
		"ROLLING_MEAN(A, 10^300)": true,
		"MID('abc', 2, 10^300)":   false, // Excel semantics: clamp to the end
	} {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("%s panicked: %v", expr, r)
				}
			}()
			got, err := evalCol(t, ctx, expr)
			if wantErr && err == nil {
				t.Errorf("%s: expected an error", expr)
			}
			if !wantErr && (err != nil || got[0] != "bc") {
				t.Errorf("%s = %v, %v; want \"bc\"", expr, got, err)
			}
		}()
	}
}

// CCL-7: registering a function while another goroutine evaluates must be
// race free (run under -race).
func TestRegisterFunctionConcurrentWithEvaluate(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(2)
		go func(i int) {
			defer wg.Done()
			RegisterFunction("BATCH4_"+strings.Repeat("X", i+1), func(args ...any) (any, error) { return 1.0, nil })
		}(i)
		go func() {
			defer wg.Done()
			// A context carries a current-row cursor, so each goroutine
			// evaluates against its own.
			ctx := mapCtx(t, map[string][]any{"A": {1.0, 2.0, 3.0}})
			_, _ = evalCol(t, ctx, "ABS(A) + 1")
		}()
	}
	wg.Wait()
}

// CCL-1 (keyword part): NULL / TRUE / FALSE in any case are literals, not
// column references.
func TestKeywordsAreCaseInsensitive(t *testing.T) {
	ctx := mapCtx(t, map[string][]any{"A": {1.0}})
	got, err := evalCol(t, ctx, "IF(TRUE, NULL, 1)")
	if err != nil {
		t.Fatal(err)
	}
	if got[0] != nil {
		t.Fatalf("IF(TRUE, NULL, 1) = %v, want nil", got[0])
	}
	got, err = evalCol(t, ctx, "IF(False, 1, 2)")
	if err != nil {
		t.Fatal(err)
	}
	if f, _ := toFloat64(got[0]); f != 2 {
		t.Fatalf("IF(False, 1, 2) = %v, want 2", got[0])
	}
}
