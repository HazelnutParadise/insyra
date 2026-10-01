package parquet

import (
	"bytes"
	"context"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/HazelnutParadise/insyra"
)

// FilterWithCCL reads a file in batches, so an expression that reads beyond the
// current row — an aggregate, the row index #, a fixed row — used to be answered
// one batch at a time. These tests hold its answer to the one the same
// expression gives on the file loaded with Read and evaluated on the table, cell
// by cell, and pin the expressions that cannot be computed over the file yet.

const (
	// wholeFileRows is how many rows writeWholeFileFixture holds: cclBatchSize,
	// cclBatchSize and the rest, so an answer computed one batch at a time
	// cannot pass for one computed over the file.
	wholeFileRows = 2500

	// wholeFileMissingRow and wholeFileNaNRow are the rows whose A is a missing
	// value and a NaN, counted the way # counts rows. They come back as they
	// were written, so an aggregate reading A meets the same values here as on
	// the loaded table.
	wholeFileMissingRow = 10
	wholeFileNaNRow     = 20
)

// writeWholeFileFixture writes a file of wholeFileRows rows whose column A
// counts up from 1 and whose column B repeats a short cycle, and returns its
// path.
func writeWholeFileFixture(t *testing.T) string {
	t.Helper()
	a := make([]any, wholeFileRows)
	b := make([]any, wholeFileRows)
	for i := range a {
		switch i {
		case wholeFileMissingRow:
			a[i] = nil
		case wholeFileNaNRow:
			a[i] = math.NaN()
		default:
			a[i] = float64(i + 1)
		}
		b[i] = int64((i * 7) % 11)
	}
	dt := insyra.NewDataTable(
		insyra.NewDataList(a...).SetName("A"),
		insyra.NewDataList(b...).SetName("B"),
	)
	path := filepath.Join(t.TempDir(), "whole-file.parquet")
	if err := Write(dt, path); err != nil {
		t.Fatalf("writing the whole-file fixture: %v", err)
	}
	return path
}

// cclKeep reports whether a value the filter expression evaluated to counts as
// passing a row, by FilterWithCCL's own rule.
func cclKeep(val any) bool {
	switch v := val.(type) {
	case bool:
		return v
	case float64:
		return v != 0
	case int:
		return v != 0
	case int64:
		return v != 0
	default:
		return val != nil
	}
}

// expectedFilter evaluates expr on the file loaded with Read and returns the
// [A, B] of every row the filter keeps, which is what FilterWithCCL has to
// return for the same file and expression.
func expectedFilter(t *testing.T, path, expr string) [][]any {
	t.Helper()
	dt, err := Read(context.Background(), path, ReadOptions{})
	if err != nil {
		t.Fatalf("Read(%s): %v", path, err)
	}
	dt.AddColUsingCCL("__keep", expr)
	if dt.Err() != nil {
		t.Fatalf("AddColUsingCCL(%q) on the loaded table: %v", expr, dt.Err())
	}

	keep := dt.GetColByName("__keep").Data()
	a := dt.GetColByName("A").Data()
	b := dt.GetColByName("B").Data()

	var kept [][]any
	for i, val := range keep {
		if !cclKeep(val) {
			continue
		}
		kept = append(kept, []any{a[i], b[i]})
	}
	return kept
}

// filteredRows turns what FilterWithCCL returned into the same [A, B] shape
// expectedFilter produces.
func filteredRows(t *testing.T, res *insyra.DataTable) [][]any {
	t.Helper()
	if res.Err() != nil {
		t.Fatalf("FilterWithCCL's table carries an error: %v", res.Err())
	}
	a := res.GetColByNumber(0).Data()
	b := res.GetColByNumber(1).Data()
	if len(a) != len(b) {
		t.Fatalf("column A holds %d values and column B holds %d", len(a), len(b))
	}
	kept := make([][]any, len(a))
	for i := range a {
		kept[i] = []any{a[i], b[i]}
	}
	return kept
}

// sameCell compares one cell: a float64 by its bits, so a value a rounding step
// away is a difference, two NaNs are the same value, and nil is only the same
// as nil.
func sameCell(got, want any) bool {
	if want == nil {
		return got == nil
	}
	switch w := want.(type) {
	case float64:
		g, ok := got.(float64)
		if !ok {
			return false
		}
		if math.IsNaN(g) && math.IsNaN(w) {
			return true
		}
		return math.Float64bits(g) == math.Float64bits(w)
	case int64:
		g, ok := got.(int64)
		return ok && g == w
	default:
		return false
	}
}

// sameRows reports whether two runs of [A, B] cells are the same, cell by cell,
// naming the first that is not.
func sameRows(t *testing.T, got, want [][]any) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("kept %d rows, want %d", len(got), len(want))
	}
	for i := range want {
		for c := range want[i] {
			if !sameCell(got[i][c], want[i][c]) {
				t.Fatalf("row %d column %d = %v (%T), want %v (%T)",
					i, c, got[i][c], got[i][c], want[i][c], want[i][c])
			}
		}
	}
}

// wholeFileExpressions are the filter expressions FilterWithCCL has to answer as
// the loaded table answers them: an aggregate over the file, the file's own row
// index, a fixed row, a fixed row range, an aggregate nested in another, one
// over a column range, one fed an argument that varies from row to row, one
// taking several columns, and one joined to the row index.
var wholeFileExpressions = []string{
	"A > AVG(A)",
	"A >= MAX(A) - 10",
	"# == 0",
	"# > 2490",
	"A == A.1500",
	"A > AVG(A) + STDEV(A)",
	"A * 2 > SUM(A:B) / COUNT(A:B)",
	"A > AVG(A - AVG(A))",
	// The average of the centred column is zero whichever rows it is computed
	// over, so the case above cannot tell a whole-file answer from a per-batch
	// one. This one moves the threshold: the file's average of A - 1000 is
	// 250.5, while each batch's own is far below its smallest value.
	"A > AVG(A - 1000)",
	"SUM(A, B) > 0 && # < 3",
	"A > SUM(A.0:1999) / 2000",
	"B == MIN(B)",
}

func TestFilterWithCCLMatchesTheLoadedTable(t *testing.T) {
	path := writeWholeFileFixture(t)

	for _, expr := range wholeFileExpressions {
		t.Run(expr, func(t *testing.T) {
			want := expectedFilter(t, path, expr)
			res, err := FilterWithCCL(context.Background(), path, expr)
			if err != nil {
				t.Fatalf("FilterWithCCL(%q): %v", expr, err)
			}
			sameRows(t, filteredRows(t, res), want)
		})
	}
}

// applyScripts are the scripts ApplyCCL has to answer as the loaded table
// answers them: a statement reading a column an earlier one created, the file's
// own row index, an assignment replacing an existing column, and a statement
// reading a fixed row.
var applyScripts = []string{
	"NEW('c') = A - AVG(A); NEW('d') = ['c'] / SUM(['c'])",
	"NEW('i') = #",
	"['A'] = A - MIN(A)",
	"NEW('e') = B.2499 + COUNT(@)",
}

// expectedApply evaluates script on the file loaded with Read and returns the
// table it leaves, which is what ApplyCCL has to write for the same file and
// script.
func expectedApply(t *testing.T, path, script string) *insyra.DataTable {
	t.Helper()
	dt, err := Read(context.Background(), path, ReadOptions{})
	if err != nil {
		t.Fatalf("Read(%s): %v", path, err)
	}
	dt.ExecuteCCL(script)
	if dt.Err() != nil {
		t.Fatalf("ExecuteCCL(%q) on the loaded table: %v", script, dt.Err())
	}
	return dt
}

// sameTableColumns reports whether got holds what want does, column by column
// by name and cell by cell, naming the first that is not.
func sameTableColumns(t *testing.T, got, want *insyra.DataTable, script string) {
	t.Helper()
	if got.Err() != nil {
		t.Fatalf("the table ApplyCCL wrote carries an error: %v", got.Err())
	}
	gotRows, gotCols := got.Size()
	wantRows, wantCols := want.Size()
	if gotRows != wantRows || gotCols != wantCols {
		t.Fatalf("ApplyCCL(%q) wrote %dx%d, the loaded table is %dx%d",
			script, gotRows, gotCols, wantRows, wantCols)
	}
	for _, name := range want.ColNames() {
		gotCol := got.GetColByName(name)
		if gotCol == nil {
			t.Fatalf("ApplyCCL(%q) wrote no column %q", script, name)
		}
		gotData := gotCol.Data()
		wantData := want.GetColByName(name).Data()
		if len(gotData) != len(wantData) {
			t.Fatalf("ApplyCCL(%q) column %q holds %d values, want %d",
				script, name, len(gotData), len(wantData))
		}
		for i, w := range wantData {
			if !sameCell(gotData[i], w) {
				t.Fatalf("ApplyCCL(%q) column %q row %d = %v (%T), want %v (%T)",
					script, name, i, gotData[i], gotData[i], w, w)
			}
		}
	}
}

func TestApplyCCLMatchesTheLoadedTable(t *testing.T) {
	for _, script := range applyScripts {
		t.Run(script, func(t *testing.T) {
			// A file of its own per script: ApplyCCL writes back to the path it
			// was given, and the expectation is the file before it was touched.
			want := expectedApply(t, writeWholeFileFixture(t), script)

			path := writeWholeFileFixture(t)
			if err := ApplyCCL(context.Background(), path, script); err != nil {
				t.Fatalf("ApplyCCL(%q): %v", script, err)
			}
			got, err := Read(context.Background(), path, ReadOptions{})
			if err != nil {
				t.Fatalf("Read after ApplyCCL(%q): %v", script, err)
			}
			sameTableColumns(t, got, want, script)
		})
	}
}

// statementScripts are the scripts ApplyCCL has to answer as the loaded table
// answers them, where a statement reads what the statements before it wrote: an
// assignment to a column of the file, an assignment reading a column a NEW
// created, a value read through the whole table (@ and @.#), and a fixed row of
// a created column. A per-row read and a whole-file read of the same script
// have to agree, which is the rule ExecuteCCL already follows.
var statementScripts = []string{
	"['A'] = A * 2; NEW('b') = A",
	"['A'] = A - 1000; NEW('b') = A - MIN(A)",
	"NEW('c') = A * 2; NEW('d') = COUNT(@.#)",
	"NEW('c') = A * 2; ['A'] = ['c'] + 1; NEW('e') = A + ['c']",
	"['A'] = 7; NEW('f') = A + #",
	"NEW('c') = A * 2; NEW('g') = COUNT(@)",
	"NEW('c') = A * 2; NEW('h') = ['c'].1500",
}

func TestApplyCCLStatementsSeeEarlierStatements(t *testing.T) {
	for _, script := range statementScripts {
		t.Run(script, func(t *testing.T) {
			// A file of its own per script: ApplyCCL writes back to the path it
			// was given, and the expectation is the file before it was touched.
			want := expectedApply(t, writeWholeFileFixture(t), script)

			path := writeWholeFileFixture(t)
			if err := ApplyCCL(context.Background(), path, script); err != nil {
				t.Fatalf("ApplyCCL(%q): %v", script, err)
			}
			got, err := Read(context.Background(), path, ReadOptions{})
			if err != nil {
				t.Fatalf("Read after ApplyCCL(%q): %v", script, err)
			}
			sameTableColumns(t, got, want, script)
		})
	}
}

func TestApplyCCLRefusesWhatItCannotComputeYet(t *testing.T) {
	path := writeWholeFileFixture(t)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading the fixture: %v", err)
	}

	// A statement FilterWithCCL would refuse is refused here too, and it is
	// refused before anything is written, so the file is left as it was. MEDIAN,
	// where CUMSUM was before it: a sequence function that is the whole right
	// hand side is computed batch by batch now.
	const script = "NEW('c') = MEDIAN(A)"
	err = ApplyCCL(context.Background(), path, script)
	if err == nil {
		t.Fatalf("ApplyCCL(%q) wrote the file instead of refusing it", script)
	}
	if !strings.Contains(err.Error(), "MEDIAN") {
		t.Errorf("error %q does not name %q", err, "MEDIAN")
	}

	after, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatalf("reading the fixture back: %v", readErr)
	}
	if !bytes.Equal(before, after) {
		t.Errorf("the file changed (%d bytes before, %d after) although the script was refused",
			len(before), len(after))
	}
}

// TestCCLErrorsNameTheFileRow holds the row an error names to the file's row:
// the expression fails only at the file's row 1500, which arrives as the second
// batch's own row 500, so a message counted from the batch would point at the
// wrong row of the data. The same expression fails the same way on the loaded
// table, so what is being checked is the numbering and not the expression.
func TestCCLErrorsNameTheFileRow(t *testing.T) {
	path := writeWholeFileFixture(t)

	const expr = "IF(# == 1500, 1 / 'x', 1) > 0"

	// The expression is a valid one that fails at one row on the loaded table
	// too, which is what makes it a fair comparison.
	dt, err := Read(context.Background(), path, ReadOptions{})
	if err != nil {
		t.Fatalf("Read(%s): %v", path, err)
	}
	dt.AddColUsingCCL("__keep", expr)
	if dt.Err() == nil {
		t.Fatalf("AddColUsingCCL(%q) on the loaded table did not fail, so it does not fail at one row either", expr)
	}

	res, err := FilterWithCCL(context.Background(), path, expr)
	if err == nil {
		t.Fatalf("FilterWithCCL(%q) kept %d rows instead of failing at row 1500", expr, len(filteredRows(t, res)))
	}
	if res != nil {
		t.Errorf("FilterWithCCL(%q) returned a table as well as the error", expr)
	}
	if !strings.Contains(err.Error(), "row 1500") {
		t.Errorf("error %q does not name the file's row 1500", err)
	}
}

func TestFilterWithCCLRefusesWhatItCannotComputeYet(t *testing.T) {
	path := writeWholeFileFixture(t)

	tests := []struct {
		expr string
		want string
	}{
		{expr: "A > MEDIAN(A)", want: "MEDIAN"},
		{expr: "LAG(A, 1) > 0", want: "LAG"},
		{expr: "A > A.(# - 1)", want: "row reference"},
	}
	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			res, err := FilterWithCCL(context.Background(), path, tt.expr)
			if err == nil {
				t.Fatalf("FilterWithCCL(%q) kept %d rows instead of refusing it", tt.expr, len(filteredRows(t, res)))
			}
			if res != nil {
				t.Errorf("FilterWithCCL(%q) returned a table as well as the error", tt.expr)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error %q does not name %q", err, tt.want)
			}
		})
	}
}

// sameNamedColumns compares only the named columns of got with the same columns
// of want, cell by cell through sameCell, naming the first that differs. ApplyCCL
// writes a column back through a parquet file, which keeps a column's own type, so
// a script that replaces a column of another type cannot be compared on that
// column; naming the ones the script created keeps the comparison on what the
// statements agree about.
func sameNamedColumns(t *testing.T, got, want *insyra.DataTable, script string, names ...string) {
	t.Helper()
	if got.Err() != nil {
		t.Fatalf("the table ApplyCCL wrote carries an error: %v", got.Err())
	}
	if len(names) == 0 {
		t.Fatal("sameNamedColumns was given no column to compare")
	}
	for _, name := range names {
		gotCol := got.GetColByName(name)
		if gotCol == nil {
			t.Fatalf("ApplyCCL(%q) wrote no column %q", script, name)
		}
		wantCol := want.GetColByName(name)
		if wantCol == nil {
			t.Fatalf("ExecuteCCL(%q) left no column %q on the loaded table", script, name)
		}
		gotData := gotCol.Data()
		wantData := wantCol.Data()
		if len(gotData) != len(wantData) {
			t.Fatalf("ApplyCCL(%q) column %q holds %d values, want %d",
				script, name, len(gotData), len(wantData))
		}
		for i, w := range wantData {
			if !sameCell(gotData[i], w) {
				t.Fatalf("ApplyCCL(%q) column %q row %d = %v (%T), want %v (%T)",
					script, name, i, gotData[i], gotData[i], w, w)
			}
		}
	}
}

// appliedAndCompared runs script through ApplyCCL on a fixture of its own and
// compares the named columns with what ExecuteCCL leaves on the same file read
// with Read. ApplyCCL writes back to the path it was given and the expectation is
// the file before it was touched, so each script needs a file of its own.
func appliedAndCompared(t *testing.T, script string, columns ...string) {
	t.Helper()
	want := expectedApply(t, writeWholeFileFixture(t), script)

	path := writeWholeFileFixture(t)
	if err := ApplyCCL(context.Background(), path, script); err != nil {
		t.Fatalf("ApplyCCL(%q): %v", script, err)
	}
	got, err := Read(context.Background(), path, ReadOptions{})
	if err != nil {
		t.Fatalf("Read after ApplyCCL(%q): %v", script, err)
	}
	sameNamedColumns(t, got, want, script, columns...)
}

// laterStatementScripts are the scripts whose second statement reads what the
// first one wrote: an assignment replaced a column, so a fixed row of it and an
// aggregate over it are taken from the written values, and a NEW column is read
// by an aggregate over the whole file.
var laterStatementScripts = []struct {
	script string
	column string
}{
	// Only u and t, which the second statement computes from the values the
	// first one wrote, are compared; TestApplyCCLWritesWhatWriteWouldWrite
	// covers how B itself is written.
	{script: "['B'] = B / 2; NEW('u') = B.1 - B", column: "u"},
	{script: "['B'] = B / 2; NEW('t') = B - AVG(B)", column: "t"},
	{script: "NEW('c') = IF(MOD(#, 1000) == 0, TRUE, A); NEW('d') = SUM(['c'])", column: "d"},
}

// TestApplyCCLLaterStatementsReadWhatEarlierOnesWrote holds a statement after an
// assignment or a NEW to the values the statements before it wrote. Resolving a
// statement over a batch rebuilt from the file would read the file's own values
// where the rows computed afterwards read what was written, so the two halves of
// one script can disagree; both have to read the same values.
func TestApplyCCLLaterStatementsReadWhatEarlierOnesWrote(t *testing.T) {
	for _, tt := range laterStatementScripts {
		t.Run(tt.script, func(t *testing.T) {
			appliedAndCompared(t, tt.script, tt.column)
		})
	}
}

// TestCCLSpreadsARowLengthResult holds a row-independent expression whose value is
// a slice as long as the table to the reading it gets on a loaded table, where
// such a value is spread over the rows: a filter keeps the rows its own element
// keeps, a NEW column takes them, and an assignment gives each row its own.
// Putting the whole slice in every row keeps a filter's rows and writes the slice
// itself into the file.
func TestCCLSpreadsARowLengthResult(t *testing.T) {
	t.Run("FilterWithCCL keeps what the row's own element keeps", func(t *testing.T) {
		const expr = "A.0:2499"
		path := writeWholeFileFixture(t)
		want := expectedFilter(t, path, expr)
		res, err := FilterWithCCL(context.Background(), path, expr)
		if err != nil {
			t.Fatalf("FilterWithCCL(%q): %v", expr, err)
		}
		sameRows(t, filteredRows(t, res), want)
	})

	// B, not A: A holds a missing value, and a missing value in a column the
	// script creates does not come back out of the file (buildArrowRecord gives
	// such a column a non-nullable field, so its null is written as a zero), so
	// that column cannot be compared without measuring the writer's own defect
	// rather than this rule. B holds no missing value, so the same question — a
	// row-length slice spread over the rows — is answerable.
	t.Run("a NEW column takes its own element", func(t *testing.T) {
		appliedAndCompared(t, "NEW('c') = B.0:2499", "c")
	})

	t.Run("an assignment gives each row its own element", func(t *testing.T) {
		appliedAndCompared(t, "['B'] = B.0:2499", "B")
	})
}

// TestApplyCCLIgnoresABareExpressionStatement holds ApplyCCL to ExecuteCCL about
// a statement that is neither a NEW nor an assignment: the loaded table runs it
// and does nothing with it, so the streaming path has to write the same table
// rather than refuse the script for a part it never evaluates.
func TestApplyCCLIgnoresABareExpressionStatement(t *testing.T) {
	const script = "CUMSUM(A); NEW('c') = A * 2"
	appliedAndCompared(t, script, "c")
}

// writeOneRowFixture writes a file whose only column A holds one value, so an
// aggregate over it has nothing to spread, and returns its path.
func writeOneRowFixture(t *testing.T) string {
	t.Helper()
	dt := insyra.NewDataTable(insyra.NewDataList(1.0).SetName("A"))
	path := filepath.Join(t.TempDir(), "one-row.parquet")
	if err := Write(dt, path); err != nil {
		t.Fatalf("writing the one-row fixture: %v", err)
	}
	return path
}

// TestCCLDefersAPartThatFails holds a part that cannot be computed over the file
// to failing where it is evaluated, the way the loaded table fails only where it
// reads the rows it has not got: a branch the condition does not take is never
// evaluated, so the script runs, while a row the part reads fails the rows that
// reach it.
func TestCCLDefersAPartThatFails(t *testing.T) {
	t.Run("a branch the condition does not take", func(t *testing.T) {
		const expr = "IF(COUNT(A) > 5000, A > A.5000, TRUE)"
		path := writeWholeFileFixture(t)
		want := expectedFilter(t, path, expr)
		res, err := FilterWithCCL(context.Background(), path, expr)
		if err != nil {
			t.Fatalf("FilterWithCCL(%q): %v", expr, err)
		}
		sameRows(t, filteredRows(t, res), want)
	})

	t.Run("a row past the end of the file", func(t *testing.T) {
		const expr = "A > A.5000"
		path := writeWholeFileFixture(t)
		res, err := FilterWithCCL(context.Background(), path, expr)
		if err == nil {
			t.Fatalf("FilterWithCCL(%q) kept %d rows instead of failing on a row past the end",
				expr, len(filteredRows(t, res)))
		}
		if res != nil {
			t.Errorf("FilterWithCCL(%q) returned a table as well as the error", expr)
		}
	})

	t.Run("an aggregate over one value the condition skips", func(t *testing.T) {
		const script = "NEW('c') = IF(COUNT(A) >= 2, STDEV(A), 0)"
		want := expectedApply(t, writeOneRowFixture(t), script)

		path := writeOneRowFixture(t)
		if err := ApplyCCL(context.Background(), path, script); err != nil {
			t.Fatalf("ApplyCCL(%q): %v", script, err)
		}
		got, err := Read(context.Background(), path, ReadOptions{})
		if err != nil {
			t.Fatalf("Read after ApplyCCL(%q): %v", script, err)
		}
		sameNamedColumns(t, got, want, script, "c")
	})
}

// TestCCLFinishesARangeOfAggregates holds a row range whose two ends are whole
// file aggregates, and one that ends before it starts, to the reading the loaded
// table gives them. Both names a range only the file's rows can settle, so each
// part is resolved first and the range is evaluated from what they left.
func TestCCLFinishesARangeOfAggregates(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for _, expr := range []string{
		"A > SUM(A.(MIN(B):MAX(B)))",
		"A > SUM(A.5:2)",
	} {
		t.Run(expr, func(t *testing.T) {
			path := writeWholeFileFixture(t)
			want := expectedFilter(t, path, expr)
			res, err := FilterWithCCL(ctx, path, expr)
			if err != nil {
				t.Fatalf("FilterWithCCL(%q): %v", expr, err)
			}
			sameRows(t, filteredRows(t, res), want)
		})
	}
}
