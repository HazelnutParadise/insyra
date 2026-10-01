package ccl

import (
	"reflect"
	"strings"
	"testing"
)

// referencedColumnsNames is the four-column table the tests bind against.
var referencedColumnsNames = map[string]int{"a": 0, "b": 1, "c": 2, "d": 3}

// boundForColumns compiles an expression and binds it against a, b, c and d.
func boundForColumns(t *testing.T, expression string) CCLNode {
	t.Helper()
	node, err := CompileExpression(expression)
	if err != nil {
		t.Fatalf("compile %q: %v", expression, err)
	}
	bound, err := Bind(node, referencedColumnsNames)
	if err != nil {
		t.Fatalf("bind %q: %v", expression, err)
	}
	return bound
}

// boundStatement compiles one statement of a script and binds it.
func boundStatement(t *testing.T, script string) CCLNode {
	t.Helper()
	nodes, err := CompileMultiline(script)
	if err != nil {
		t.Fatalf("compile %q: %v", script, err)
	}
	if len(nodes) != 1 {
		t.Fatalf("compile %q: got %d statements, want 1", script, len(nodes))
	}
	bound, err := Bind(nodes[0], referencedColumnsNames)
	if err != nil {
		t.Fatalf("bind %q: %v", script, err)
	}
	return bound
}

func TestReferencedColumns(t *testing.T) {
	tests := []struct {
		expression string
		want       []int
	}{
		{"A + C", []int{0, 2}},
		{"['b'] * 2", []int{1}},
		{"SUM(A:C)", []int{0, 1, 2}},
		{"@.3", []int{0, 1, 2, 3}},
		{"COUNT(@)", []int{0, 1, 2, 3}},
		{"MEDIAN(B) > A.(# - 1)", []int{0, 1}},
		{"#", []int{}},
		{"IF(A > 0, B, C)", []int{0, 1, 2}},
		{"LAG(CUMSUM(D), 1)", []int{3}},
		{"A.(B.0)", []int{0, 1}},
		{"D + A + D", []int{0, 3}},
		{"1 < B <= D", []int{1, 3}},
		{"B:D.2", []int{1, 2, 3}},
	}
	for _, tt := range tests {
		t.Run(tt.expression, func(t *testing.T) {
			got, err := ReferencedColumns(boundForColumns(t, tt.expression), 4)
			if err != nil {
				t.Fatalf("ReferencedColumns: %v", err)
			}
			if len(got) != len(tt.want) || (len(got) > 0 && !reflect.DeepEqual(got, tt.want)) {
				t.Fatalf("ReferencedColumns(%q) = %v, want %v", tt.expression, got, tt.want)
			}
		})
	}
}

func TestReferencedColumnsOfStatements(t *testing.T) {
	tests := []struct {
		script string
		want   []int
	}{
		{"NEW('x') = B", []int{1}},
		{"['a'] = C", []int{2}},
		{"NEW('y') = SUM(A:B) + #", []int{0, 1}},
	}
	for _, tt := range tests {
		t.Run(tt.script, func(t *testing.T) {
			got, err := ReferencedColumns(boundStatement(t, tt.script), 4)
			if err != nil {
				t.Fatalf("ReferencedColumns: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("ReferencedColumns(%q) = %v, want %v", tt.script, got, tt.want)
			}
		})
	}
}

func TestReferencedColumnsNeedsABoundExpression(t *testing.T) {
	for _, expression := range []string{"A + 1", "['a'] + 1", "[A] + 1", "SUM(B)"} {
		t.Run(expression, func(t *testing.T) {
			node, err := CompileExpression(expression)
			if err != nil {
				t.Fatalf("compile: %v", err)
			}
			got, err := ReferencedColumns(node, 4)
			if err == nil {
				t.Fatalf("ReferencedColumns(%q) = %v, want an error", expression, got)
			}
			if !strings.Contains(err.Error(), "not bound") {
				t.Fatalf("error %q does not say the expression is not bound", err)
			}
		})
	}
}

func TestReferencedColumnsLeavesOutColumnsPastTheEnd(t *testing.T) {
	// The caller has refused such a column with FirstColPastEnd before it asks
	// what is read; the ones inside the table are still listed.
	got, err := ReferencedColumns(boundForColumns(t, "A + D + Z"), 4)
	if err != nil {
		t.Fatalf("ReferencedColumns: %v", err)
	}
	if want := []int{0, 3}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	got, err = ReferencedColumns(boundForColumns(t, "SUM(C:E)"), 4)
	if err != nil {
		t.Fatalf("ReferencedColumns: %v", err)
	}
	if want := []int{2}; !reflect.DeepEqual(got, want) {
		t.Fatalf("a range running past the end: got %v, want %v", got, want)
	}
}

func TestStreamRefusal(t *testing.T) {
	refused := []string{
		"MEDIAN(A)",
		"CUMSUM(A) + 1",
		"A.(# - 1)",
		"COUNT(IF(A > 0, A:B, 0))",
	}
	for _, expression := range refused {
		t.Run("refuses "+expression, func(t *testing.T) {
			node, err := CompileExpression(expression)
			if err != nil {
				t.Fatalf("compile: %v", err)
			}
			if StreamRefusal(node) == nil {
				t.Fatalf("StreamRefusal(%q) = nil, want a refusal", expression)
			}
		})
	}
	accepted := []string{
		"SUM(A)",
		"LAG(A, 1)",
		"A > AVG(A)",
		"A.0",
	}
	for _, expression := range accepted {
		t.Run("accepts "+expression, func(t *testing.T) {
			node, err := CompileExpression(expression)
			if err != nil {
				t.Fatalf("compile: %v", err)
			}
			if err := StreamRefusal(node); err != nil {
				t.Fatalf("StreamRefusal(%q) = %v, want nil", expression, err)
			}
		})
	}
}

func TestStreamRefusalMatchesResolveWholeTable(t *testing.T) {
	// StreamRefusal is the check ResolveWholeTable runs before it reads
	// anything, so the two agree about what is refused, statements included.
	for _, script := range []string{
		"NEW('x') = MEDIAN(A)",
		"NEW('x') = SUM(A)",
		"NEW('x') = CUMSUM(A)",
		"['a'] = CUMSUM(A) + 1",
		"['a'] = A.(# - 1)",
	} {
		t.Run(script, func(t *testing.T) {
			nodes, err := CompileMultiline(script)
			if err != nil || len(nodes) != 1 {
				t.Fatalf("compile: %v (%d statements)", err, len(nodes))
			}
			refusal := StreamRefusal(nodes[0])
			_, resolveErr := ResolveWholeTable(nodes[0], 0, []string{"a", "b", "c", "d"}, noBatches)
			if (refusal == nil) != (resolveErr == nil) {
				t.Fatalf("StreamRefusal = %v, ResolveWholeTable = %v", refusal, resolveErr)
			}
			if refusal != nil && refusal.Error() != resolveErr.Error() {
				t.Fatalf("StreamRefusal says %q, ResolveWholeTable says %q", refusal, resolveErr)
			}
		})
	}
}
