package ccl_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra/engine/ccl"
)

// engine/ccl is the supported way to reach the CCL compiler from outside the
// module, and it had no test. Every function here forwards to internal/ccl, so
// what needs checking is that the forwarding is wired correctly and that the
// re-exported types are usable from a consumer's package — which is why this
// file is an external test package.

func mapContext(t *testing.T) ccl.Context {
	t.Helper()
	ctx, err := ccl.NewMapContext(map[string][]any{
		"A": {1, 2, 3},
		"B": {10, 20, 30},
	})
	if err != nil {
		t.Fatalf("NewMapContext: %v", err)
	}
	return ctx
}

func TestCompileBindAndEvaluate(t *testing.T) {
	ctx := mapContext(t)

	node, err := ccl.CompileExpression("A + B")
	if err != nil {
		t.Fatalf("CompileExpression: %v", err)
	}
	bound, err := ccl.Bind(node, map[string]int{"A": 0, "B": 1})
	if err != nil {
		t.Fatalf("Bind: %v", err)
	}

	got, err := ccl.Evaluate(bound, ctx)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if got != 11.0 {
		t.Errorf("A + B on row 0: got %v (%T), want 11", got, got)
	}

	if err := ctx.SetRowIndex(2); err != nil {
		t.Fatalf("SetRowIndex: %v", err)
	}
	got, err = ccl.Evaluate(bound, ctx)
	if err != nil {
		t.Fatalf("Evaluate on row 2: %v", err)
	}
	if got != 33.0 {
		t.Errorf("A + B on row 2: got %v, want 33", got)
	}
}

// The whole point of re-exporting CompileError and EvalError as aliases is that
// errors.As works on them from outside the module.
func TestCompileErrorIsMatchable(t *testing.T) {
	_, err := ccl.CompileExpression("SUM(A B)")
	if err == nil {
		t.Fatal("a malformed expression compiled")
	}
	var compileErr *ccl.CompileError
	if !errors.As(err, &compileErr) {
		t.Fatalf("errors.As did not match *ccl.CompileError, got %T: %v", err, err)
	}
}

// Evaluate hands back the underlying failure as it is. It evaluates one node
// against one context, so there is no statement or row for an *EvalError to
// name; the DataTable methods are what wrap a failure in one. The doc comment
// on the re-exported type used to imply otherwise.
func TestEvaluateReturnsTheUnderlyingError(t *testing.T) {
	ctx, err := ccl.NewMapContext(map[string][]any{"A": {1}, "B": {0}})
	if err != nil {
		t.Fatalf("NewMapContext: %v", err)
	}
	node, err := ccl.CompileExpression("A / B")
	if err != nil {
		t.Fatalf("CompileExpression: %v", err)
	}
	bound, err := ccl.Bind(node, map[string]int{"A": 0, "B": 1})
	if err != nil {
		t.Fatalf("Bind: %v", err)
	}

	_, err = ccl.Evaluate(bound, ctx)
	if err == nil {
		t.Fatal("dividing by zero gave no error")
	}
	if !strings.Contains(err.Error(), "division by zero") {
		t.Errorf("error %q does not name the cause", err)
	}
	var evalErr *ccl.EvalError
	if errors.As(err, &evalErr) {
		t.Error("Evaluate now wraps in *EvalError; update the type's doc comment, which says the DataTable methods are what do that")
	}
}

func TestMultilineAndStatementHelpers(t *testing.T) {
	nodes, err := ccl.CompileMultiline("NEW('C') = A + B; A = A * 2")
	if err != nil {
		t.Fatalf("CompileMultiline: %v", err)
	}
	if len(nodes) != 2 {
		t.Fatalf("CompileMultiline gave %d nodes, want 2", len(nodes))
	}

	name, expr, isNew := ccl.GetNewColInfo(nodes[0])
	if !isNew || name != "C" || expr == nil {
		t.Errorf("GetNewColInfo on NEW('C') = …: got (%q, %v, %v)", name, expr != nil, isNew)
	}
	if !ccl.IsNewColNode(nodes[0]) {
		t.Error("IsNewColNode on NEW('C') = … is false")
	}

	if !ccl.IsAssignmentNode(nodes[1]) {
		t.Error("IsAssignmentNode on A = A * 2 is false")
	}
	if target, ok := ccl.GetAssignmentTarget(nodes[1]); !ok || target != "A" {
		t.Errorf("GetAssignmentTarget: got (%q, %v), want (\"A\", true)", target, ok)
	}
	if ccl.GetExpressionNode(nodes[1]) == nil {
		t.Error("GetExpressionNode returned nil for an assignment")
	}

	stmts, err := ccl.CompileMultilineStatements("NEW('C') = A + B; A = A * 2")
	if err != nil {
		t.Fatalf("CompileMultilineStatements: %v", err)
	}
	if len(stmts) != 2 {
		t.Fatalf("CompileMultilineStatements gave %d statements, want 2", len(stmts))
	}
}

func TestIsRowDependent(t *testing.T) {
	rowDependent, err := ccl.CompileExpression("A + 1")
	if err != nil {
		t.Fatalf("CompileExpression: %v", err)
	}
	constant, err := ccl.CompileExpression("1 + 1")
	if err != nil {
		t.Fatalf("CompileExpression: %v", err)
	}

	if !ccl.IsRowDependent(rowDependent) {
		t.Error("IsRowDependent(\"A + 1\") is false")
	}
	if ccl.IsRowDependent(constant) {
		t.Error("IsRowDependent(\"1 + 1\") is true")
	}
}

// RegisterFunction reaches the same registry the evaluator reads. The name is
// deliberately unlikely to collide, because there is no way to unregister.
func TestRegisterFunction(t *testing.T) {
	ccl.RegisterStandardFunctions() // idempotent; exercises the forwarder

	ccl.RegisterFunction("ENGINETESTDOUBLE", func(args ...any) (any, error) {
		if len(args) != 1 {
			return nil, errors.New("ENGINETESTDOUBLE takes one argument")
		}
		// A custom function receives the cell as it is stored — a Go int here,
		// not the float64 the arithmetic operators coerce to.
		n, ok := args[0].(int)
		if !ok {
			return nil, fmt.Errorf("want an int, got %T", args[0])
		}
		return n * 2, nil
	})

	ctx := mapContext(t)
	node, err := ccl.CompileExpression("ENGINETESTDOUBLE(A)")
	if err != nil {
		t.Fatalf("CompileExpression: %v", err)
	}
	bound, err := ccl.Bind(node, map[string]int{"A": 0, "B": 1})
	if err != nil {
		t.Fatalf("Bind: %v", err)
	}
	got, err := ccl.Evaluate(bound, ctx)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	// The function's return value reaches the caller unconverted, so this is an
	// int and not the float64 an arithmetic expression would have produced.
	if got != 2 {
		t.Errorf("ENGINETESTDOUBLE(A) on row 0: got %v (%T), want int 2", got, got)
	}
}

// Both are documented no-ops kept for API compatibility; calling them must stay
// harmless.
func TestResetDepthsAreNoOps(t *testing.T) {
	ccl.ResetEvalDepth()
	ccl.ResetFuncCallDepth()
}
