package parquet

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

// A sequence function reads rows the batch after the current one has not
// arrived with, so FilterWithCCL and ApplyCCL used to refuse every one of them.
// The tests below hold the streamed answer to the one the same expression gives
// on the file loaded with Read, and the placements a stream cannot reach to the
// answer, or the error, the loaded table gives them.

// applySequenceScripts are the scripts whose whole right-hand side is a sequence
// function, which ApplyCCL has to answer as the loaded table answers them: a
// cumulative one, a shift each way, a shift wider than a batch, a window wider
// than a batch, a difference, a percentage change, an expression inside a
// cumulative one, a second statement reading what a sequence wrote, a shift
// reaching past the last row, an assignment replacing a column of the file, and
// a second statement reading a shift the first one wrote.
var applySequenceScripts = []struct {
	script  string
	columns []string
}{
	{script: "NEW('c') = CUMSUM(A)", columns: []string{"c"}},
	{script: "NEW('n') = LEAD(A, 3); NEW('m') = ['n'] - A", columns: []string{"n", "m"}},
	{script: "NEW('l') = LAG(A, 1001)", columns: []string{"l"}},
	{script: "NEW('r') = ROLLING_MEAN(A, 1500)", columns: []string{"r"}},
	{script: "NEW('d') = DIFF(B)", columns: []string{"d"}},
	{script: "NEW('p') = PCT_CHANGE(A, 2)", columns: []string{"p"}},
	{script: "NEW('x') = CUMMAX(A - AVG(A))", columns: []string{"x"}},
	{script: "NEW('s') = ROLLING_STD(A, 3); NEW('t') = LEAD(['s'], 1)", columns: []string{"s", "t"}},
	{script: "NEW('y') = LEAD(A, 2500)", columns: []string{"y"}},
	{script: "['A'] = LAG(A, 1)", columns: []string{"A"}},
	{script: "NEW('k') = LEAD(A, 1200); NEW('j') = CUMSUM(['k'])", columns: []string{"k", "j"}},
}

func TestApplyCCLStreamsSequences(t *testing.T) {
	for _, tt := range applySequenceScripts {
		t.Run(tt.script, func(t *testing.T) {
			appliedAndCompared(t, tt.script, tt.columns...)
		})
	}
}

// filterSequenceExpressions are the filter expressions that are one sequence
// function, which FilterWithCCL has to answer as the loaded table answers them.
// The shifts and windows reach past a single batch, so a batch's own answer
// would keep the wrong rows.
var filterSequenceExpressions = []string{
	"DIFF(A)",
	"LAG(A, 1001)",
	"LEAD(A, 3)",
	"CUMSUM(B)",
	"ROLLING_MAX(A, 1200)",
}

func TestFilterWithCCLStreamsSequences(t *testing.T) {
	for _, expr := range filterSequenceExpressions {
		t.Run(expr, func(t *testing.T) {
			path := writeWholeFileFixture(t)
			want := expectedFilter(t, path, expr)

			res, err := FilterWithCCL(context.Background(), path, expr)
			if err != nil {
				t.Fatalf("FilterWithCCL(%q): %v", expr, err)
			}
			sameRows(t, filteredRows(t, res), want)
		})
	}
}

// TestCCLGivesTheTablesAnswerForASequenceInsideAnExpression holds the placement a
// stream cannot reach, one sequence function feeding another expression, to the
// loaded table: it is computed over the whole of the column it reads, so the
// answer is the table's, and so is the error where the table fails. A failing
// script leaves the file as it was.
func TestCCLGivesTheTablesAnswerForASequenceInsideAnExpression(t *testing.T) {
	t.Run("FilterWithCCL fails on a sequence inside a comparison as the table does", func(t *testing.T) {
		const expr = "CUMSUM(A) > 10"
		path := writeWholeFileFixture(t)

		dt, err := Read(context.Background(), path, ReadOptions{})
		if err != nil {
			t.Fatalf("Read(%s): %v", path, err)
		}
		dt.AddColUsingCCL("__keep", expr)
		if dt.Err() == nil || !strings.Contains(dt.Err().Error(), "invalid operands") {
			t.Fatalf("AddColUsingCCL(%q) on the loaded table: error %v, want one holding %q", expr, dt.Err(), "invalid operands")
		}

		res, err := FilterWithCCL(context.Background(), path, expr)
		if err == nil {
			t.Fatalf("FilterWithCCL(%q) kept %d rows where the loaded table fails", expr, len(filteredRows(t, res)))
		}
		if res != nil {
			t.Errorf("FilterWithCCL(%q) returned a table as well as the error", expr)
		}
		if !strings.Contains(err.Error(), "invalid operands") {
			t.Errorf("error %.200q does not hold %q, which the table's does", err, "invalid operands")
		}
	})

	t.Run("ApplyCCL fails on a sequence inside an expression as the table does", func(t *testing.T) {
		const script = "NEW('c') = CUMSUM(A) + 1"
		path := writeWholeFileFixture(t)
		before, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading the fixture: %v", err)
		}

		dt, err := Read(context.Background(), path, ReadOptions{})
		if err != nil {
			t.Fatalf("Read(%s): %v", path, err)
		}
		dt.ExecuteCCL(script)
		if dt.Err() == nil || !strings.Contains(dt.Err().Error(), "invalid operands") {
			t.Fatalf("ExecuteCCL(%q) on the loaded table: error %v, want one holding %q", script, dt.Err(), "invalid operands")
		}

		err = ApplyCCL(context.Background(), path, script)
		if err == nil {
			t.Fatalf("ApplyCCL(%q) wrote the file where the loaded table fails", script)
		}
		if !strings.Contains(err.Error(), "invalid operands") {
			t.Errorf("error %.200q does not hold %q, which the table's does", err, "invalid operands")
		}

		after, readErr := os.ReadFile(path)
		if readErr != nil {
			t.Fatalf("reading the fixture back: %v", readErr)
		}
		if !bytes.Equal(before, after) {
			t.Errorf("the file changed (%d bytes before, %d after) although the script failed",
				len(before), len(after))
		}
	})

	t.Run("ApplyCCL computes an aggregate of a sequence as the table does", func(t *testing.T) {
		appliedAndCompared(t, "NEW('c') = SUM(CUMSUM(A))", "c")
	})
}

// TestApplyCCLWritesANewColumnsMissingValues holds a column a script creates
// against the missing values it copies: a field that is not nullable drops them,
// and the writer's zero takes their place, so a column of A comes back holding
// 0 where A was missing.
func TestApplyCCLWritesANewColumnsMissingValues(t *testing.T) {
	dt := insyra.NewDataTable(insyra.NewDataList(1.0, nil, 3.0).SetName("A"))
	path := filepath.Join(t.TempDir(), "missing.parquet")
	if err := Write(dt, path); err != nil {
		t.Fatalf("writing the missing-value fixture: %v", err)
	}

	const script = "NEW('c') = A"
	if err := ApplyCCL(context.Background(), path, script); err != nil {
		t.Fatalf("ApplyCCL(%q): %v", script, err)
	}

	got, err := Read(context.Background(), path, ReadOptions{})
	if err != nil {
		t.Fatalf("Read after ApplyCCL(%q): %v", script, err)
	}
	if got.Err() != nil {
		t.Fatalf("the table ApplyCCL wrote carries an error: %v", got.Err())
	}

	want := []any{1.0, nil, 3.0}
	data := got.GetColByName("c").Data()
	if len(data) != len(want) {
		t.Fatalf("ApplyCCL(%q) wrote column c with %d values, want %d", script, len(data), len(want))
	}
	for i, w := range want {
		if !sameCell(data[i], w) {
			t.Errorf("ApplyCCL(%q) column c row %d = %v (%T), want %v (%T)",
				script, i, data[i], data[i], w, w)
		}
	}
}

// TestApplyCCLSettlesANewColumnsTypeInTheFirstRowGroup holds the type of a column
// a script creates against a first row group that holds nothing but missing
// values: ROLLING_MEAN answers nothing for its first 1499 rows, so a type
// settled from the batch alone would be text and every later number would be
// written as a string.
func TestApplyCCLSettlesANewColumnsTypeInTheFirstRowGroup(t *testing.T) {
	const script = "NEW('r') = ROLLING_MEAN(A, 1500)"

	path := writeWholeFileFixture(t)
	if err := ApplyCCL(context.Background(), path, script); err != nil {
		t.Fatalf("ApplyCCL(%q): %v", script, err)
	}

	got, err := Read(context.Background(), path, ReadOptions{})
	if err != nil {
		t.Fatalf("Read after ApplyCCL(%q): %v", script, err)
	}
	if got.Err() != nil {
		t.Fatalf("the table ApplyCCL wrote carries an error: %v", got.Err())
	}

	data := got.GetColByName("r").Data()
	// The window is 1500 rows wide, so no row before 1499 has a value at all;
	// the first row group ends at 1000, so nothing in it is a number either.
	for i := 1499; i < len(data); i++ {
		if data[i] == nil {
			continue
		}
		if _, ok := data[i].(float64); !ok {
			t.Fatalf("ApplyCCL(%q) column r row %d = %v (%T), want a float64",
				script, i, data[i], data[i])
		}
	}
	values := 0
	for _, v := range data {
		if v != nil {
			values++
		}
	}
	if values == 0 {
		t.Fatalf("ApplyCCL(%q) wrote no value into column r at all", script)
	}
}
