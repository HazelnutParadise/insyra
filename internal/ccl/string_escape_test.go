package ccl

import (
	"strings"
	"testing"
)

// CCL-27 (#364): a string literal had no way to hold its own quote character.
// As in Excel, a quote written twice inside a literal stands for one. Before
// this, two quotes in a row always ended one literal and opened another, which
// no expression could use, so the new reading changes no expression that
// compiled.
func TestDoubledQuoteInsideALiteral(t *testing.T) {
	for _, c := range []struct {
		expr string
		want string
	}{
		{`'it''s'`, `it's`},
		{`"say ""hi"""`, `say "hi"`},
		{`''''`, `'`},
		{`""""`, `"`},
		{`'a''''b'`, `a''b`},
		{`'say "hi"'`, `say "hi"`},
		{`"it's"`, `it's`},
		{`'He said "it''s"'`, `He said "it's"`},
		{`''`, ``},
		{`'a''b' & 'c'`, `a'bc`},
		{`CONCAT('it''s', ' ok')`, `it's ok`},
		{`'a\b'`, `a\b`}, // a backslash is an ordinary character
	} {
		got, err := evalExpr(t, c.expr)
		if err != nil {
			t.Errorf("%s: %v", c.expr, err)
			continue
		}
		if got != c.want {
			t.Errorf("%s = %q, want %q", c.expr, got, c.want)
		}
	}

	for _, expr := range []string{`'it''s`, `'''`, `'a' 'b'`, `'it\'s'`} {
		if got, err := evalExpr(t, expr); err == nil {
			t.Errorf("%s = %q, want an error", expr, got)
		}
	}
	// The backslash is literal, so the third quote opens a string never closed.
	if _, err := evalExpr(t, `'it\'s'`); err == nil || !strings.Contains(err.Error(), "unclosed string") {
		t.Errorf(`'it\'s': error %v, want unclosed string`, err)
	}
}

// A column name in brackets follows the same rule.
func TestDoubledQuoteInsideABracketColumnName(t *testing.T) {
	ctx := mapCtx(t, map[string][]any{"O'Brien": {1.0}, `say "hi"`: {2.0}})
	names := map[string]int{"O'Brien": 0, `say "hi"`: 1}
	for _, c := range []struct {
		expr string
		want float64
	}{
		{`['O''Brien']`, 1},
		{`["O'Brien"]`, 1},
		{`['say "hi"']`, 2},
		{`["say ""hi"""]`, 2},
	} {
		node, err := CompileExpression(c.expr)
		if err != nil {
			t.Errorf("%s: %v", c.expr, err)
			continue
		}
		bound, err := Bind(node, names)
		if err != nil {
			t.Errorf("%s: %v", c.expr, err)
			continue
		}
		got, err := Evaluate(bound, ctx)
		if err != nil {
			t.Errorf("%s: %v", c.expr, err)
			continue
		}
		if got != c.want {
			t.Errorf("%s = %v, want %v", c.expr, got, c.want)
		}
	}
}

// A script is split into statements before each is compiled, and the split
// must see a doubled quote as part of the literal, as the tokenizer does.
func TestStatementSplitKeepsDoubledQuotes(t *testing.T) {
	script := "NEW('x') = 'a;b''c\nd'\nNEW('it''s') = 1; NEW('y') = \"e;\"\"f\""
	stmts, err := CompileMultilineStatements(script)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"NEW('x') = 'a;b''c\nd'", "NEW('it''s') = 1", "NEW('y') = \"e;\"\"f\""}
	if len(stmts) != len(want) {
		t.Fatalf("got %d statements, want %d: %#v", len(stmts), len(want), stmts)
	}
	for i, st := range stmts {
		if st.Src != want[i] {
			t.Errorf("statement %d = %q, want %q", i, st.Src, want[i])
		}
	}
	nc, ok := stmts[1].Node.(*cclNewColNode)
	if !ok || nc.colName != "it's" {
		t.Errorf("NEW('it''s') created %#v, want a column named it's", stmts[1].Node)
	}
}
