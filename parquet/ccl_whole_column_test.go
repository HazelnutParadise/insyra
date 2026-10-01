package parquet

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/HazelnutParadise/insyra"
	engineccl "github.com/HazelnutParadise/insyra/engine/ccl"
	"github.com/HazelnutParadise/insyra/internal/ccl"
)

// FilterWithCCL and ApplyCCL compute most of what reads beyond the current row
// without holding whole columns. What cannot be computed that way — MEDIAN, an
// aggregate a caller registered, a row computed from the current one, a column
// range inside an aggregate's argument, a sequence function that is not the
// whole expression — is computed by reading the whole of the columns it reads
// and evaluating it the way the DataTable CCL methods do, so the answer, or the
// error, is the loaded table's. The tests below hold both to the loaded table.

// wholeColumnScripts are the ApplyCCL scripts that need whole columns, with the
// columns they create, which have to equal what ExecuteCCL gives on the loaded
// table: a median, a row computed from the current one, a column range inside an
// aggregate, a sequence inside a sequence, an aggregate of a sequence, a later
// statement reading the whole of a column an earlier one made, and a sequence
// reading a column a whole-column statement made.
var wholeColumnScripts = []struct {
	script  string
	columns []string
}{
	{script: "NEW('c') = A - MEDIAN(A)", columns: []string{"c"}},
	{script: "NEW('p') = IF(# > 0, A.(# - 1), 0)", columns: []string{"p"}},
	{script: "NEW('k') = COUNT(IF(A > 0, A:B, 0))", columns: []string{"k"}},
	{script: "NEW('l') = LAG(CUMSUM(A), 1)", columns: []string{"l"}},
	{script: "NEW('s') = SUM(CUMSUM(B))", columns: []string{"s"}},
	{script: "NEW('c') = A * 2; NEW('m') = MEDIAN(['c']); NEW('d') = ['m'] + 1", columns: []string{"c", "m", "d"}},
	{script: "NEW('q') = MEDIAN(A) + #; NEW('n') = LEAD(['q'], 2)", columns: []string{"q", "n"}},
}

func TestApplyCCLComputesWholeColumnParts(t *testing.T) {
	for _, tt := range wholeColumnScripts {
		t.Run(tt.script, func(t *testing.T) {
			appliedAndCompared(t, tt.script, tt.columns...)
		})
	}

	// A median needs A whole and nothing else: B is in the file and is not read.
	t.Run("only the columns it reads", func(t *testing.T) {
		before := wholeColumnsLoaded.Load()
		appliedAndCompared(t, "NEW('c') = A - MEDIAN(A)", "c")
		if got := wholeColumnsLoaded.Load() - before; got != 1 {
			t.Errorf("the whole-column pass read %d columns, want 1 (A)", got)
		}
	})
}

// zzParquetFirst is an aggregate a caller registers: the first value that is not
// missing.
func zzParquetFirst(args ...[]any) (any, error) {
	for _, v := range args[0] {
		if v != nil {
			return v, nil
		}
	}
	return nil, nil
}

// TestApplyCCLComputesARegisteredAggregate holds an aggregate a caller registered
// to the loaded table's answer. Its arithmetic is the caller's, so no streaming
// form stands for it, and it is computed over the column it reads.
func TestApplyCCLComputesARegisteredAggregate(t *testing.T) {
	engineccl.RegisterAggregateFunction("ZZPARQUETFIRST", zzParquetFirst)
	appliedAndCompared(t, "NEW('f') = ZZPARQUETFIRST(A)", "f")
}

// wholeColumnFilters are the filter expressions that need whole columns, which
// FilterWithCCL has to answer as the loaded table answers them.
var wholeColumnFilters = []string{
	"A > MEDIAN(A)",
	"IF(# > 0, A > A.(# - 1), FALSE)",
	"COUNT(IF(A > 0, A:B, 0)) > 0 && A > 2000",
	"LAG(CUMSUM(A), 1)",
}

func TestFilterWithCCLComputesWholeColumnParts(t *testing.T) {
	path := writeWholeFileFixture(t)
	for _, expr := range wholeColumnFilters {
		t.Run(expr, func(t *testing.T) {
			want := expectedFilter(t, path, expr)
			if len(want) == 0 {
				t.Fatalf("the loaded table keeps no row for %q, so the comparison proves nothing", expr)
			}
			res, err := FilterWithCCL(context.Background(), path, expr)
			if err != nil {
				t.Fatalf("FilterWithCCL(%q): %v", expr, err)
			}
			sameRows(t, filteredRows(t, res), want)
		})
	}
}

// tableScriptError returns the error ExecuteCCL leaves on the file loaded with
// Read, which is the error ApplyCCL has to give for the same script.
func tableScriptError(t *testing.T, path, script string) error {
	t.Helper()
	dt, err := Read(context.Background(), path, ReadOptions{})
	if err != nil {
		t.Fatalf("Read(%s): %v", path, err)
	}
	dt.ExecuteCCL(script)
	return dt.Err()
}

// shorten cuts a message to its first characters: the table's own error prints
// the whole column it could not use.
func shorten(err error) string {
	s := err.Error()
	if len(s) > 200 {
		return s[:200] + "..."
	}
	return s
}

// assertFileUntouched fails unless the file at path is byte for byte what before
// holds and its directory holds nothing but that file.
func assertFileUntouched(t *testing.T, path string, before []byte) {
	t.Helper()
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading the file back: %v", err)
	}
	if !bytes.Equal(before, after) {
		t.Errorf("the file changed (%d bytes before, %d after) although the script failed",
			len(before), len(after))
	}
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatalf("reading the directory: %v", err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Errorf("a temporary file is left behind: %s", e.Name())
		}
	}
}

func TestApplyCCLGivesTheTablesErrorForAWholeColumnPart(t *testing.T) {
	tests := []struct {
		script string
		want   string
	}{
		// A sequence inside an expression: the table cannot add a number to a column.
		{script: "NEW('c') = CUMSUM(A) + 1", want: "invalid operands"},
		// A column that is not there, in an expression that needs whole columns.
		{script: "NEW('c') = MEDIAN(['zz'])", want: "zz"},
		// And in one that does not.
		{script: "NEW('c') = ['zz']", want: "zz"},
		// An assignment to a column that is not there, whose right-hand side needs
		// whole columns.
		{script: "['zz'] = MEDIAN(A)", want: "zz"},
	}
	for _, tt := range tests {
		t.Run(tt.script, func(t *testing.T) {
			tableErr := tableScriptError(t, writeWholeFileFixture(t), tt.script)
			if tableErr == nil {
				t.Fatalf("ExecuteCCL(%q) on the loaded table did not fail, so it is no comparison", tt.script)
			}
			if !strings.Contains(tableErr.Error(), tt.want) {
				t.Fatalf("the table's error %q does not hold %q", shorten(tableErr), tt.want)
			}

			path := writeWholeFileFixture(t)
			before, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("reading the fixture: %v", err)
			}
			err = ApplyCCL(context.Background(), path, tt.script)
			if err == nil {
				t.Fatalf("ApplyCCL(%q) wrote the file although the table fails on it", tt.script)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error %q does not hold %q, which the table's error does", shorten(err), tt.want)
			}
			assertFileUntouched(t, path, before)
		})
	}
}

// TestApplyCCLFailsOnALaterWholeColumnStatementBeforeWriting holds a script
// whose whole-column statement is the second one and fails: the first statement
// has been computed by then, and nothing of it reaches the file.
func TestApplyCCLFailsOnALaterWholeColumnStatementBeforeWriting(t *testing.T) {
	const script = "NEW('c') = A * 2; NEW('d') = MEDIAN(['c']) + CUMSUM(A) + 1"
	if tableScriptError(t, writeWholeFileFixture(t), script) == nil {
		t.Fatalf("ExecuteCCL(%q) on the loaded table did not fail", script)
	}

	path := writeWholeFileFixture(t)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading the fixture: %v", err)
	}
	if err := ApplyCCL(context.Background(), path, script); err == nil {
		t.Fatalf("ApplyCCL(%q) wrote the file although the table fails on it", script)
	}
	assertFileUntouched(t, path, before)
}

// TestWholeColumnsContextRefusesAColumnItDidNotRead holds the context a
// whole-column pass evaluates on to what its name says: a column that was not
// read is an error to read, never a column of missing values.
func TestWholeColumnsContextRefusesAColumnItDidNotRead(t *testing.T) {
	c := newWholeColumnsContext([]string{"a", "b"}, [][]any{{1.0, 2.0}, nil}, 2)

	if got := c.GetColCount(); got != 2 {
		t.Errorf("GetColCount() = %d, want 2", got)
	}
	if got := c.GetRowCount(); got != 2 {
		t.Errorf("GetRowCount() = %d, want 2", got)
	}

	// The column that was read is read as it is.
	if v, err := c.GetCell(0, 1); err != nil || v != 2.0 {
		t.Errorf("GetCell(0, 1) = %v, %v, want 2, nil", v, err)
	}
	if err := c.SetRowIndex(1); err != nil {
		t.Fatalf("SetRowIndex(1): %v", err)
	}
	if v, err := c.GetColByName("a"); err != nil || v != 2.0 {
		t.Errorf("GetColByName(\"a\") = %v, %v, want 2, nil", v, err)
	}

	const want = "column b was not read for this expression"
	reads := map[string]func() error{
		"GetCell":          func() error { _, err := c.GetCell(1, 0); return err },
		"GetCellByName":    func() error { _, err := c.GetCellByName("b", 0); return err },
		"GetColData":       func() error { _, err := c.GetColData(1); return err },
		"GetColDataByName": func() error { _, err := c.GetColDataByName("b"); return err },
		"GetColByName":     func() error { _, err := c.GetColByName("b"); return err },
		"GetRowAt":         func() error { _, err := c.GetRowAt(0); return err },
	}
	for name, read := range reads {
		if err := read(); err == nil || err.Error() != want {
			t.Errorf("%s of the column that was not read: error %v, want %q", name, err, want)
		}
	}
	if got := c.GetCol(1); got != nil {
		t.Errorf("GetCol(1) = %v, want nil", got)
	}
}

func TestLoadColumnsReadsOnlyTheColumnsWanted(t *testing.T) {
	path := writeWholeFileFixture(t)
	totalRows, colNames, _, err := cclFileInfo(path)
	if err != nil {
		t.Fatalf("cclFileInfo: %v", err)
	}

	t.Run("from the file", func(t *testing.T) {
		before := wholeColumnsLoaded.Load()
		names, cols, err := loadColumns(context.Background(), path, colNames, nil, totalRows, nil, []int{1})
		if err != nil {
			t.Fatalf("loadColumns: %v", err)
		}
		if !reflect.DeepEqual(names, []string{"A", "B"}) {
			t.Errorf("names = %v, want [A B]", names)
		}
		if len(cols) != 2 || cols[0] != nil {
			t.Fatalf("cols holds %d columns, the first %v, want 2 with the first not read", len(cols), cols[0])
		}
		if len(cols[1]) != wholeFileRows {
			t.Errorf("column B holds %d values, want %d", len(cols[1]), wholeFileRows)
		}
		if got := wholeColumnsLoaded.Load() - before; got != 1 {
			t.Errorf("loadColumns counted %d columns, want 1", got)
		}
	})

	t.Run("as the earlier statements leave the file", func(t *testing.T) {
		prior, err := ccl.CompileMultiline("NEW('c') = A * 2")
		if err != nil {
			t.Fatalf("compiling: %v", err)
		}
		names, cols, err := loadColumns(context.Background(), path, colNames, prior, totalRows, nil, []int{2})
		if err != nil {
			t.Fatalf("loadColumns: %v", err)
		}
		if !reflect.DeepEqual(names, []string{"A", "B", "c"}) {
			t.Errorf("names = %v, want [A B c]", names)
		}
		if cols[0] != nil || cols[1] != nil {
			t.Errorf("columns that were not wanted were read")
		}
		if len(cols[2]) != wholeFileRows || cols[2][0] != 2.0 || cols[2][wholeFileRows-1] != float64(wholeFileRows*2) {
			t.Errorf("column c = %v ... %v over %d rows, want 2 ... %d over %d",
				cols[2][0], cols[2][len(cols[2])-1], len(cols[2]), wholeFileRows*2, wholeFileRows)
		}
	})
}

// TestCCLWholeColumnPartsOnAFileWithNoRows holds a file with no rows to the
// loaded table: nothing is evaluated, a filter keeps no row and keeps the
// columns, and ApplyCCL leaves the file as it was.
func TestCCLWholeColumnPartsOnAFileWithNoRows(t *testing.T) {
	t.Run("ApplyCCL", func(t *testing.T) {
		path := writeNoRowsFixture(t)
		before, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading the fixture: %v", err)
		}
		if err := ApplyCCL(context.Background(), path, "NEW('c') = A - MEDIAN(A); ['A'] = MEDIAN(A)"); err != nil {
			t.Fatalf("ApplyCCL: %v", err)
		}
		after, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading the fixture back: %v", err)
		}
		if !bytes.Equal(before, after) {
			t.Errorf("ApplyCCL changed a file with no rows (%d bytes before, %d after)", len(before), len(after))
		}
	})

	t.Run("FilterWithCCL", func(t *testing.T) {
		for _, expr := range []string{"A > MEDIAN(A)", "MEDIAN(A) > 0"} {
			res, err := FilterWithCCL(context.Background(), writeNoRowsFixture(t), expr)
			if err != nil {
				t.Fatalf("FilterWithCCL(%q): %v", expr, err)
			}
			if got := res.ColNames(); !reflect.DeepEqual(got, []string{"A"}) {
				t.Errorf("FilterWithCCL(%q) returned the columns %v, want [A]", expr, got)
			}
			if rows, _ := res.Size(); rows != 0 {
				t.Errorf("FilterWithCCL(%q) kept %d rows of a file with none", expr, rows)
			}
		}
	})
}

// TestApplyCCLWholeColumnAssignment holds an assignment whose right-hand side
// needs whole columns to the loaded table: it replaces the column, and a later
// statement reads the replaced column.
func TestApplyCCLWholeColumnAssignment(t *testing.T) {
	appliedAndCompared(t, "['B'] = B - MEDIAN(B); NEW('u') = B + 1", "u")
}

// TestWholeColumnValuesAreTheTables holds the values a whole-column pass gives
// for each of its two kinds of input, an expression and a statement, to the
// loaded table's, without the file being written.
func TestWholeColumnValuesAreTheTables(t *testing.T) {
	path := writeWholeFileFixture(t)
	totalRows, colNames, _, err := cclFileInfo(path)
	if err != nil {
		t.Fatalf("cclFileInfo: %v", err)
	}

	t.Run("an expression", func(t *testing.T) {
		const expr = "A - MEDIAN(A)"
		node, err := ccl.CompileExpression(expr)
		if err != nil {
			t.Fatalf("compiling: %v", err)
		}
		got, err := wholeColumnValues(context.Background(), path, colNames, nil, totalRows, nil, node)
		if err != nil {
			t.Fatalf("wholeColumnValues: %v", err)
		}
		dt, err := Read(context.Background(), path, ReadOptions{})
		if err != nil {
			t.Fatalf("Read: %v", err)
		}
		dt.AddColUsingCCL("__v", expr)
		if dt.Err() != nil {
			t.Fatalf("AddColUsingCCL: %v", dt.Err())
		}
		want := dt.GetColByName("__v").Data()
		if len(got) != len(want) {
			t.Fatalf("got %d values, want %d", len(got), len(want))
		}
		for i := range want {
			if !sameCell(got[i], want[i]) {
				t.Fatalf("row %d = %v (%T), want %v (%T)", i, got[i], got[i], want[i], want[i])
			}
		}
	})

	t.Run("a statement", func(t *testing.T) {
		const script = "NEW('c') = A - MEDIAN(A)"
		nodes, err := ccl.CompileMultiline(script)
		if err != nil || len(nodes) != 1 {
			t.Fatalf("compiling: %v (%d statements)", err, len(nodes))
		}
		got, err := wholeColumnValues(context.Background(), path, colNames, nil, totalRows, nil, nodes[0])
		if err != nil {
			t.Fatalf("wholeColumnValues: %v", err)
		}
		want := expectedApply(t, path, script).GetColByName("c").Data()
		if len(got) != len(want) {
			t.Fatalf("got %d values, want %d", len(got), len(want))
		}
		for i := range want {
			if !sameCell(got[i], want[i]) {
				t.Fatalf("row %d = %v (%T), want %v (%T)", i, got[i], got[i], want[i], want[i])
			}
		}
	})

	t.Run("a column past the last one", func(t *testing.T) {
		node, err := ccl.CompileExpression("MEDIAN(Z) > 0")
		if err != nil {
			t.Fatalf("compiling: %v", err)
		}
		_, err = wholeColumnValues(context.Background(), path, colNames, nil, totalRows, nil, node)
		if err == nil {
			t.Fatal("wholeColumnValues read a column past the end of the file")
		}
		dt := insyra.NewDataTable(insyra.NewDataList(1.0).SetName("A"), insyra.NewDataList(2.0).SetName("B"))
		dt.AddColUsingCCL("__v", "MEDIAN(Z) > 0")
		if dt.Err() == nil {
			t.Fatal("the loaded table accepted a column past its end")
		}
		if !strings.Contains(err.Error(), "Z") {
			t.Errorf("error %q does not name the column Z", shorten(err))
		}
	})
}

// writeCountingFixture writes a file of rows rows with one column for each of
// names, in order, and returns its path. The first counts up from 1, the second
// repeats a short cycle of whole numbers and the third counts in halves, so every
// row differs from its neighbours in every column.
func writeCountingFixture(t *testing.T, rows int, names ...string) string {
	t.Helper()
	cols := make([]*insyra.DataList, len(names))
	for c, name := range names {
		data := make([]any, rows)
		for i := range data {
			switch c {
			case 0:
				data[i] = float64(i + 1)
			case 1:
				data[i] = int64((i * 7) % 11)
			default:
				data[i] = float64(i) / 2
			}
		}
		cols[c] = insyra.NewDataList(data...).SetName(name)
	}
	path := filepath.Join(t.TempDir(), "counting.parquet")
	if err := Write(insyra.NewDataTable(cols...), path); err != nil {
		t.Fatalf("writing the counting fixture: %v", err)
	}
	return path
}

// TestApplyCCLComputesARowInvariantNewColumnOnce holds a NEW whose expression does
// not change from row to row to the table's way of computing it, once: a
// sequence function over a running total has the same answer on every row of
// the loaded table, and evaluating it for each row of the file cost a pass over
// the whole column per row, 6 seconds for 20,000 rows where the table needs a
// millisecond.
func TestApplyCCLComputesARowInvariantNewColumnOnce(t *testing.T) {
	const (
		rows   = 30000
		script = "NEW('l') = LAG(CUMSUM(A), 1)"
		budget = 3 * time.Second
	)
	want := expectedApply(t, writeCountingFixture(t, rows, "A"), script)

	path := writeCountingFixture(t, rows, "A")
	start := time.Now()
	err := ApplyCCL(context.Background(), path, script)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("ApplyCCL(%q): %v", script, err)
	}
	if elapsed > budget {
		t.Errorf("ApplyCCL(%q) took %v on %d rows, want under %v: the expression does not change from row to row, so it is computed once",
			script, elapsed, rows, budget)
	}

	got, err := Read(context.Background(), path, ReadOptions{})
	if err != nil {
		t.Fatalf("Read after ApplyCCL(%q): %v", script, err)
	}
	sameNamedColumns(t, got, want, script, "l")
}

// TestApplyCCLStopsAWholeColumnStatementWhenCancelled holds the whole-column pass
// to its context: a statement that works for a long time over the rows of the
// whole file stops when the context ends instead of running to its end, and the
// file is left as it was.
//
// The statement needs whole columns because of the MEDIAN, and each row then
// builds a string of 200 kB, which costs the same on row 1 and on row 39,999. The
// cost of a row stays small next to the deadline (about 65 microseconds, and about
// five times that under the race detector), so how soon the pass notices the
// deadline is a matter of how often it looks, not of how slow the machine is.
// A pass that never looks takes about ten seconds, and one that looks every
// 1,024 rows returns within a few hundred milliseconds of the deadline even
// under the race detector, so the budget leaves room for a slow machine.
func TestApplyCCLStopsAWholeColumnStatementWhenCancelled(t *testing.T) {
	const (
		rows     = 150000
		script   = "NEW('c') = IF(MEDIAN(A) > 0, LEN(REPEAT('ab', 100000 + MOD(A, 2))), 0)"
		deadline = 200 * time.Millisecond
		budget   = 3 * time.Second
	)

	path := writeCountingFixture(t, rows, "A")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading the fixture: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), deadline)
	defer cancel()
	start := time.Now()
	err = ApplyCCL(ctx, path, script)
	elapsed := time.Since(start)

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("ApplyCCL(%q) under a %v deadline returned %v after %v, want context.DeadlineExceeded",
			script, deadline, err, elapsed)
	}
	if elapsed > budget {
		t.Errorf("ApplyCCL(%q) returned after %v, want under %v: the deadline was %v", script, elapsed, budget, deadline)
	}
	assertFileUntouched(t, path, before)
}

// TestFilterWithCCLStopsAWholeColumnExpressionWhenCancelled holds the other half
// of the whole-column pass to its context: an expression evaluated row by row over
// the whole file, which is not a statement, stops when the context ends as well.
func TestFilterWithCCLStopsAWholeColumnExpressionWhenCancelled(t *testing.T) {
	const (
		rows     = 150000
		expr     = "IF(MEDIAN(A) > 0, LEN(REPEAT('ab', 100000 + MOD(A, 2))) > 0, FALSE)"
		deadline = 200 * time.Millisecond
		budget   = 3 * time.Second
	)

	path := writeCountingFixture(t, rows, "A")
	ctx, cancel := context.WithTimeout(context.Background(), deadline)
	defer cancel()
	start := time.Now()
	res, err := FilterWithCCL(ctx, path, expr)
	elapsed := time.Since(start)

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("FilterWithCCL(%q) under a %v deadline returned %v after %v, want context.DeadlineExceeded",
			expr, deadline, err, elapsed)
	}
	if res != nil {
		t.Errorf("FilterWithCCL(%q) returned a table as well as the error", expr)
	}
	if elapsed > budget {
		t.Errorf("FilterWithCCL(%q) returned after %v, want under %v: the deadline was %v", expr, elapsed, budget, deadline)
	}
}

// TestApplyCCLWritesEachRowOfAt holds @ to the loaded table, where each row of a
// column that takes @ holds its own row: a statement whose other parts need whole
// columns evaluates every row through the one context, and the row the context
// handed out for @ was the context's own buffer, so every row ended up holding
// the last one.
func TestApplyCCLWritesEachRowOfAt(t *testing.T) {
	const (
		rows   = 1500
		script = "NEW('r') = IF(MEDIAN(A) > 0, @, 0); NEW('s') = @"
	)
	want := expectedApply(t, writeCountingFixture(t, rows, "A", "B", "C"), script)

	path := writeCountingFixture(t, rows, "A", "B", "C")
	if err := ApplyCCL(context.Background(), path, script); err != nil {
		t.Fatalf("ApplyCCL(%q): %v", script, err)
	}
	got, err := Read(context.Background(), path, ReadOptions{})
	if err != nil {
		t.Fatalf("Read after ApplyCCL(%q): %v", script, err)
	}

	// A row is a list of cells on the loaded table, and a parquet column holds a
	// row as its text, so the two are compared by the text a list of cells prints.
	for _, name := range []string{"r", "s"} {
		wantCol := want.GetColByName(name)
		gotCol := got.GetColByName(name)
		if wantCol == nil || gotCol == nil {
			t.Fatalf("column %q is missing: the loaded table has %v, the file has %v", name, wantCol != nil, gotCol != nil)
		}
		wantData, gotData := wantCol.Data(), gotCol.Data()
		if len(gotData) != rows || len(wantData) != rows {
			t.Fatalf("column %q holds %d values and the loaded table %d, want %d", name, len(gotData), len(wantData), rows)
		}
		for _, row := range []int{0, 1, rows - 1} {
			if gotText, ok := gotData[row].(string); !ok || gotText != fmt.Sprint(wantData[row]) {
				t.Errorf("column %q row %d = %v (%T), want the row %v the loaded table holds",
					name, row, gotData[row], gotData[row], wantData[row])
			}
		}
	}
}

// TestApplyCCLReportsAMissingTargetFirst holds an assignment to a column that is
// not there to the table's order of checks: the target is resolved before the
// right-hand side is bound, so the error is the missing target and not a column
// the right-hand side reads, and the file is not read for a statement that
// cannot be written.
func TestApplyCCLReportsAMissingTargetFirst(t *testing.T) {
	const (
		script = "['zz'] = MEDIAN(['yy'])"
		reason = "assignment target column 'zz' does not exist"
	)
	tableErr := tableScriptError(t, writeWholeFileFixture(t), script)
	if tableErr == nil || !strings.Contains(tableErr.Error(), reason) {
		t.Fatalf("the table's error for %q is %v, want one holding %q", script, tableErr, reason)
	}

	path := writeWholeFileFixture(t)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading the fixture: %v", err)
	}
	loaded := wholeColumnsLoaded.Load()

	err = ApplyCCL(context.Background(), path, script)
	if err == nil {
		t.Fatalf("ApplyCCL(%q) wrote the file although its target is not there", script)
	}
	if !strings.Contains(err.Error(), reason) {
		t.Errorf("error %q does not hold %q, which the table's error does", shorten(err), reason)
	}
	if strings.Contains(err.Error(), "yy") {
		t.Errorf("error %q names the column the right-hand side reads, which the table never gets to", shorten(err))
	}
	if got := wholeColumnsLoaded.Load() - loaded; got != 0 {
		t.Errorf("the whole-column pass read %d columns for a statement whose target is not there, want 0", got)
	}
	assertFileUntouched(t, path, before)
}
