package ccl

import (
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"
)

// sliceRowContext is a MapContext whose whole row is a []any in column order.
// DataTable's and parquet's contexts give GetRowAt that shape; MapContext gives
// a map instead, so without this a comparison of `@.19` would be about that
// difference rather than about the row.
type sliceRowContext struct {
	*MapContext
	colNames []string
}

func (c *sliceRowContext) GetRowAt(rowIndex int) (any, error) {
	if rowIndex < 0 || rowIndex >= c.Rows {
		return nil, fmt.Errorf("row index %d out of range (total rows: %d)", rowIndex, c.Rows)
	}
	row := make([]any, len(c.colNames))
	for i, name := range c.colNames {
		row[i] = c.Data[name][rowIndex]
	}
	return row, nil
}

// batchesMaxPasses is how many passes a resolution may make over the batches.
// No expression needs more than a handful, so the limit is here to turn a
// resolution that cannot substitute what it computed into a failure instead of
// a loop over the table that never ends.
const batchesMaxPasses = 10

// batchesOf reads the given batches in order, counting how many passes were
// made over them.
func batchesOf(list []*batchContext, passes *int) Batches {
	return func(yield func(batch GlobalRowContext) error) error {
		*passes++
		if *passes > batchesMaxPasses {
			return fmt.Errorf("read the table more than %d times, so a computed part was not substituted", batchesMaxPasses)
		}
		for _, b := range list {
			if err := yield(b); err != nil {
				return err
			}
		}
		return nil
	}
}

// streamTestData is 25 rows of two columns: A counts 1 to 25 with a blank in
// row 4 and a NaN in row 9, B holds (i*7)%11.
func streamTestData() map[string][]any {
	a := make([]any, 25)
	b := make([]any, 25)
	for i := range a {
		switch i {
		case 3:
			a[i] = nil
		case 8:
			a[i] = math.NaN()
		default:
			a[i] = float64(i + 1)
		}
		b[i] = int64((i * 7) % 11)
	}
	return map[string][]any{"A": a, "B": b}
}

// evalWhole evaluates expr once per row of the whole table.
func evalWhole(t *testing.T, expr string, data map[string][]any) []any {
	t.Helper()
	ctx, err := NewMapContext(data)
	if err != nil {
		t.Fatalf("whole table %s: %v", expr, err)
	}
	node, err := CompileExpression(expr)
	if err != nil {
		t.Fatalf("whole table %s: %v", expr, err)
	}
	whole := &sliceRowContext{MapContext: ctx, colNames: ctx.ColNames}
	out := make([]any, 0, ctx.Rows)
	for i := 0; i < ctx.Rows; i++ {
		if err := whole.SetRowIndex(i); err != nil {
			t.Fatalf("whole table %s row %d: %v", expr, i, err)
		}
		v, err := Evaluate(node, whole)
		if err != nil {
			t.Fatalf("whole table %s row %d: %v", expr, i, err)
		}
		out = append(out, v)
	}
	return out
}

// evalBatched resolves expr for batched reading, then evaluates it once per row
// of every batch, and reports how many passes the resolution made.
func evalBatched(t *testing.T, expr string, data map[string][]any, size int) ([]any, int) {
	t.Helper()
	node, err := CompileExpression(expr)
	if err != nil {
		t.Fatalf("batched %s: %v", expr, err)
	}
	batches := splitIntoBatches(t, data, size)
	passes := 0
	resolved, err := ResolveWholeTable(node, len(data["A"]), []string{"A", "B"}, batchesOf(batches, &passes))
	if err != nil {
		t.Fatalf("batched %s: %v", expr, err)
	}
	var out []any
	for _, batch := range batches {
		for i := 0; i < batch.GetRowCount(); i++ {
			if err := batch.SetRowIndex(i); err != nil {
				t.Fatalf("batched %s row %d: %v", expr, i, err)
			}
			v, err := Evaluate(resolved, batch)
			if err != nil {
				t.Fatalf("batched %s row %d: %v", expr, i, err)
			}
			out = append(out, v)
		}
	}
	return out, passes
}

// sameResolvedValue compares two cell values, treating a NaN as equal only to a
// NaN, and comparing a []any row or range element by element.
func sameResolvedValue(want, got any) bool {
	switch w := want.(type) {
	case float64:
		g, ok := got.(float64)
		if !ok {
			return false
		}
		if math.IsNaN(w) || math.IsNaN(g) {
			return math.IsNaN(w) && math.IsNaN(g)
		}
		return w == g
	case []any:
		g, ok := got.([]any)
		if !ok || len(w) != len(g) {
			return false
		}
		for i := range w {
			if !sameResolvedValue(w[i], g[i]) {
				return false
			}
		}
		return true
	default:
		return reflect.DeepEqual(want, got)
	}
}

// compareResolvedRows reports every row where a batched result differs from the
// whole table's.
func compareResolvedRows(t *testing.T, expr string, size int, want, got []any) {
	t.Helper()
	if len(want) != len(got) {
		t.Errorf("%s in batches of %d: %d results, want %d", expr, size, len(got), len(want))
		return
	}
	for i := range want {
		if !sameResolvedValue(want[i], got[i]) {
			t.Errorf("%s in batches of %d row %d: got %#v, want %#v", expr, size, i, got[i], want[i])
		}
	}
}

// evalResolvedError resolves expr for batched reading, which must succeed, and
// returns the error evaluating the result raises on its first row.
func evalResolvedError(t *testing.T, expr string, data map[string][]any) error {
	t.Helper()
	node, err := CompileExpression(expr)
	if err != nil {
		t.Fatalf("batched %s: %v", expr, err)
	}
	batches := splitIntoBatches(t, data, 4)
	passes := 0
	resolved, err := ResolveWholeTable(node, len(data["A"]), []string{"A", "B"}, batchesOf(batches, &passes))
	if err != nil {
		t.Fatalf("batched %s: resolution failed with %v", expr, err)
	}
	if len(batches) == 0 || batches[0].GetRowCount() == 0 {
		t.Fatalf("batched %s: no row to evaluate", expr)
	}
	if err := batches[0].SetRowIndex(0); err != nil {
		t.Fatalf("batched %s: %v", expr, err)
	}
	_, err = Evaluate(resolved, batches[0])
	if err == nil {
		t.Fatalf("batched %s: no error on the first row, want one", expr)
	}
	return err
}

// TestResolveReplacesPartsInsideARange pins the substitution reaching inside a
// `:` node. batchesOf refuses the pass after batchesMaxPasses, so a part the
// replacement cannot reach fails here instead of reading the table forever.
func TestResolveReplacesPartsInsideARange(t *testing.T) {
	data := streamTestData()
	exprs := []string{
		"SUM(A.(MIN(B):MAX(B)))",
		"SUM(A.((B.0):(B.1)))",
		"A > SUM(A.(MIN(B):MAX(B)))",
	}
	for _, size := range []int{1, 4, 7, 100} {
		for _, expr := range exprs {
			want := evalWhole(t, expr, data)
			got, _ := evalBatched(t, expr, data, size)
			compareResolvedRows(t, expr, size, want, got)
		}
	}
}

func TestResolveReversedRowRangeIsEmpty(t *testing.T) {
	data := streamTestData()
	for _, size := range []int{1, 4, 7, 100} {
		for _, expr := range []string{"SUM(A.5:2)", "COUNT(A.20:3)"} {
			want := evalWhole(t, expr, data)
			got, _ := evalBatched(t, expr, data, size)
			compareResolvedRows(t, expr, size, want, got)
		}
	}
}

func TestResolveDefersAPartThatFails(t *testing.T) {
	oneRow := map[string][]any{"A": {1.0}, "B": {int64(2)}}
	data := streamTestData()

	// The whole table never evaluates the part that cannot be computed there,
	// so resolving must not fail the expression either.
	for _, tc := range []struct {
		expr string
		data map[string][]any
	}{
		{"IF(COUNT(A) >= 2, A > STDEV(A), TRUE)", oneRow},
		{"IF(COUNT(A) > 50, A.50, 0)", data},
		{"COUNT(A) > 50 && A > A.50", data},
		{"IF(COUNT(A) > 50, AVG(A - A.50), 1)", data},
	} {
		want := evalWhole(t, tc.expr, tc.data)
		got, _ := evalBatched(t, tc.expr, tc.data, 4)
		compareResolvedRows(t, tc.expr, 4, want, got)
	}

	// Evaluated where they are reached, the same parts fail as they do on the
	// whole table.
	for _, tc := range []struct {
		expr string
		data map[string][]any
		want string
	}{
		{"A > STDEV(A)", oneRow, "STDEV requires at least 2 numeric values"},
		{"A > A.50", data, "out of range"},
	} {
		if err := evalResolvedError(t, tc.expr, tc.data); !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: error %q does not say %q", tc.expr, err, tc.want)
		}
	}
}

func TestResolveRefusesAColumnRangeInsideAnAggregateArgument(t *testing.T) {
	data := streamTestData()
	for _, expr := range []string{"COUNT(IF(A > 0, A:B, 0))", "SUM(A:B * 2)", "AVG(A - COUNT(IF(A > 0, A:B, 0)))"} {
		node, err := CompileExpression(expr)
		if err != nil {
			t.Fatalf("%s: %v", expr, err)
		}
		batches := splitIntoBatches(t, data, 4)
		passes := 0
		_, err = ResolveWholeTable(node, len(data["A"]), []string{"A", "B"}, batchesOf(batches, &passes))
		if err == nil {
			t.Errorf("%s: no error, want one naming the column range", expr)
			continue
		}
		if !strings.Contains(err.Error(), "column range") {
			t.Errorf("%s: error %q does not name a column range", expr, err)
		}
		if passes != 0 {
			t.Errorf("%s: read the table %d times before failing, want 0", expr, passes)
		}
	}

	// A range the aggregate is handed directly, or that names the columns a
	// row access reads, is expanded by the resolution rather than refused.
	// The same holds at every depth: a nested aggregate takes its own range
	// argument directly, and a `.` takes its range on the left, wherever they sit.
	for _, expr := range []string{
		"SUM(A:B)", "A:B.3", "COUNT(A:B) + 1", "SUM(A:B, #)", "IF(A > 0, A:B, 0)",
		"AVG(A - SUM(A:B))", "AVG(A - SUM(A:B.0))", "A > AVG(A - SUM(A:B))",
	} {
		node, err := CompileExpression(expr)
		if err != nil {
			t.Fatalf("%s: %v", expr, err)
		}
		batches := splitIntoBatches(t, data, 4)
		passes := 0
		if _, err := ResolveWholeTable(node, len(data["A"]), []string{"A", "B"}, batchesOf(batches, &passes)); err != nil {
			t.Errorf("%s: %v", expr, err)
		}
	}
}

func TestResolveRefusesOnAnEmptyTable(t *testing.T) {
	data := map[string][]any{"A": {}, "B": {}}
	for _, tc := range []struct {
		expr string
		want string
	}{
		{"LAG(A, 1)", "LAG"},
		{"MEDIAN(A)", "MEDIAN"},
		{"A.(# - 1)", "row reference"},
		{"COUNT(IF(A > 0, A:B, 0))", "column range"},
	} {
		node, err := CompileExpression(tc.expr)
		if err != nil {
			t.Fatalf("%s: %v", tc.expr, err)
		}
		batches := splitIntoBatches(t, data, 4)
		passes := 0
		_, err = ResolveWholeTable(node, 0, []string{"A", "B"}, batchesOf(batches, &passes))
		if err == nil {
			t.Errorf("%s on an empty table: no error, want one naming %s", tc.expr, tc.want)
			continue
		}
		if !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s on an empty table: error %q does not name %s", tc.expr, err, tc.want)
		}
		if passes != 0 {
			t.Errorf("%s on an empty table: read the table %d times before failing, want 0", tc.expr, passes)
		}
	}
}

func TestResolveFixedRowsMatchesTheWholeTable(t *testing.T) {
	data := streamTestData()
	exprs := []string{
		"A == A.0",
		"A.17",
		"B.24",
		"@.19",
		"A:B.3",
		"A.3:20",
		"A + B.(24 - 1)",
		"#",
		"A.#",
		"# * 2",
	}
	for _, size := range []int{1, 4, 7, 100} {
		for _, expr := range exprs {
			want := evalWhole(t, expr, data)
			got, _ := evalBatched(t, expr, data, size)
			if len(want) != len(got) {
				t.Errorf("%s in batches of %d: %d results, want %d", expr, size, len(got), len(want))
				continue
			}
			for i := range want {
				if !sameResolvedValue(want[i], got[i]) {
					t.Errorf("%s in batches of %d row %d: got %#v, want %#v", expr, size, i, got[i], want[i])
				}
			}
		}
	}
}

func TestResolveReadsOnlyWhenItMust(t *testing.T) {
	data := streamTestData()
	for _, tc := range []struct {
		expr string
		want int
	}{
		{"A > 1", 0},
		{"#", 0},
		{"A.#", 0},
		{"A.17", 1},
		{"A.0 + B.24", 1},
	} {
		_, passes := evalBatched(t, tc.expr, data, 4)
		if passes != tc.want {
			t.Errorf("%s read the table %d times, want %d", tc.expr, passes, tc.want)
		}
	}
}

func TestResolveRefusesWhatItCannotCompute(t *testing.T) {
	data := streamTestData()
	for _, tc := range []struct {
		expr string
		want string
	}{
		{"LAG(A, 1)", "LAG"},
		{"MEDIAN(A)", "MEDIAN"},
		{"A.(# - 1)", "row reference"},
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
}

func TestResolveEmptyTable(t *testing.T) {
	data := streamTestData()
	node, err := CompileExpression("A.17")
	if err != nil {
		t.Fatal(err)
	}
	batches := splitIntoBatches(t, data, 4)
	passes := 0
	resolved, err := ResolveWholeTable(node, 0, []string{"A", "B"}, batchesOf(batches, &passes))
	if err != nil {
		t.Fatalf("empty table: %v", err)
	}
	if resolved != node {
		t.Errorf("empty table: returned %T, want the node it was given", resolved)
	}
	if passes != 0 {
		t.Errorf("empty table: read the table %d times, want 0", passes)
	}
}

func TestResolveAggregatesMatchesTheWholeTable(t *testing.T) {
	data := streamTestData()
	exprs := []string{
		"A > AVG(A)",
		"A - AVG(A)",
		"SUM(A) / COUNT(A)",
		"VAR(A) + STDEVP(B)",
		"STDEV(A)",
		"VARP(B)",
		"MAX(A) - MIN(B)",
		"AVG(A - AVG(A))",
		"AVG(A - SUM(A:B))",
		"AVG(A - SUM(A:B.0))",
		"SUM(A:B)",
		"COUNT(@)",
		"SUM(A, B, 3)",
		"SUM(A.3:20)",
		"IF(A > AVG(A), 1, 0)",
		"# * SUM(A)",
		"A + B.(COUNT(A) - 1)",
		"SUM()",
		"SUM(A, #)",
	}
	for _, size := range []int{1, 4, 7, 100} {
		for _, expr := range exprs {
			want := evalWhole(t, expr, data)
			got, _ := evalBatched(t, expr, data, size)
			if len(want) != len(got) {
				t.Errorf("%s in batches of %d: %d results, want %d", expr, size, len(got), len(want))
				continue
			}
			for i := range want {
				if !sameResolvedValue(want[i], got[i]) {
					t.Errorf("%s in batches of %d row %d: got %#v, want %#v", expr, size, i, got[i], want[i])
				}
			}
		}
	}
}

func TestResolveAggregatesShareReads(t *testing.T) {
	data := streamTestData()
	for _, tc := range []struct {
		expr string
		want int
	}{
		{"A > AVG(A)", 1},
		{"VAR(A)", 2},
		{"SUM(A) + MAX(B)", 1},
		{"AVG(A - AVG(A))", 2},
		{"SUM(A:B)", 2},
		{"SUM()", 0},
		{"SUM(A, #)", 0},
	} {
		_, passes := evalBatched(t, tc.expr, data, 4)
		if passes != tc.want {
			t.Errorf("%s read the table %d times, want %d", tc.expr, passes, tc.want)
		}
	}
}

func TestResolveAggregateErrorIsReportedWhereEvaluated(t *testing.T) {
	data := map[string][]any{"A": {1.0}, "B": {2.0}}
	err := evalResolvedError(t, "VAR(A)", data)
	if !strings.Contains(err.Error(), "VAR requires at least 2 numeric values") {
		t.Errorf("VAR of one value: error %q does not say VAR requires at least 2 numeric values", err)
	}
}
