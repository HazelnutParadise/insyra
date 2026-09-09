package ccl

import (
	"errors"
	"strings"
	"testing"
)

// CCL-16: a compile failure must be a typed value carrying where in the
// expression the problem is, measured the same way everywhere.
func TestCompileErrorCarriesOffsetAndText(t *testing.T) {
	for _, tc := range []struct {
		expr string
		near string
	}{
		{"SUM(A B)", "B"},
		{"1 +)", ")"},
		{"中文 + 1", "中文"},
	} {
		_, err := CompileExpression(tc.expr)
		if err == nil {
			t.Errorf("%s compiled", tc.expr)
			continue
		}
		var ce *CompileError
		if !errors.As(err, &ce) {
			t.Errorf("%s: error is %T, want *CompileError", tc.expr, err)
			continue
		}
		if ce.Expr != tc.expr {
			t.Errorf("%s: Expr = %q", tc.expr, ce.Expr)
		}
		if ce.Offset < 0 || ce.Offset > len(tc.expr) {
			t.Errorf("%s: Offset = %d, outside the expression", tc.expr, ce.Offset)
			continue
		}
		// The offset must point at the reported text inside the expression.
		if tc.near != "" && !strings.HasPrefix(tc.expr[ce.Offset:], tc.near) {
			t.Errorf("%s: Offset %d points at %q, want it to point at %q",
				tc.expr, ce.Offset, tc.expr[ce.Offset:], tc.near)
		}
	}
}

// A position must always mean a byte offset. The parser used to report an
// index into the token list under the same word.
func TestCompileErrorPositionIsAByteOffset(t *testing.T) {
	// "SUM(A B)": the fourth token is B, whose byte offset is 6. A token index
	// would be 3, which points at a space.
	_, err := CompileExpression("SUM(A B)")
	var ce *CompileError
	if !errors.As(err, &ce) {
		t.Fatalf("error is %T, want *CompileError", err)
	}
	if ce.Offset != 6 {
		t.Errorf("Offset = %d, want 6 (the byte offset of B, not its token index)", ce.Offset)
	}
}

// CCL-16: no Go struct dumps or pointers in a message.
func TestErrorMessagesHaveNoGoInternals(t *testing.T) {
	ctx := mapCtx(t, map[string][]any{"a": {1.0}, "b": {2.0}})
	for _, expr := range []string{"1 +)", "SUM(A B)", "A:1", "1:B", "(1+2):(3+4)"} {
		node, err := CompileExpression(expr)
		if err == nil {
			bound, bindErr := Bind(node, ctx.ColNameMap)
			if bindErr != nil {
				err = bindErr
			} else {
				_, err = Evaluate(bound, ctx)
			}
		}
		if err == nil {
			continue
		}
		msg := err.Error()
		for _, bad := range []string{"0x", "&{", "{0 ", "{5 "} {
			if strings.Contains(msg, bad) {
				t.Errorf("%s: message leaks Go internals (%q): %s", expr, bad, msg)
			}
		}
	}
}

// An evaluation failure must name the row and keep the cause reachable.
func TestEvalErrorCarriesRowAndUnwraps(t *testing.T) {
	inner := errors.New("division by zero")
	e := &EvalError{Expr: "A / B", Row: 1, Err: inner}
	if !errors.Is(e, inner) {
		t.Error("EvalError does not unwrap to its cause")
	}
	if !strings.Contains(e.Error(), "row 1") {
		t.Errorf("message should name the row: %s", e.Error())
	}
	if !strings.Contains(e.Error(), "A / B") {
		t.Errorf("message should name the expression: %s", e.Error())
	}

	rowless := &EvalError{Expr: "SUM(A)", Row: -1, Err: inner}
	if strings.Contains(rowless.Error(), "row -1") {
		t.Errorf("a row-independent expression must not report row -1: %s", rowless.Error())
	}
}
