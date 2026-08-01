package ccl

import (
	"reflect"
	"strings"
	"testing"
)

// Behavior-preservation suite for flatten-ccl-operator-chains.
//
// The flattening must be a pure representation change: for every input, the
// folded AST must produce the same value, the same error (byte-identical
// message), the same evaluation order, and the same IsRowDependent /
// containsRowIndex / Bind results as the nested-binary shape it replaces.
// unfoldChains rewrites fold nodes back into nested binaries so the two
// shapes can be compared directly on the same engine.

// unfoldChains converts every cclFoldChainNode back into the equivalent
// left-nested cclBinaryOpNode spine (test helper; recursion is fine at test
// input sizes).
func unfoldChains(n cclNode) cclNode {
	switch t := n.(type) {
	case *cclFoldChainNode:
		acc := unfoldChains(t.init)
		for i, operand := range t.operands {
			acc = &cclBinaryOpNode{op: t.ops[i], left: acc, right: unfoldChains(operand)}
		}
		return acc
	case *cclBinaryOpNode:
		return &cclBinaryOpNode{op: t.op, left: unfoldChains(t.left), right: unfoldChains(t.right)}
	case *cclChainedComparisonNode:
		vals := make([]cclNode, len(t.values))
		for i, v := range t.values {
			vals[i] = unfoldChains(v)
		}
		return &cclChainedComparisonNode{ops: t.ops, values: vals}
	case *funcCallNode:
		args := make([]cclNode, len(t.args))
		for i, a := range t.args {
			args[i] = unfoldChains(a)
		}
		return &funcCallNode{name: t.name, args: args}
	case *cclAssignmentNode:
		return &cclAssignmentNode{target: t.target, expr: unfoldChains(t.expr)}
	case *cclNewColNode:
		return &cclNewColNode{colName: t.colName, expr: unfoldChains(t.expr)}
	default:
		return n
	}
}

// containsFold reports whether any fold node remains (sanity check for
// unfoldChains itself).
func containsFold(n cclNode) bool {
	switch t := n.(type) {
	case *cclFoldChainNode:
		return true
	case *cclBinaryOpNode:
		return containsFold(t.left) || containsFold(t.right)
	case *cclChainedComparisonNode:
		for _, v := range t.values {
			if containsFold(v) {
				return true
			}
		}
	case *funcCallNode:
		for _, a := range t.args {
			if containsFold(a) {
				return true
			}
		}
	case *cclAssignmentNode:
		return containsFold(t.expr)
	case *cclNewColNode:
		return containsFold(t.expr)
	}
	return false
}

// foldDifferentialCorpus exercises chains across operators, precedences,
// types, error positions, and interactions with '.', ':', comparisons,
// functions, and unary operators.
var foldDifferentialCorpus = []string{
	// pure arithmetic chains, mixed precedence spines
	"1+2+3+4+5",
	"10-2-3",
	"100/5/2",
	"2^3^2",
	"10%4%3",
	"2*3*4",
	"1+2*3+4",
	"2*3+4*5",
	"1+2-3*4/2^2",
	"1*2+3",
	"1+2*3-4/2",
	// parens and unary interspersed
	"(1+2)*(3+4)+(5-6)",
	"1 + -2 + -3",
	"-1 - -2 - -3",
	"+1 + +2 - -3",
	// string concatenation chains and coercion
	"'a' & 'b' & 'c'",
	"'v=' & 1 & 2",
	"1 & 2 & 'x'",
	// boolean chains (no short-circuit!) and comparison→logical
	"true && true && false",
	"true || false || true",
	"true && false || true",
	"false || true && true",
	"1 > 0 && 2 > 1 && 3 > 2",
	"1 < 2 && 3 > 2 || false",
	"1 < 2 < 3 && true",
	// nil in chains
	"nil + 1 + 2",
	"1 + nil + 2",
	"'s' & nil & 'e'",
	// error cases — position and order sensitivity
	"1 + 'x'",
	"1 + 'x' + 2",
	"'x' + 1 + 2",
	"1 + 2 + 3 + 'x'",
	"1/0",
	"1 + 1/0 + 2",
	"1 + 'x' + (1/0)",
	"(1/0) + 'x'",
	"true && 1 && false",
	"1 && 2",
	"true || 1/0 > 0",
	// columns (identifier and bracket-name forms)
	"['a'] + ['b'] + 1",
	"A + B + 1",
	"A + B - A * B",
	"['a'] & '-' & ['b']",
	// functions with chain args, chains of function results
	"IF(1 < 2, 1+2+3, 4+5+6)",
	"ABS(0-5) + ABS(0-6) + 1",
	"IF(A > 0, A, 0-A) + 1 + 2",
	// '.' and ':' interactions (must stay binary; flush interactions)
	"A + B + SUM(A:B)",
	"SUM(A:B) + 1 + 2",
	"@.0 == @.0 && true",
}

func foldTestContext(t *testing.T) *MapContext {
	t.Helper()
	ctx, err := NewMapContext(map[string][]any{
		"a": {5.0, -3.0},
		"b": {2.0, 50.0},
	})
	if err != nil {
		t.Fatal(err)
	}
	return ctx
}

// TestFoldChain_DifferentialAgainstUnfolded is the core no-behavior-change
// guarantee: value, error message, row-dependence, and row-index detection
// must be identical between the folded AST and its unfolded equivalent, both
// pre-Bind and post-Bind.
func TestFoldChain_DifferentialAgainstUnfolded(t *testing.T) {
	ctx := foldTestContext(t)
	colNameMap := map[string]int{"a": 0, "b": 1}

	for _, expr := range foldDifferentialCorpus {
		t.Run(expr, func(t *testing.T) {
			folded, err := CompileExpression(expr)
			if err != nil {
				t.Fatalf("corpus expression failed to compile: %v", err)
			}
			unfolded := unfoldChains(folded)
			if containsFold(unfolded) {
				t.Fatal("unfoldChains left a fold node behind (helper bug)")
			}

			if f, u := IsRowDependent(folded), IsRowDependent(unfolded); f != u {
				t.Errorf("IsRowDependent mismatch: folded=%v unfolded=%v", f, u)
			}
			if f, u := containsRowIndex(folded), containsRowIndex(unfolded); f != u {
				t.Errorf("containsRowIndex mismatch: folded=%v unfolded=%v", f, u)
			}

			compare := func(stage string, fn, un cclNode) {
				fv, fe := Evaluate(fn, ctx)
				uv, ue := Evaluate(un, ctx)
				if (fe == nil) != (ue == nil) {
					t.Fatalf("[%s] error presence mismatch: folded=%v unfolded=%v", stage, fe, ue)
				}
				if fe != nil {
					if fe.Error() != ue.Error() {
						t.Errorf("[%s] error message mismatch:\n folded:   %s\n unfolded: %s", stage, fe, ue)
					}
					return
				}
				if !reflect.DeepEqual(fv, uv) {
					t.Errorf("[%s] value mismatch: folded=%v (%T) unfolded=%v (%T)", stage, fv, fv, uv, uv)
				}
			}

			compare("unbound", folded, unfolded)

			fb, fbErr := Bind(folded, colNameMap)
			ub, ubErr := Bind(unfolded, colNameMap)
			if (fbErr == nil) != (ubErr == nil) {
				t.Fatalf("Bind error presence mismatch: folded=%v unfolded=%v", fbErr, ubErr)
			}
			if fbErr != nil {
				if fbErr.Error() != ubErr.Error() {
					t.Errorf("Bind error message mismatch:\n folded:   %s\n unfolded: %s", fbErr, ubErr)
				}
				return
			}
			compare("bound", fb, ub)
		})
	}
}

// TestFoldChain_GoldenValues pins order-sensitive semantics with exact
// expected values — left association for every operator (including ^, which
// CCL folds left unlike mathematical convention) and non-short-circuiting.
func TestFoldChain_GoldenValues(t *testing.T) {
	ctx := foldTestContext(t)
	cases := []struct {
		expr string
		want any
	}{
		{"1+2+3+4", 10.0},
		{"10-2-3", 5.0},
		{"100/5/2", 10.0},
		{"2^3^2", 64.0}, // (2^3)^2, NOT 2^(3^2)=512
		{"2*3*4", 24.0},
		{"1+2*3+4", 11.0},
		{"10-2*3-1", 3.0},
		{"true && true && false", false},
		{"false || false || true", true},
		{"true && false || true", true}, // (true&&false)||true
		{"1 > 0 && 2 > 1 && 3 > 2", true},
	}
	for _, tc := range cases {
		t.Run(tc.expr, func(t *testing.T) {
			node, err := CompileExpression(tc.expr)
			if err != nil {
				t.Fatalf("compile: %v", err)
			}
			got, err := Evaluate(node, ctx)
			if err != nil {
				t.Fatalf("evaluate: %v", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %v (%T), want %v (%T)", got, got, tc.want, tc.want)
			}
		})
	}
}

// TestFoldChain_NoShortCircuit pins that && / || still evaluate their right
// operands (binary chains never short-circuited; folding must not either).
func TestFoldChain_NoShortCircuit(t *testing.T) {
	ctx := foldTestContext(t)
	for _, expr := range []string{
		"false && (1/0 > 0)",
		"true || (1/0 > 0)",
		"false && (1/0 > 0) && true",
		"true || (1/0 > 0) || false",
	} {
		t.Run(expr, func(t *testing.T) {
			node, err := CompileExpression(expr)
			if err != nil {
				t.Fatalf("compile: %v", err)
			}
			if _, err := Evaluate(node, ctx); err == nil {
				t.Error("expected division-by-zero error to surface (no short-circuit), got nil")
			}
		})
	}
}

// TestFoldChain_StructuralInvariants pins the parser's output shape: single
// operators keep the binary node (zero change for the common case), runs of
// two or more general operators fold, and '.' / ':' always stay binary.
func TestFoldChain_StructuralInvariants(t *testing.T) {
	mustCompile := func(expr string) cclNode {
		t.Helper()
		n, err := CompileExpression(expr)
		if err != nil {
			t.Fatalf("compile %q: %v", expr, err)
		}
		return n
	}

	// Single op → binary node.
	if n, ok := mustCompile("A + B").(*cclBinaryOpNode); !ok {
		t.Errorf("A + B: want *cclBinaryOpNode, got %T", n)
	}
	// ≥2 ops → fold node with the ops in order.
	if n, ok := mustCompile("A + B - 1").(*cclFoldChainNode); !ok {
		t.Errorf("A + B - 1: want *cclFoldChainNode, got %T", n)
	} else if !reflect.DeepEqual(n.ops, []string{"+", "-"}) {
		t.Errorf("A + B - 1: ops = %v, want [+ -]", n.ops)
	}
	// Higher-precedence subexpression is absorbed into the operand, and a
	// single-op run inside it stays binary.
	if n, ok := mustCompile("A + B + C * D").(*cclFoldChainNode); !ok {
		t.Errorf("A + B + C * D: want fold, got %T", n)
	} else if _, ok := n.operands[1].(*cclBinaryOpNode); !ok {
		t.Errorf("A + B + C * D: operands[1] = %T, want *cclBinaryOpNode (C*D)", n.operands[1])
	}
	// ':' stays binary as a fold init (flush interaction).
	if n, ok := mustCompile("A:B + 1 + 2").(*cclFoldChainNode); !ok {
		t.Errorf("A:B + 1 + 2: want fold, got %T", n)
	} else if rangeNode, ok := n.init.(*cclBinaryOpNode); !ok || rangeNode.op != ":" {
		t.Errorf("A:B + 1 + 2: init = %T, want binary ':'", n.init)
	}
	// ':' consumed inside a right operand stays binary; single '+' stays binary.
	if n, ok := mustCompile("SUM(A:B) + 1").(*cclBinaryOpNode); !ok {
		t.Errorf("SUM(A:B) + 1: want *cclBinaryOpNode, got %T", n)
	}
	// '.' stays binary as fold init.
	if n, ok := mustCompile("@.0 & 'x' & 'y'").(*cclFoldChainNode); !ok {
		t.Errorf("@.0 & 'x' & 'y': want fold, got %T", n)
	} else if dotNode, ok := n.init.(*cclBinaryOpNode); !ok || dotNode.op != "." {
		t.Errorf("@.0 & 'x' & 'y': init = %T, want binary '.'", n.init)
	}
	// '^' chains fold (and stay left-associative — value pinned in golden test).
	if _, ok := mustCompile("2^3^2").(*cclFoldChainNode); !ok {
		t.Errorf("2^3^2: want fold")
	}
	// Comparison chains still use the chained-comparison node, not folds.
	if _, ok := mustCompile("1 < A <= B").(*cclChainedComparisonNode); !ok {
		t.Errorf("1 < A <= B: want *cclChainedComparisonNode")
	}
}

// TestFoldChain_LongChainDifferential cross-checks a longer randomized-ish
// chain (mixed +,-,*,& with parenthesized subterms) against the unfolded
// shape, to catch any accumulation bug that tiny corpus items might miss.
func TestFoldChain_LongChainDifferential(t *testing.T) {
	ctx := foldTestContext(t)
	ops := []string{"+", "-", "*", "+", "-"}
	var sb strings.Builder
	sb.WriteString("1")
	for i := range 200 {
		sb.WriteString(ops[i%len(ops)])
		switch i % 4 {
		case 0:
			sb.WriteString("2")
		case 1:
			sb.WriteString("(3-1)")
		case 2:
			sb.WriteString("-2")
		case 3:
			sb.WriteString("['a']")
		}
	}
	expr := sb.String()

	folded, err := CompileExpression(expr)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	fv, fe := Evaluate(folded, ctx)
	uv, ue := Evaluate(unfoldChains(folded), ctx)
	if (fe == nil) != (ue == nil) || (fe != nil && fe.Error() != ue.Error()) {
		t.Fatalf("error mismatch: folded=%v unfolded=%v", fe, ue)
	}
	if fe == nil && !reflect.DeepEqual(fv, uv) {
		t.Errorf("value mismatch: folded=%v unfolded=%v", fv, uv)
	}
}
