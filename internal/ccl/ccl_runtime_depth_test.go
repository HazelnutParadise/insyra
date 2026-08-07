package ccl

import (
	"fmt"
	"testing"
)

func deepBinaryAST(levels int) cclNode {
	var node cclNode = &cclNumberNode{value: 1}
	for i := 0; i < levels; i++ {
		node = &cclBinaryOpNode{
			op:    "+",
			left:  node,
			right: &cclNumberNode{value: 0},
		}
	}
	return node
}

func TestEvaluate_OverDeepASTReturnsExistingDepthError(t *testing.T) {
	if maxEvalDepth != 10000 {
		t.Fatalf("maxEvalDepth = %d, want 10000", maxEvalDepth)
	}
	ctx, err := NewMapContext(map[string][]any{"x": {0.0}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = Evaluate(deepBinaryAST(maxEvalDepth+1), ctx)
	want := fmt.Sprintf("evaluate: maximum recursion depth exceeded (%d), possibly infinite recursion", maxEvalDepth)
	if err == nil || err.Error() != want {
		t.Fatalf("Evaluate over-deep AST error = %v, want %q", err, want)
	}
}

func TestEvaluate_FunctionCallGuardReturnsExistingDepthError(t *testing.T) {
	if maxFuncCallDepth != 20 {
		t.Fatalf("maxFuncCallDepth = %d, want 20", maxFuncCallDepth)
	}
	oldLimit := maxFuncCallDepth
	maxFuncCallDepth = 0
	t.Cleanup(func() { maxFuncCallDepth = oldLimit })

	node, err := CompileExpression("ABS(1)")
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := NewMapContext(map[string][]any{"x": {0.0}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = Evaluate(node, ctx)
	want := "callFunction: maximum function call depth exceeded, possibly recursive function calls"
	if err == nil || err.Error() != want {
		t.Fatalf("Evaluate function-call guard error = %v, want %q", err, want)
	}
}
