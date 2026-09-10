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

// Benchmarks for ccl-performance. An aggregate inside a per-row expression was
// recomputed for every row over a fresh copy of the column; a rolling window
// pulled each element out of its interface once per window it appeared in.
func benchPerfContext(b *testing.B, rows int) *MapContext {
	b.Helper()
	a := make([]any, rows)
	s := make([]any, rows)
	for i := range a {
		a[i] = float64(i%1000) + 1
		s[i] = "item-" + string(rune('a'+i%5))
	}
	ctx, err := NewMapContext(map[string][]any{"a": a, "s": s})
	if err != nil {
		b.Fatal(err)
	}
	return ctx
}

func benchPerRow(b *testing.B, ctx *MapContext, expr string, fold bool) {
	b.Helper()
	node, err := CompileExpression(expr)
	if err != nil {
		b.Fatal(err)
	}
	bound, err := Bind(node, ctx.ColNameMap)
	if err != nil {
		b.Fatal(err)
	}
	if fold {
		bound = FoldRowInvariantAggregates(bound, ctx)
	}
	b.ResetTimer()
	for b.Loop() {
		for i := range ctx.Rows {
			if err := ctx.SetRowIndex(i); err != nil {
				b.Fatal(err)
			}
			if _, err := Evaluate(bound, ctx); err != nil {
				b.Fatal(err)
			}
		}
	}
}

func BenchmarkAggregateInRowExpr_Folded(b *testing.B) {
	benchPerRow(b, benchPerfContext(b, 5000), "A / SUM(A)", true)
}

func BenchmarkAggregateInRowExpr_Unfolded(b *testing.B) {
	benchPerRow(b, benchPerfContext(b, 5000), "A / SUM(A)", false)
}

func BenchmarkZScore_Folded(b *testing.B) {
	benchPerRow(b, benchPerfContext(b, 5000), "(A - AVG(A)) / STDEV(A)", true)
}

func BenchmarkStringOperand(b *testing.B) {
	benchPerRow(b, benchPerfContext(b, 5000), "B & 'x'", true)
}

func BenchmarkRegexMatch(b *testing.B) {
	benchPerRow(b, benchPerfContext(b, 5000), "REGEX_MATCH(B, 'item-[a-z]')", true)
}

func BenchmarkRollingMean(b *testing.B) {
	rows := 20000
	col := make([]any, rows)
	for i := range col {
		col[i] = float64(i%1000) + 1
	}
	b.ResetTimer()
	for b.Loop() {
		if _, err := seqRollingMean([][]any{col, {500}}...); err != nil {
			b.Fatal(err)
		}
	}
}
