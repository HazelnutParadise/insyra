package ccl

import (
	"strings"
	"testing"
)

// CCL-31 (#365) and CCL-32 (#366): two places where CCL wrote something into a
// cell instead of saying it could not do what it was asked.

func evalExpr(t *testing.T, expr string) (any, error) {
	t.Helper()
	ctx, err := NewMapContext(map[string][]any{"A": {1}})
	if err != nil {
		t.Fatalf("NewMapContext: %v", err)
	}
	node, err := CompileExpression(expr)
	if err != nil {
		return nil, err
	}
	bound, err := Bind(node, map[string]int{"A": 0})
	if err != nil {
		return nil, err
	}
	return Evaluate(bound, ctx)
}

// A literal too large for a float64 became +Inf, because ParseFloat's error was
// dropped on the floor. Every row then carried an infinity nobody asked for.
func TestNumberLiteralOverflowIsAnError(t *testing.T) {
	huge := strings.Repeat("9", 400)

	_, err := evalExpr(t, huge)
	if err == nil {
		t.Fatal("a 400-digit literal compiled")
	}
	if !strings.Contains(err.Error(), "out of range") && !strings.Contains(err.Error(), "too large") {
		t.Errorf("the error %q does not say the number is out of range", err)
	}

	// The negative form goes through the unary-minus path.
	if _, err := evalExpr(t, "-"+huge); err == nil {
		t.Error("a negative 400-digit literal compiled")
	}
}

// Scientific notation works inside VALUE('1e3') and appears in CCL's own string
// output, but the tokenizer split the literal into 1 and an identifier e5.
func TestScientificNotationLiterals(t *testing.T) {
	tests := []struct {
		expr string
		want float64
	}{
		{expr: "1e5", want: 100000},
		{expr: "1E5", want: 100000},
		{expr: "1e-3", want: 0.001},
		{expr: "1.5e2", want: 150},
		{expr: "2e+3", want: 2000},
		{expr: "1e5 + 1", want: 100001},
	}
	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			got, err := evalExpr(t, tt.expr)
			if err != nil {
				t.Fatalf("%v", err)
			}
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// An identifier that merely starts with e is still an identifier.
func TestScientificNotationDoesNotSwallowIdentifiers(t *testing.T) {
	ctx, err := NewMapContext(map[string][]any{"A": {2}, "E1": {7}})
	if err != nil {
		t.Fatalf("NewMapContext: %v", err)
	}
	node, err := CompileExpression("['E1'] + 1")
	if err != nil {
		t.Fatalf("CompileExpression: %v", err)
	}
	bound, err := Bind(node, map[string]int{"A": 0, "E1": 1})
	if err != nil {
		t.Fatalf("Bind: %v", err)
	}
	got, err := Evaluate(bound, ctx)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if got != 8.0 {
		t.Errorf("got %v, want 8", got)
	}
}

// A format verb that does not match the value used to put Go's own error text
// into the cell: TOSTR(1.5, '%d') wrote "%!d(float64=1.5)".
func TestTOSTR_BadFormatIsAnError(t *testing.T) {
	tests := []struct {
		expr string
		name string
	}{
		{name: "wrong verb", expr: "TOSTR(1.5, '%d')"},
		{name: "no verb", expr: "TOSTR(1, '%')"},
		{name: "too many verbs", expr: "TOSTR(1, '%d %d')"},
		{name: "unknown verb", expr: "TOSTR(1, '%q%y')"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := evalExpr(t, tt.expr)
			if err == nil {
				t.Fatalf("got %q with no error", got)
			}
			if s, ok := got.(string); ok && strings.Contains(s, "%!") {
				t.Errorf("the formatting error reached the cell: %q", s)
			}
		})
	}
}

// The formats that do match still work.
func TestTOSTR_GoodFormats(t *testing.T) {
	tests := []struct {
		expr string
		want string
	}{
		{expr: "TOSTR(1.5, '%.2f')", want: "1.50"},
		{expr: "TOSTR(1.5)", want: "1.5"},
		{expr: "TOSTR('x', '%s!')", want: "x!"},
		{expr: "TOSTR(1, '%v')", want: "1"},
		// A literal percent is not a verb.
		{expr: "TOSTR(50, '%.0f%%')", want: "50%"},
	}
	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			got, err := evalExpr(t, tt.expr)
			if err != nil {
				t.Fatalf("%v", err)
			}
			if got != tt.want {
				t.Errorf("got %v, want %q", got, tt.want)
			}
		})
	}
}

// CCL-36 (#368) asked for rune-level positions in the tokenizer's error: a
// non-ASCII identifier used to report `unexpected character '¸' at position 1`,
// a byte from the middle of the character. ccl-error-reporting already snaps the
// offset to a character boundary and quotes the text there. This keeps it that
// way.
func TestNonASCIIErrorsPointAtACharacter(t *testing.T) {
	tests := []struct {
		expr string
		near string
	}{
		{expr: "中文 + 1", near: `near "中"`},
		{expr: "A + 日本語", near: `near "日"`},
		{expr: "'ok' & 中", near: `near "中"`},
	}
	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			_, err := evalExpr(t, tt.expr)
			if err == nil {
				t.Fatal("a non-ASCII identifier compiled")
			}
			if !strings.Contains(err.Error(), tt.near) {
				t.Errorf("the error %q does not quote the character: want %s", err, tt.near)
			}
			// No stray replacement character from a half-read rune.
			if strings.ContainsRune(err.Error(), '�') {
				t.Errorf("the error %q carries a broken rune", err)
			}
		})
	}
}
