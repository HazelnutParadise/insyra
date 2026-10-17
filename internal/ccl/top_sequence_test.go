package ccl

import (
	"strings"
	"testing"
)

// wholeSequence evaluates expr once on the whole table, which is how a sequence
// function that is the whole expression gives its column: the evaluator reads
// every row of the column itself and hands the outputs back as one []any.
func wholeSequence(t *testing.T, expr string, data map[string][]any) []any {
	t.Helper()
	ctx, err := NewMapContext(data)
	if err != nil {
		t.Fatalf("whole table %s: %v", expr, err)
	}
	if err := ctx.SetRowIndex(0); err != nil {
		t.Fatalf("whole table %s: %v", expr, err)
	}
	node, err := CompileExpression(expr)
	if err != nil {
		t.Fatalf("whole table %s: %v", expr, err)
	}
	v, err := Evaluate(node, ctx)
	if err != nil {
		t.Fatalf("whole table %s: %v", expr, err)
	}
	out, ok := v.([]any)
	if !ok {
		t.Fatalf("whole table %s: got %T, want the column []any", expr, v)
	}
	return out
}

// batchedSequence resolves expr for batched reading, feeds a TopSequence one
// batch at a time and returns its outputs in row order, together with the number
// of passes the resolution made over the batches. The batches Push reads are the
// caller's own, so they are not counted.
func batchedSequence(t *testing.T, expr string, data map[string][]any, size int) ([]any, int) {
	t.Helper()
	colNames := []string{"A", "B"}
	node, err := CompileExpression(expr)
	if err != nil {
		t.Fatalf("batched %s: %v", expr, err)
	}
	batches := splitIntoBatches(t, data, size)
	passes := 0
	resolved, err := ResolveWholeTable(node, len(data["A"]), colNames, batchesOf(batches, &passes))
	if err != nil {
		t.Fatalf("batched %s: %v", expr, err)
	}
	seq, ok, err := NewTopSequence(resolved, len(data["A"]), colNames)
	if err != nil {
		t.Fatalf("batched %s: %v", expr, err)
	}
	if !ok {
		t.Fatalf("batched %s: not a sequence call that is the whole expression", expr)
	}
	var out []any
	for _, batch := range batches {
		values, err := seq.Push(batch)
		if err != nil {
			t.Fatalf("batched %s: %v", expr, err)
		}
		out = append(out, values...)
	}
	rest, err := seq.Flush()
	if err != nil {
		t.Fatalf("batched %s: %v", expr, err)
	}
	return append(out, rest...), passes
}

// sameSequenceRow compares one output of a sequence function. sameResolvedValue
// does, except for a row that is a map: reflect.DeepEqual calls a NaN unequal to
// itself, and streamTestData holds one, so `LAG(@, 1)` carries a row holding NaN
// out of every batch. Maps are therefore walked as well as slices, and every
// leaf goes through sameResolvedValue, which is reflect.DeepEqual for the
// numbers, strings and nils beside it.
func sameSequenceRow(want, got any) bool {
	switch w := want.(type) {
	case []any:
		g, ok := got.([]any)
		if !ok || len(w) != len(g) {
			return false
		}
		for i := range w {
			if !sameSequenceRow(w[i], g[i]) {
				return false
			}
		}
		return true
	case map[string]any:
		g, ok := got.(map[string]any)
		if !ok || len(w) != len(g) {
			return false
		}
		for name, v := range w {
			other, present := g[name]
			if !present || !sameSequenceRow(v, other) {
				return false
			}
		}
		return true
	default:
		return sameResolvedValue(want, got)
	}
}

func TestTopSequenceMatchesTheWholeTable(t *testing.T) {
	data := streamTestData()
	// wantPasses is how many passes over the table resolving the expression may
	// make before the sequence starts: one for the whole-table part inside it,
	// none when it has none. A pass more means the sequence itself was read off
	// the table rather than computed from the batches.
	for _, tc := range []struct {
		expr       string
		wantPasses int
	}{
		{"LAG(A, 1)", 0},
		{"LEAD(A, 3)", 0},
		{"LAG(A, -2)", 0},
		{"LEAD(B, -1)", 0},
		{"DIFF(A)", 0},
		{"PCT_CHANGE(A, 2)", 0},
		{"CUMSUM(A)", 0},
		{"CUMPROD(B)", 0},
		{"CUMMAX(A - AVG(A))", 1},
		{"CUMMIN(B)", 0},
		{"ROLLING_MEAN(A, 4)", 0},
		{"ROLLING_STD(A * 2, 3)", 0},
		{"ROLLING_SUM(B, 30)", 0},
		{"LAG(@, 1)", 0},
		{"LEAD(A:B, 2)", 0},
		{"LAG(#, 1)", 0},
		{"CUMSUM(A + B.3)", 1},
		{"LAG(A, COUNT(A) - 23)", 1},
	} {
		t.Run(tc.expr, func(t *testing.T) {
			want := wholeSequence(t, tc.expr, data)
			for _, size := range []int{1, 4, 7, 100} {
				got, passes := batchedSequence(t, tc.expr, data, size)
				if passes != tc.wantPasses {
					t.Errorf("%s in batches of %d: resolution read the table %d times, want %d", tc.expr, size, passes, tc.wantPasses)
				}
				if len(want) != len(got) {
					t.Errorf("%s in batches of %d: %d outputs, want %d", tc.expr, size, len(got), len(want))
					continue
				}
				for i := range want {
					if !sameSequenceRow(want[i], got[i]) {
						t.Errorf("%s in batches of %d row %d: got %#v, want %#v", tc.expr, size, i, got[i], want[i])
					}
				}
			}
		})
	}
}

func TestTopSequenceIsRefusedElsewhere(t *testing.T) {
	data := streamTestData()
	for _, tc := range []struct {
		expr string
		want string
	}{
		{"CUMSUM(A) + 1", "CUMSUM"},
		{"LAG(CUMSUM(A), 1)", "CUMSUM"},
		{"SUM(CUMSUM(A))", "CUMSUM"},
		{"IF(A > 1, LAG(A, 1), 0)", "LAG"},
		{"CUMSUM(A) == A", "CUMSUM"},
	} {
		node, err := CompileExpression(tc.expr)
		if err != nil {
			t.Fatalf("%s: %v", tc.expr, err)
		}
		batches := splitIntoBatches(t, data, 4)
		passes := 0
		_, err = ResolveWholeTable(node, len(data["A"]), []string{"A", "B"}, batchesOf(batches, &passes))
		if err == nil {
			t.Errorf("%s: no error, want one naming %s", tc.expr, tc.want)
			continue
		}
		if !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: error %q does not name %s", tc.expr, err, tc.want)
		}
		if passes != 0 {
			t.Errorf("%s: read the table %d times before failing, want 0", tc.expr, passes)
		}
	}

	// The right-hand side of a NEW or an assignment is where a sequence
	// function may stand, so these resolve; one buried in arithmetic beside it
	// does not.
	for _, tc := range []struct {
		script string
		refuse bool
		want   string
	}{
		{script: "NEW('c') = CUMSUM(A)"},
		{script: "['A'] = LAG(A, 1)"},
		{script: "NEW('c') = CUMSUM(A) * 2", refuse: true, want: "CUMSUM"},
	} {
		nodes, err := CompileMultiline(tc.script)
		if err != nil {
			t.Fatalf("%s: %v", tc.script, err)
		}
		if len(nodes) != 1 {
			t.Fatalf("%s: %d statements, want 1", tc.script, len(nodes))
		}
		batches := splitIntoBatches(t, data, 4)
		passes := 0
		_, err = ResolveWholeTable(nodes[0], len(data["A"]), []string{"A", "B"}, batchesOf(batches, &passes))
		if tc.refuse {
			if err == nil {
				t.Errorf("%s: no error, want one naming %s", tc.script, tc.want)
				continue
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("%s: error %q does not name %s", tc.script, err, tc.want)
			}
			if passes != 0 {
				t.Errorf("%s: read the table %d times before failing, want 0", tc.script, passes)
			}
			continue
		}
		if err != nil {
			t.Errorf("%s: %v", tc.script, err)
		}
	}
}

func TestNewTopSequenceRefusesWhatItCannotStream(t *testing.T) {
	data := streamTestData()
	colNames := []string{"A", "B"}
	for _, tc := range []struct {
		expr string
		want string
	}{
		{"CUMSUM(1)", "does not change from row to row"},
		{"LAG(A, #)", "must be a constant"},
		{"LAG(A, CUMSUM(A))", "must be a constant"},
		{"ROLLING_MEAN(A, 0)", "window must be > 0"},
	} {
		node, err := CompileExpression(tc.expr)
		if err != nil {
			t.Fatalf("%s: %v", tc.expr, err)
		}
		seq, ok, err := NewTopSequence(node, len(data["A"]), colNames)
		if err == nil {
			t.Errorf("%s: no error, want one saying %q", tc.expr, tc.want)
			continue
		}
		if !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: error %q does not say %q", tc.expr, err, tc.want)
		}
		if !ok {
			t.Errorf("%s: ok is false beside an error, want true", tc.expr)
		}
		if seq != nil {
			t.Errorf("%s: a sequence beside an error", tc.expr)
		}
	}

	// A call with no argument at all has no column to read.
	node, err := CompileExpression("LAG()")
	if err != nil {
		t.Fatalf("LAG(): %v", err)
	}
	if _, _, err := NewTopSequence(node, len(data["A"]), colNames); err == nil {
		t.Error("LAG(): no error, want one about the column argument")
	}

	// Not a sequence call at all: ok is false and there is nothing to say.
	node, err = CompileExpression("A + 1")
	if err != nil {
		t.Fatalf("A + 1: %v", err)
	}
	seq, ok, err := NewTopSequence(node, len(data["A"]), colNames)
	if ok {
		t.Errorf("A + 1: ok is true, want false")
	}
	if err != nil {
		t.Errorf("A + 1: %v", err)
	}
	if seq != nil {
		t.Error("A + 1: a sequence for an expression that is not a sequence call")
	}
}

// TestTopSequenceRefusesAReRegisteredName checks that a built-in name a caller
// has replaced stands down the built-in stream, the way a re-registered
// aggregate does. It cannot run in parallel because it changes the registry.
func TestTopSequenceRefusesAReRegisteredName(t *testing.T) {
	data := streamTestData()
	original, ok := lookupSequenceFunction("LAG")
	if !ok {
		t.Fatal("LAG is not registered")
	}
	t.Cleanup(func() { registerSequenceFunction("LAG", original) })

	RegisterSequenceFunction("LAG", func(args ...[]any) ([]any, error) {
		out := make([]any, len(args[0]))
		copy(out, args[0])
		return out, nil
	})

	node, err := CompileExpression("LAG(A, 1)")
	if err != nil {
		t.Fatalf("LAG(A, 1): %v", err)
	}
	_, ok, err = NewTopSequence(node, len(data["A"]), []string{"A", "B"})
	if err == nil {
		t.Fatal("a re-registered LAG: no error, want one saying it is not supported")
	}
	if !strings.Contains(err.Error(), "LAG") || !strings.Contains(err.Error(), "not supported") {
		t.Errorf("a re-registered LAG: error %q does not say LAG is not supported", err)
	}
	if !ok {
		t.Error("a re-registered LAG: ok is false beside an error, want true")
	}
}
