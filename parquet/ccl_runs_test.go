package parquet

import (
	"context"
	"reflect"
	"testing"

	"github.com/HazelnutParadise/insyra/internal/ccl"
)

// A stage of ApplyCCL's pipeline has to hold rows as columns of Go values and
// hand every statement the same accessors a loaded table gives, whatever the
// rows were read from. These tests hold that seam: a context built over a run
// answers as one built over a record does, the pipeline carries each run
// through every statement in order, and what leaves the last stage is what the
// file ends up holding.

// runContextFixture is four rows of three columns starting at the file's row 10,
// with a missing value in the first column and a nil in the second, so the
// accessors are asked for cells that are absent as well as cells that are not.
func runContextFixture() cclRun {
	return cclRun{
		offset: 10,
		names:  []string{"A", "B", "C"},
		cols: [][]any{
			{1.0, nil, 3.0, 4.0},
			{"a", "b", "c", "d"},
			{int64(1), int64(2), int64(3), int64(4)},
		},
	}
}

func TestRunContextReadsItsColumns(t *testing.T) {
	r := runContextFixture()
	c := newRunContext(r)

	if got := c.GetColCount(); got != 3 {
		t.Errorf("GetColCount: got %d, want 3", got)
	}
	if got := c.GetRowCount(); got != 4 {
		t.Errorf("GetRowCount: got %d, want 4", got)
	}
	if got := c.GetRowIndex(); got != 0 {
		t.Errorf("GetRowIndex on a fresh context: got %d, want 0", got)
	}
	if got, want := c.GlobalRowIndex(), 10; got != want {
		t.Errorf("GlobalRowIndex on a fresh context: got %d, want %d", got, want)
	}

	for _, tt := range []struct {
		col, row int
		want     any
	}{
		{col: 0, row: 0, want: 1.0},
		{col: 0, row: 1, want: nil},
		{col: 1, row: 3, want: "d"},
		{col: 2, row: 2, want: int64(3)},
	} {
		got, err := c.GetCell(tt.col, tt.row)
		if err != nil {
			t.Fatalf("GetCell(%d, %d): %v", tt.col, tt.row, err)
		}
		if got != tt.want {
			t.Errorf("GetCell(%d, %d): got %v (%T), want %v (%T)", tt.col, tt.row, got, got, tt.want, tt.want)
		}
	}

	for _, tt := range []struct{ col, row int }{{-1, 0}, {3, 0}, {0, -1}, {0, 4}} {
		if _, err := c.GetCell(tt.col, tt.row); err == nil {
			t.Errorf("GetCell(%d, %d) gave no error", tt.col, tt.row)
		}
	}

	got, err := c.GetCellByName("B", 2)
	if err != nil {
		t.Fatalf("GetCellByName(\"B\", 2): %v", err)
	}
	if got != "c" {
		t.Errorf("GetCellByName(\"B\", 2): got %v, want \"c\"", got)
	}
	if _, err := c.GetCellByName("nope", 0); err == nil {
		t.Error("GetCellByName on an unknown name gave no error")
	}

	row, err := c.GetRowAt(2)
	if err != nil {
		t.Fatalf("GetRowAt(2): %v", err)
	}
	if want := []any{3.0, "c", int64(3)}; !reflect.DeepEqual(row, want) {
		t.Errorf("GetRowAt(2): got %v, want %v", row, want)
	}
	for _, idx := range []int{-1, 4} {
		if _, err := c.GetRowAt(idx); err == nil {
			t.Errorf("GetRowAt(%d) gave no error", idx)
		}
	}

	// GetColData hands out a copy: a statement that writes into the column it
	// read must not change what the next statement reads.
	col, err := c.GetColData(0)
	if err != nil {
		t.Fatalf("GetColData(0): %v", err)
	}
	if want := []any{1.0, nil, 3.0, 4.0}; !reflect.DeepEqual(col, want) {
		t.Errorf("GetColData(0): got %v, want %v", col, want)
	}
	col[0] = 99.0
	if again, err := c.GetColData(0); err != nil || again[0] != 1.0 {
		t.Errorf("GetColData(0) after the caller wrote into it: got (%v, %v), want the run's own 1", again, err)
	}

	if col, err := c.GetColDataByName("B"); err != nil || !reflect.DeepEqual(col, []any{"a", "b", "c", "d"}) {
		t.Errorf("GetColDataByName(\"B\"): got (%v, %v), want [a b c d]", col, err)
	}
	for _, idx := range []int{-1, 3} {
		if _, err := c.GetColData(idx); err == nil {
			t.Errorf("GetColData(%d) gave no error", idx)
		}
	}
	if _, err := c.GetColDataByName("nope"); err == nil {
		t.Error("GetColDataByName on an unknown name gave no error")
	}

	all, err := c.GetAllData()
	if err != nil {
		t.Fatalf("GetAllData: %v", err)
	}
	want := []any{
		1.0, nil, 3.0, 4.0,
		"a", "b", "c", "d",
		int64(1), int64(2), int64(3), int64(4),
	}
	if !reflect.DeepEqual(all, want) {
		t.Errorf("GetAllData: got %v, want %v", all, want)
	}

	// The current row follows the row index, and # is counted from the file.
	if err := c.SetRowIndex(2); err != nil {
		t.Fatalf("SetRowIndex(2): %v", err)
	}
	if got, want := c.GlobalRowIndex(), 12; got != want {
		t.Errorf("GlobalRowIndex on the run's third row: got %d, want %d", got, want)
	}
	if got, want := c.GetCurrentRow(), []any{3.0, "c", int64(3)}; !reflect.DeepEqual(got, want) {
		t.Errorf("GetCurrentRow: got %v, want %v", got, want)
	}
	if got := c.GetCol(2); got != int64(3) {
		t.Errorf("GetCol(2): got %v (%T), want int64(3)", got, got)
	}
	if v, err := c.GetColByName("A"); err != nil || v != 3.0 {
		t.Errorf("GetColByName(\"A\"): got (%v, %v), want (3, nil)", v, err)
	}
	for _, idx := range []int{-1, 4} {
		if err := c.SetRowIndex(idx); err == nil {
			t.Errorf("SetRowIndex(%d) gave no error", idx)
		}
	}
}

// A column a NEW statement created is a column of the run afterwards, and the
// run the caller handed over is left as it was.
func TestRunContextAddsAndReplacesColumns(t *testing.T) {
	r := runContextFixture()
	c := newRunContext(r)

	if err := c.SetRowIndex(1); err != nil {
		t.Fatalf("SetRowIndex(1): %v", err)
	}

	c.addColumn("D", []any{10.0, 11.0, 12.0, 13.0})
	if got := c.GetColCount(); got != 4 {
		t.Errorf("GetColCount after addColumn: got %d, want 4", got)
	}
	if idx, err := c.GetColIndexByName("D"); err != nil || idx != 3 {
		t.Errorf("GetColIndexByName(\"D\"): got (%d, %v), want (3, nil)", idx, err)
	}
	if col, err := c.GetColDataByName("D"); err != nil || !reflect.DeepEqual(col, []any{10.0, 11.0, 12.0, 13.0}) {
		t.Errorf("GetColDataByName(\"D\"): got (%v, %v), want [10 11 12 13]", col, err)
	}
	if got, want := c.GetCurrentRow(), []any{nil, "b", int64(2), 11.0}; !reflect.DeepEqual(got, want) {
		t.Errorf("GetCurrentRow after addColumn: got %v, want %v", got, want)
	}
	if len(r.names) != 3 {
		t.Errorf("addColumn changed the run it was given: it names %v, want the file's three columns", r.names)
	}

	c.setColumn(1, []any{"x", "y", "z", "w"})
	got, err := c.GetCell(1, 1)
	if err != nil {
		t.Fatalf("GetCell(1, 1) after setColumn: %v", err)
	}
	if got != "y" {
		t.Errorf("GetCell(1, 1) after setColumn: got %v, want \"y\"", got)
	}
	if got, want := c.GetCurrentRow(), []any{nil, "y", int64(2), 11.0}; !reflect.DeepEqual(got, want) {
		t.Errorf("GetCurrentRow after setColumn: got %v, want %v", got, want)
	}
	if want := []any{"a", "b", "c", "d"}; !reflect.DeepEqual(r.cols[1], want) {
		t.Errorf("setColumn changed the run it was given: column B is %v, want %v", r.cols[1], want)
	}

	// The rows the context hands back are the ones the statements left, with
	// what they wrote and where they started in the file.
	out := c.run()
	if out.offset != r.offset {
		t.Errorf("run().offset: got %d, want %d", out.offset, r.offset)
	}
	if out.rows() != 4 {
		t.Errorf("run().rows(): got %d, want 4", out.rows())
	}
	if !reflect.DeepEqual(out.names, []string{"A", "B", "C", "D"}) {
		t.Errorf("run().names: got %v, want [A B C D]", out.names)
	}
	if !reflect.DeepEqual(out.cols[3], []any{10.0, 11.0, 12.0, 13.0}) {
		t.Errorf("run() does not hold the column the statement added: %v", out.cols[3])
	}
	if !reflect.DeepEqual(out.cols[1], []any{"x", "y", "z", "w"}) {
		t.Errorf("run() does not hold what the statement wrote: %v", out.cols[1])
	}
}

// Each statement of a script is a stage, so a run pushed in comes out with every
// statement applied to it, in order, and where it started in the file.
func TestPipelinePassesRunsThroughEveryStage(t *testing.T) {
	nodes, err := ccl.CompileMultiline("NEW('c') = A * 2; NEW('d') = ['c'] + 1")
	if err != nil {
		t.Fatalf("compiling the script: %v", err)
	}

	p, err := newPipeline(nodes, 6, []string{"A"}, nil)
	if err != nil {
		t.Fatalf("newPipeline: %v", err)
	}

	// Two runs of the same six rows, so the second one continues where the
	// first ended the way the second batch of a file does.
	pushed := []cclRun{
		{offset: 0, names: []string{"A"}, cols: [][]any{{1.0, 2.0, 3.0}}},
		{offset: 3, names: []string{"A"}, cols: [][]any{{4.0, 5.0, 6.0}}},
	}

	var got []cclRun
	for _, r := range pushed {
		outs, err := p.push(r)
		if err != nil {
			t.Fatalf("pushing the run at row %d: %v", r.offset, err)
		}
		if len(outs) != 1 {
			t.Fatalf("pushing the run at row %d gave %d runs, want 1", r.offset, len(outs))
		}
		got = append(got, outs...)
	}

	// A statement that finishes every row at once holds nothing back, so the
	// end of the file has nothing left to flush.
	tail, err := p.flush()
	if err != nil {
		t.Fatalf("flushing the pipeline: %v", err)
	}
	if len(tail) != 0 {
		t.Errorf("flush gave %d runs, want none", len(tail))
	}

	want := []struct {
		offset    int
		a, cc, dd []any
	}{
		{
			offset: 0,
			a:      []any{1.0, 2.0, 3.0},
			cc:     []any{2.0, 4.0, 6.0},
			dd:     []any{3.0, 5.0, 7.0},
		},
		{
			offset: 3,
			a:      []any{4.0, 5.0, 6.0},
			cc:     []any{8.0, 10.0, 12.0},
			dd:     []any{9.0, 11.0, 13.0},
		},
	}

	if len(got) != len(want) {
		t.Fatalf("the pipeline produced %d runs, want %d", len(got), len(want))
	}
	for i, w := range want {
		out := got[i]
		if out.offset != w.offset {
			t.Errorf("run %d came out at row %d, want %d", i, out.offset, w.offset)
		}
		if out.rows() != 3 {
			t.Errorf("run %d holds %d rows, want 3", i, out.rows())
		}
		if !reflect.DeepEqual(out.names, []string{"A", "c", "d"}) {
			t.Errorf("run %d names %v, want [A c d]", i, out.names)
		}
		for col, want := range [][]any{w.a, w.cc, w.dd} {
			if !reflect.DeepEqual(out.cols[col], want) {
				t.Errorf("run %d column %q is %v, want %v", i, out.names[col], out.cols[col], want)
			}
		}
	}

	// The runs the caller pushed are the pipeline's input, not its output.
	for _, r := range pushed {
		if !reflect.DeepEqual(r.names, []string{"A"}) || len(r.cols) != 1 {
			t.Errorf("push changed the run at row %d: it holds %d columns named %v", r.offset, len(r.cols), r.names)
		}
	}
}

// What leaves the pipeline is what the file ends up holding: a script whose
// statements build on each other, across the batches the file arrives in, has
// to leave the loaded table's answer behind.
func TestApplyCCLWritesWhatThePipelineFinishes(t *testing.T) {
	const script = "NEW('c') = A * 2; ['A'] = ['c'] - A; NEW('d') = # + COUNT(A)"

	// A file of its own: ApplyCCL writes back to the path it was given, and the
	// expectation is the file before it was touched.
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
}

// A run with no rows still goes through every stage, and comes out with the
// columns a run with rows would have: a NEW statement's column is one of them,
// or a statement after it that writes into that column has nothing to write to.
func TestPipelineAddsANewColumnToAnEmptyRun(t *testing.T) {
	nodes, err := ccl.CompileMultiline("NEW('c') = A * 2; ['c'] = LAG(['c'], 1)")
	if err != nil {
		t.Fatalf("compiling the script: %v", err)
	}

	for _, tt := range []struct {
		name  string
		empty cclRun
	}{
		// A run read from a record with no rows: a column of no values each.
		{name: "a column of no values", empty: cclRun{names: []string{"A"}, cols: [][]any{{}}}},
		// A run that only knows its names, which is what the tail of a batch that
		// has been taken whole is.
		{name: "names only", empty: cclRun{names: []string{"A"}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			p, err := newPipeline(nodes, 3, []string{"A"}, nil)
			if err != nil {
				t.Fatalf("newPipeline: %v", err)
			}

			outs, err := p.push(tt.empty)
			if err != nil {
				t.Fatalf("pushing a run of no rows: %v", err)
			}
			if len(outs) != 1 {
				t.Fatalf("pushing a run of no rows gave %d runs, want 1", len(outs))
			}
			if outs[0].rows() != 0 {
				t.Errorf("the run of no rows came out with %d rows", outs[0].rows())
			}
			if !reflect.DeepEqual(outs[0].names, []string{"A", "c"}) {
				t.Errorf("the run of no rows came out naming %v, want [A c]", outs[0].names)
			}
			if len(outs[0].cols) != 2 {
				t.Errorf("the run of no rows came out with %d columns, want 2", len(outs[0].cols))
			}

			// The rows that follow are answered as if the empty run had not been
			// there, and the first of them has no row before it for LAG.
			got, err := p.push(cclRun{names: []string{"A"}, cols: [][]any{{1.0, 2.0, 3.0}}})
			if err != nil {
				t.Fatalf("pushing the rows: %v", err)
			}
			tail, err := p.flush()
			if err != nil {
				t.Fatalf("flushing the pipeline: %v", err)
			}
			got = append(got, tail...)
			if len(got) != 1 {
				t.Fatalf("the rows came out as %d runs, want 1", len(got))
			}
			out := got[0]
			if !reflect.DeepEqual(out.names, []string{"A", "c"}) {
				t.Errorf("the rows came out naming %v, want [A c]", out.names)
			}
			if want := []any{1.0, 2.0, 3.0}; !reflect.DeepEqual(out.cols[0], want) {
				t.Errorf("column A is %v, want %v", out.cols[0], want)
			}
			if want := []any{nil, 2.0, 4.0}; !reflect.DeepEqual(out.cols[1], want) {
				t.Errorf("column c is %v, want %v", out.cols[1], want)
			}
		})
	}
}
