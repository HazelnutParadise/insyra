package ccl

import (
	"strings"
	"testing"
)

// Benchmarks guarding the operator-chain flattening change
// (flatten-ccl-operator-chains): common-case compile/eval must not regress,
// long-chain eval should improve.

const (
	benchSimpleExpr = "['a'] + ['b'] * 2"
	benchMediumExpr = "IF(['a'] > 0 && ['b'] < 100, ['a'] * 2 + ['b'], ['a'] - ['b'])"
)

func benchChainExpr(terms int) string {
	var sb strings.Builder
	sb.WriteString("1")
	for range terms {
		sb.WriteString("+1")
	}
	return sb.String()
}

func benchContext(b *testing.B) *MapContext {
	b.Helper()
	ctx, err := NewMapContext(map[string][]any{
		"a": {5.0, -3.0, 42.0},
		"b": {2.0, 50.0, 7.0},
	})
	if err != nil {
		b.Fatal(err)
	}
	return ctx
}

func benchCompile(b *testing.B, expr string) {
	b.Helper()
	b.ReportAllocs()
	for b.Loop() {
		if _, err := CompileExpression(expr); err != nil {
			b.Fatal(err)
		}
	}
}

func benchEval(b *testing.B, expr string) {
	b.Helper()
	node, err := CompileExpression(expr)
	if err != nil {
		b.Fatal(err)
	}
	ctx := benchContext(b)
	b.ReportAllocs()
	for b.Loop() {
		if _, err := Evaluate(node, ctx); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCompile_Simple(b *testing.B)    { benchCompile(b, benchSimpleExpr) }
func BenchmarkCompile_Medium(b *testing.B)    { benchCompile(b, benchMediumExpr) }
func BenchmarkCompile_Chain100(b *testing.B)  { benchCompile(b, benchChainExpr(100)) }
func BenchmarkCompile_Chain5000(b *testing.B) { benchCompile(b, benchChainExpr(5000)) }

func BenchmarkEval_Simple(b *testing.B)    { benchEval(b, benchSimpleExpr) }
func BenchmarkEval_Medium(b *testing.B)    { benchEval(b, benchMediumExpr) }
func BenchmarkEval_Chain100(b *testing.B)  { benchEval(b, benchChainExpr(100)) }
func BenchmarkEval_Chain5000(b *testing.B) { benchEval(b, benchChainExpr(5000)) }
