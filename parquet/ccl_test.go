package parquet

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/apache/arrow/go/v17/arrow"
	"github.com/apache/arrow/go/v17/arrow/array"
	"github.com/apache/arrow/go/v17/arrow/memory"
)

// The whole CCL bridge — FilterWithCCL, ApplyCCL and the 17 parquetContext
// methods behind them — had never been run by a test.

// Column names here are deliberately longer than an Excel column reference and
// are addressed as ['name'], because resolveAssignTarget reads a bare target as
// a column *letter* before it tries it as a column *name*.

// fixture writes a four-column file and returns its path.
//
//	num   int64    1  2  3  4
//	score float64  10 20 30 40
//	label string   a  b  c  d
//	flag  bool     T  F  T  F
func fixture(t *testing.T) string {
	t.Helper()
	dt := insyra.NewDataTable(
		insyra.NewDataList(1, 2, 3, 4).SetName("num"),
		insyra.NewDataList(10.0, 20.0, 30.0, 40.0).SetName("score"),
		insyra.NewDataList("a", "b", "c", "d").SetName("label"),
		insyra.NewDataList(true, false, true, false).SetName("flag"),
	)
	path := filepath.Join(t.TempDir(), "fixture.parquet")
	if err := Write(dt, path); err != nil {
		t.Fatalf("writing the fixture: %v", err)
	}
	return path
}

// colValues reads one column of a result table back as a plain slice.
func colValues(t *testing.T, dt *insyra.DataTable, name string) []any {
	t.Helper()
	col := dt.GetColByName(name)
	if col == nil {
		t.Fatalf("column %q is missing from the result", name)
	}
	return col.Data()
}

func TestFilterWithCCL_KeepsMatchingRows(t *testing.T) {
	path := fixture(t)

	res, err := FilterWithCCL(context.Background(), path, "['num'] > 2")
	if err != nil {
		t.Fatalf("FilterWithCCL: %v", err)
	}

	rows, cols := res.Size()
	if rows != 2 || cols != 4 {
		t.Fatalf("result size: got %d rows and %d columns, want 2 and 4", rows, cols)
	}
	if got, want := colValues(t, res, "num"), []any{int64(3), int64(4)}; !reflect.DeepEqual(got, want) {
		t.Errorf("num: got %v, want %v", got, want)
	}
	if got, want := colValues(t, res, "score"), []any{30.0, 40.0}; !reflect.DeepEqual(got, want) {
		t.Errorf("score: got %v, want %v", got, want)
	}
	if got, want := colValues(t, res, "label"), []any{"c", "d"}; !reflect.DeepEqual(got, want) {
		t.Errorf("label: got %v, want %v", got, want)
	}
	if got, want := colValues(t, res, "flag"), []any{true, false}; !reflect.DeepEqual(got, want) {
		t.Errorf("flag: got %v, want %v", got, want)
	}
}

// A filter over a string column and one over a boolean column go through
// different branches of the "does this row pass" switch.
func TestFilterWithCCL_StringAndBooleanConditions(t *testing.T) {
	path := fixture(t)

	res, err := FilterWithCCL(context.Background(), path, "['label'] == 'b'")
	if err != nil {
		t.Fatalf("FilterWithCCL on a string column: %v", err)
	}
	if got, want := colValues(t, res, "num"), []any{int64(2)}; !reflect.DeepEqual(got, want) {
		t.Errorf("string filter: got %v, want %v", got, want)
	}

	res, err = FilterWithCCL(context.Background(), path, "['flag']")
	if err != nil {
		t.Fatalf("FilterWithCCL on a bool column: %v", err)
	}
	if got, want := colValues(t, res, "num"), []any{int64(1), int64(3)}; !reflect.DeepEqual(got, want) {
		t.Errorf("bool filter: got %v, want %v", got, want)
	}
}

// No match is not an error, and the result keeps its columns so a caller can
// still read the schema off it.
func TestFilterWithCCL_NoMatch(t *testing.T) {
	path := fixture(t)

	res, err := FilterWithCCL(context.Background(), path, "['num'] > 100")
	if err != nil {
		t.Fatalf("FilterWithCCL: %v", err)
	}

	rows, cols := res.Size()
	if rows != 0 {
		t.Errorf("rows: got %d, want 0", rows)
	}
	if cols != 4 {
		t.Errorf("columns: got %d, want 4 — the schema should survive an empty result", cols)
	}
	if got := res.ColNames(); !reflect.DeepEqual(got, []string{"num", "score", "label", "flag"}) {
		t.Errorf("column names: got %v", got)
	}
}

func TestFilterWithCCL_InvalidExpression(t *testing.T) {
	path := fixture(t)

	tests := []struct {
		name string
		expr string
		want string
	}{
		{name: "unbalanced", expr: "['num'] > ", want: "failed to compile CCL expression"},
		{name: "assignment", expr: "['num'] = 1", want: "does not support assignment syntax"},
		{name: "NEW", expr: "NEW('x') = 1", want: "does not support NEW function"},
		{name: "unknown column", expr: "['nope'] > 1", want: "column name 'nope' not found"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := FilterWithCCL(context.Background(), path, tt.expr)
			if err == nil {
				t.Fatalf("expected an error, got a result of %v", res)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error %q does not mention %q", err, tt.want)
			}
		})
	}
}

// A bad expression is rejected before the file is opened, so the path is never
// even looked at.
func TestFilterWithCCL_CompileErrorBeatsAMissingFile(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "not-there.parquet")

	_, err := FilterWithCCL(context.Background(), missing, "((")
	if err == nil {
		t.Fatal("expected a compile error")
	}
	if !strings.Contains(err.Error(), "failed to compile CCL expression") {
		t.Errorf("error %q is not the compile error", err)
	}
}

func TestFilterWithCCL_MissingFile(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "not-there.parquet")

	_, err := FilterWithCCL(context.Background(), missing, "['num'] > 1")
	if err == nil {
		t.Fatal("a missing file gave no error")
	}
	if !strings.Contains(err.Error(), "no such file") && !strings.Contains(err.Error(), "cannot find") {
		t.Errorf("error %q does not look like a missing-file error", err)
	}
}

// Cancelling the context stops the filter rather than returning a partial table.
func TestFilterWithCCL_CancelledContext(t *testing.T) {
	path := fixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := FilterWithCCL(ctx, path, "['num'] > 0")
	if err == nil {
		t.Fatal("a cancelled context gave no error")
	}
}

func TestApplyCCL_AddsAColumn(t *testing.T) {
	path := fixture(t)

	if err := ApplyCCL(context.Background(), path, "NEW('double') = ['score'] * 2"); err != nil {
		t.Fatalf("ApplyCCL: %v", err)
	}

	dt, err := Read(context.Background(), path, ReadOptions{})
	if err != nil {
		t.Fatalf("reading back: %v", err)
	}
	if got, want := dt.ColNames(), []string{"num", "score", "label", "flag", "double"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("columns after ApplyCCL: got %v, want %v", got, want)
	}
	if got, want := colValues(t, dt, "double"), []any{20.0, 40.0, 60.0, 80.0}; !reflect.DeepEqual(got, want) {
		t.Errorf("double: got %v, want %v", got, want)
	}
	// The original columns are unchanged.
	if got, want := colValues(t, dt, "num"), []any{int64(1), int64(2), int64(3), int64(4)}; !reflect.DeepEqual(got, want) {
		t.Errorf("num: got %v, want %v", got, want)
	}
}

func TestApplyCCL_OverwritesAnExistingColumn(t *testing.T) {
	path := fixture(t)

	if err := ApplyCCL(context.Background(), path, "['score'] = ['score'] + 1"); err != nil {
		t.Fatalf("ApplyCCL: %v", err)
	}

	dt, err := Read(context.Background(), path, ReadOptions{})
	if err != nil {
		t.Fatalf("reading back: %v", err)
	}
	if got, want := colValues(t, dt, "score"), []any{11.0, 21.0, 31.0, 41.0}; !reflect.DeepEqual(got, want) {
		t.Errorf("score: got %v, want %v", got, want)
	}
}

// An assignment into an int64 column is written back as int64, so a fractional
// result is truncated rather than widening the column.
func TestApplyCCL_KeepsTheColumnType(t *testing.T) {
	path := fixture(t)

	if err := ApplyCCL(context.Background(), path, "['num'] = ['num'] * 1.5"); err != nil {
		t.Fatalf("ApplyCCL: %v", err)
	}

	dt, err := Read(context.Background(), path, ReadOptions{})
	if err != nil {
		t.Fatalf("reading back: %v", err)
	}
	// 1.5, 3.0, 4.5, 6.0 written into an int64 column.
	if got, want := colValues(t, dt, "num"), []any{int64(1), int64(3), int64(4), int64(6)}; !reflect.DeepEqual(got, want) {
		t.Errorf("num: got %v, want %v", got, want)
	}
}

func TestApplyCCL_MultipleStatements(t *testing.T) {
	path := fixture(t)

	err := ApplyCCL(context.Background(), path, "NEW('sum') = ['num'] + ['score']; ['score'] = 0")
	if err != nil {
		t.Fatalf("ApplyCCL: %v", err)
	}

	dt, err := Read(context.Background(), path, ReadOptions{})
	if err != nil {
		t.Fatalf("reading back: %v", err)
	}
	if got, want := colValues(t, dt, "sum"), []any{11.0, 22.0, 33.0, 44.0}; !reflect.DeepEqual(got, want) {
		t.Errorf("sum: got %v, want %v", got, want)
	}
	if got, want := colValues(t, dt, "score"), []any{0.0, 0.0, 0.0, 0.0}; !reflect.DeepEqual(got, want) {
		t.Errorf("score: got %v, want %v", got, want)
	}
}

// ApplyCCL replaces the file through a temporary sibling. A script it cannot
// run must leave the original exactly as it was, and must not leave the
// temporary file behind.
func TestApplyCCL_FailureLeavesTheFileIntact(t *testing.T) {
	tests := []struct {
		name   string
		script string
	}{
		{name: "does not compile", script: "NEW('x') = ((("},
		{name: "assigns to a column that is not there", script: "['nope'] = 1"},
		{name: "reads a column that is not there", script: "NEW('x') = ['nope'] + 1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := fixture(t)
			before, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("reading the fixture: %v", err)
			}

			if err := ApplyCCL(context.Background(), path, tt.script); err == nil {
				t.Fatal("expected an error")
			}

			after, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("the original file is gone: %v", err)
			}
			if !reflect.DeepEqual(before, after) {
				t.Error("the original file changed despite the failure")
			}
			entries, err := os.ReadDir(filepath.Dir(path))
			if err != nil {
				t.Fatalf("listing the directory: %v", err)
			}
			if len(entries) != 1 {
				t.Errorf("leftover files in the directory: %v", entries)
			}
		})
	}
}

func TestApplyCCL_MissingFile(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "not-there.parquet")

	if err := ApplyCCL(context.Background(), missing, "NEW('x') = 1"); err == nil {
		t.Fatal("a missing file gave no error")
	}
}

// --- parquetContext ---------------------------------------------------------

// record builds the same four columns as the fixture as an in-memory Arrow
// record. The caller releases it.
func record(t *testing.T) arrow.Record {
	t.Helper()
	schema := arrow.NewSchema([]arrow.Field{
		{Name: "num", Type: arrow.PrimitiveTypes.Int64},
		{Name: "score", Type: arrow.PrimitiveTypes.Float64},
		{Name: "label", Type: arrow.BinaryTypes.String},
		{Name: "flag", Type: arrow.FixedWidthTypes.Boolean},
	}, nil)

	b := array.NewRecordBuilder(memory.DefaultAllocator, schema)
	defer b.Release()
	b.Field(0).(*array.Int64Builder).AppendValues([]int64{1, 2, 3, 4}, nil)
	b.Field(1).(*array.Float64Builder).AppendValues([]float64{10, 20, 30, 40}, nil)
	b.Field(2).(*array.StringBuilder).AppendValues([]string{"a", "b", "c", "d"}, nil)
	b.Field(3).(*array.BooleanBuilder).AppendValues([]bool{true, false, true, false}, nil)
	return b.NewRecord()
}

func newTestContext(t *testing.T) *parquetContext {
	t.Helper()
	rec := record(t)
	t.Cleanup(rec.Release)
	return newParquetContext(rec, []string{"num", "score", "label", "flag"})
}

// GetCol and GetColByName read the current row; the rest read the record.
func TestParquetContext_CurrentRowAccessors(t *testing.T) {
	c := newTestContext(t)

	if got := c.GetRowIndex(); got != 0 {
		t.Errorf("GetRowIndex on a fresh context: got %d, want 0", got)
	}
	if got := c.GetCol(0); got != int64(1) {
		t.Errorf("GetCol(0): got %v (%T), want int64(1)", got, got)
	}
	if got, want := c.GetCurrentRow(), []any{int64(1), 10.0, "a", true}; !reflect.DeepEqual(got, want) {
		t.Errorf("GetCurrentRow: got %v, want %v", got, want)
	}

	if err := c.SetRowIndex(2); err != nil {
		t.Fatalf("SetRowIndex(2): %v", err)
	}
	if got := c.GetRowIndex(); got != 2 {
		t.Errorf("GetRowIndex after SetRowIndex(2): got %d, want 2", got)
	}
	if got := c.GetCol(2); got != "c" {
		t.Errorf("GetCol(2) on row 2: got %v, want \"c\"", got)
	}
	v, err := c.GetColByName("score")
	if err != nil {
		t.Fatalf("GetColByName(\"score\"): %v", err)
	}
	if v != 30.0 {
		t.Errorf("GetColByName(\"score\") on row 2: got %v, want 30", v)
	}
}

// A column index past the end answers nil rather than failing, which is what
// lets a CCL expression referring to a column the file does not have evaluate
// at all.
func TestParquetContext_GetColPastTheEnd(t *testing.T) {
	c := newTestContext(t)

	if got := c.GetCol(4); got != nil {
		t.Errorf("GetCol(4) on a four-column record: got %v, want nil", got)
	}
}

func TestParquetContext_GetColByNameUnknown(t *testing.T) {
	c := newTestContext(t)

	_, err := c.GetColByName("nope")
	if err == nil {
		t.Fatal("an unknown column name gave no error")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error %q does not say the name was not found", err)
	}
}

// A row index that does not exist is refused, and the context keeps the row it
// was already on.
func TestParquetContext_SetRowIndexOutOfRange(t *testing.T) {
	c := newTestContext(t)
	if err := c.SetRowIndex(1); err != nil {
		t.Fatalf("SetRowIndex(1): %v", err)
	}

	for _, idx := range []int{-1, 4, 100} {
		if err := c.SetRowIndex(idx); err == nil {
			t.Errorf("SetRowIndex(%d) gave no error", idx)
		}
	}
	if got := c.GetRowIndex(); got != 1 {
		t.Errorf("a refused SetRowIndex moved the row: got %d, want 1", got)
	}
}

func TestParquetContext_GetCell(t *testing.T) {
	c := newTestContext(t)

	// One cell of each column type, to pin what a caller actually receives.
	tests := []struct {
		col  int
		row  int
		want any
	}{
		{col: 0, row: 0, want: int64(1)},
		{col: 1, row: 3, want: 40.0},
		{col: 2, row: 1, want: "b"},
		{col: 3, row: 2, want: true},
	}
	for _, tt := range tests {
		got, err := c.GetCell(tt.col, tt.row)
		if err != nil {
			t.Fatalf("GetCell(%d, %d): %v", tt.col, tt.row, err)
		}
		if got != tt.want {
			t.Errorf("GetCell(%d, %d): got %v (%T), want %v (%T)", tt.col, tt.row, got, got, tt.want, tt.want)
		}
	}

	for _, tt := range []struct{ col, row int }{{-1, 0}, {4, 0}, {0, -1}, {0, 4}} {
		if _, err := c.GetCell(tt.col, tt.row); err == nil {
			t.Errorf("GetCell(%d, %d) gave no error", tt.col, tt.row)
		}
	}
}

func TestParquetContext_GetCellByName(t *testing.T) {
	c := newTestContext(t)

	got, err := c.GetCellByName("label", 3)
	if err != nil {
		t.Fatalf("GetCellByName: %v", err)
	}
	if got != "d" {
		t.Errorf("GetCellByName(\"label\", 3): got %v, want \"d\"", got)
	}
	if _, err := c.GetCellByName("nope", 0); err == nil {
		t.Error("an unknown column name gave no error")
	}
	if _, err := c.GetCellByName("label", 99); err == nil {
		t.Error("a row index past the end gave no error")
	}
}

func TestParquetContext_GetRowAt(t *testing.T) {
	c := newTestContext(t)

	row, err := c.GetRowAt(1)
	if err != nil {
		t.Fatalf("GetRowAt(1): %v", err)
	}
	if got, want := row, any([]any{int64(2), 20.0, "b", false}); !reflect.DeepEqual(got, want) {
		t.Errorf("GetRowAt(1): got %v, want %v", got, want)
	}
	for _, idx := range []int{-1, 4} {
		if _, err := c.GetRowAt(idx); err == nil {
			t.Errorf("GetRowAt(%d) gave no error", idx)
		}
	}
}

func TestParquetContext_ColumnData(t *testing.T) {
	c := newTestContext(t)

	got, err := c.GetColData(0)
	if err != nil {
		t.Fatalf("GetColData(0): %v", err)
	}
	if want := []any{int64(1), int64(2), int64(3), int64(4)}; !reflect.DeepEqual(got, want) {
		t.Errorf("GetColData(0): got %v, want %v", got, want)
	}

	got, err = c.GetColDataByName("label")
	if err != nil {
		t.Fatalf("GetColDataByName(\"label\"): %v", err)
	}
	if want := []any{"a", "b", "c", "d"}; !reflect.DeepEqual(got, want) {
		t.Errorf("GetColDataByName(\"label\"): got %v, want %v", got, want)
	}

	for _, idx := range []int{-1, 4} {
		if _, err := c.GetColData(idx); err == nil {
			t.Errorf("GetColData(%d) gave no error", idx)
		}
	}
	if _, err := c.GetColDataByName("nope"); err == nil {
		t.Error("GetColDataByName on an unknown name gave no error")
	}
}

// GetAllData is column-major: every value of column 0, then column 1, and so on.
func TestParquetContext_GetAllData(t *testing.T) {
	c := newTestContext(t)

	got, err := c.GetAllData()
	if err != nil {
		t.Fatalf("GetAllData: %v", err)
	}
	want := []any{
		int64(1), int64(2), int64(3), int64(4),
		10.0, 20.0, 30.0, 40.0,
		"a", "b", "c", "d",
		true, false, true, false,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("GetAllData: got %v, want %v", got, want)
	}
}

func TestParquetContext_CountsAndIndexLookup(t *testing.T) {
	c := newTestContext(t)

	if got := c.GetColCount(); got != 4 {
		t.Errorf("GetColCount: got %d, want 4", got)
	}
	if got := c.GetRowCount(); got != 4 {
		t.Errorf("GetRowCount: got %d, want 4", got)
	}

	idx, err := c.GetColIndexByName("flag")
	if err != nil || idx != 3 {
		t.Errorf("GetColIndexByName(\"flag\"): got (%d, %v), want (3, nil)", idx, err)
	}
	if idx, err := c.GetColIndexByName("nope"); err == nil || idx != -1 {
		t.Errorf("GetColIndexByName on an unknown name: got (%d, %v), want (-1, an error)", idx, err)
	}
}

// Parquet has no row names, and the context says so rather than inventing one.
func TestParquetContext_RowNamesAreNotSupported(t *testing.T) {
	c := newTestContext(t)

	idx, err := c.GetRowIndexByName("anything")
	if err == nil {
		t.Fatal("GetRowIndexByName gave no error")
	}
	if idx != -1 {
		t.Errorf("GetRowIndexByName: got index %d, want -1", idx)
	}
	if !strings.Contains(err.Error(), "row names are not supported") {
		t.Errorf("error %q does not say row names are unsupported", err)
	}
}

// Every accessor has to survive a context built with no record at all, because
// that is what an empty batch produces.
func TestParquetContext_NoRecord(t *testing.T) {
	c := newParquetContext(nil, []string{"num"})

	if got := c.GetColCount(); got != 0 {
		t.Errorf("GetColCount: got %d, want 0", got)
	}
	if got := c.GetRowCount(); got != 0 {
		t.Errorf("GetRowCount: got %d, want 0", got)
	}
	if got := c.GetCol(0); got != nil {
		t.Errorf("GetCol(0): got %v, want nil", got)
	}
	if _, err := c.GetCell(0, 0); err == nil {
		t.Error("GetCell gave no error without a record")
	}
	if _, err := c.GetRowAt(0); err == nil {
		t.Error("GetRowAt gave no error without a record")
	}
	if _, err := c.GetColData(0); err == nil {
		t.Error("GetColData gave no error without a record")
	}
	if _, err := c.GetAllData(); err == nil {
		t.Error("GetAllData gave no error without a record")
	}
	if err := c.SetRowIndex(0); err == nil {
		t.Error("SetRowIndex gave no error without a record")
	}
}

func TestResolveAssignTarget(t *testing.T) {
	tests := []struct {
		name     string
		target   string
		colNames []string
		want     string
		ok       bool
	}{
		{name: "quoted name", target: "'score'", colNames: []string{"num", "score"}, want: "score", ok: true},
		{name: "quoted name that is not there", target: "'nope'", colNames: []string{"num", "score"}, want: "", ok: false},
		{name: "column letter", target: "B", colNames: []string{"num", "score"}, want: "score", ok: true},
		{name: "column letter past the end", target: "Z", colNames: []string{"num", "score"}, want: "", ok: false},
		// "score" reads as a column reference too, but its index is far past the
		// end, so it falls through to the name match.
		{name: "bare name", target: "score", colNames: []string{"num", "score"}, want: "score", ok: true},
		{name: "bare name with a digit", target: "n1", colNames: []string{"n1"}, want: "n1", ok: true},
		{name: "bare name that is nowhere", target: "n1", colNames: []string{"num"}, want: "", ok: false},
		// The trap: a bare target is read as a column *letter* before it is read
		// as a column *name*. Here "B" is the name of column 0 and also the
		// reference for column 1, and the reference wins.
		{name: "a name that is also an in-range letter", target: "B", colNames: []string{"B", "other"}, want: "other", ok: true},
		// The quoted form is unambiguous and picks the column actually named "B".
		{name: "the same name quoted", target: "'B'", colNames: []string{"B", "other"}, want: "B", ok: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := resolveAssignTarget(tt.target, tt.colNames)
			if got != tt.want || ok != tt.ok {
				t.Errorf("resolveAssignTarget(%q, %v): got (%q, %v), want (%q, %v)", tt.target, tt.colNames, got, ok, tt.want, tt.ok)
			}
		})
	}
}
