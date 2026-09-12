package isr

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

// quietTest silences the warnings these tests provoke and restores the global
// config. error_test.go has the same helper, but it lives in the external
// isr_test package and this file has to be in-package: dt and dl are
// unexported, so an external test cannot name them.
func quietTest(t *testing.T) {
	t.Helper()
	level := insyra.Config.GetLogLevel()
	panicOnError := insyra.Config.GetPanicOnError()
	insyra.Config.SetLogLevel(insyra.LogLevelFatal)
	insyra.Config.SetPanicOnError(false)
	t.Cleanup(func() {
		insyra.Config.SetLogLevel(level)
		insyra.Config.SetPanicOnError(panicOnError)
	})
}

// The README calls isr the recommended entry point for new code, and it sat at
// 44.2%: the whole DL surface except From, every selector on DT, two thirds of
// DT.From's type switch, five of the window wrappers, Name(), and CCL had never
// been called. This file is in-package because dt and dl are unexported, so an
// external test cannot name them.

func sampleDT() *dt {
	return DT.From(DLs{
		DL.From(1, 2, 3).SetName("A"),
		DL.From("x", "y", "z").SetName("B"),
	})
}

// --- DL ---------------------------------------------------------------------

func TestDL_OfAtPush(t *testing.T) {
	quietTest(t)

	l := DL.Of(1, 2, 3)
	if got := l.Len(); got != 3 {
		t.Fatalf("DL.Of gave a list of %d, want 3", got)
	}
	if got := l.At(0); got != 1 {
		t.Errorf("At(0): got %v, want 1", got)
	}
	if got := l.At(2); got != 3 {
		t.Errorf("At(2): got %v, want 3", got)
	}
	// Out of range records the failure and answers nil rather than panicking.
	if got := l.At(99); got != nil {
		t.Errorf("At(99): got %v, want nil", got)
	}

	l.Push(4).Push(5, 6)
	if got, want := l.Data(), []any{1, 2, 3, 4, 5, 6}; !reflect.DeepEqual(got, want) {
		t.Errorf("after Push: got %v, want %v", got, want)
	}
}

// From flattens what it is given, so a slice argument becomes its elements.
func TestDL_FromFlattens(t *testing.T) {
	quietTest(t)

	if got, want := DL.From([]int{1, 2}, 3).Data(), []any{1, 2, 3}; !reflect.DeepEqual(got, want) {
		t.Errorf("DL.From([]int{1,2}, 3): got %v, want %v", got, want)
	}
}

func TestPtrDL_StillWorks(t *testing.T) {
	quietTest(t)

	l := PtrDL(insyra.NewDataList(1, 2))
	if l == nil || l.Len() != 2 {
		t.Fatalf("PtrDL gave %v", l)
	}
}

// --- DT.From ----------------------------------------------------------------

func TestDT_From_SliceForms(t *testing.T) {
	quietTest(t)

	tests := []struct {
		name string
		in   any
		rows int
		cols int
	}{
		{name: "[][]any", in: [][]any{{1, "a"}, {2, "b"}}, rows: 2, cols: 2},
		{name: "[][]int", in: [][]int{{1, 2}, {3, 4}}, rows: 2, cols: 2},
		{name: "[][]float64", in: [][]float64{{1.5}, {2.5}}, rows: 2, cols: 1},
		{name: "[][]string", in: [][]string{{"a", "b"}}, rows: 1, cols: 2},
		{name: "[][]bool", in: [][]bool{{true}, {false}}, rows: 2, cols: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			table := DT.From(tt.in)
			rows, cols := table.Size()
			if rows != tt.rows || cols != tt.cols {
				t.Errorf("size: got %dx%d, want %dx%d", rows, cols, tt.rows, tt.cols)
			}
		})
	}
}

func TestDT_From_DataListForms(t *testing.T) {
	quietTest(t)

	t.Run("*insyra.DataList", func(t *testing.T) {
		table := DT.From(insyra.NewDataList(1, 2, 3).SetName("A"))
		if rows, cols := table.Size(); rows != 3 || cols != 1 {
			t.Errorf("size: got %dx%d, want 3x1", rows, cols)
		}
	})
	t.Run("*dl", func(t *testing.T) {
		table := DT.From(DL.From(1, 2, 3))
		if rows, cols := table.Size(); rows != 3 || cols != 1 {
			t.Errorf("size: got %dx%d, want 3x1", rows, cols)
		}
	})
	t.Run("[]*insyra.DataList", func(t *testing.T) {
		table := DT.From([]*insyra.DataList{
			insyra.NewDataList(1, 2).SetName("A"),
			insyra.NewDataList(3, 4).SetName("B"),
		})
		if rows, cols := table.Size(); rows != 2 || cols != 2 {
			t.Errorf("size: got %dx%d, want 2x2", rows, cols)
		}
	})
	t.Run("[]dl", func(t *testing.T) {
		table := DT.From([]dl{*DL.From(1, 2), *DL.From(3, 4)})
		if rows, cols := table.Size(); rows != 2 || cols != 2 {
			t.Errorf("size: got %dx%d, want 2x2", rows, cols)
		}
	})
}

func TestDT_From_RowsAndCols(t *testing.T) {
	quietTest(t)

	t.Run("Rows", func(t *testing.T) {
		table := DT.From(Rows{
			Row{"A": 1, "B": "x"},
			Row{"A": 2, "B": "y"},
		})
		if rows, cols := table.Size(); rows != 2 || cols != 2 {
			t.Errorf("size: got %dx%d, want 2x2", rows, cols)
		}
	})
	t.Run("Col", func(t *testing.T) {
		table := DT.From(Col{"A": 1, "B": 2})
		if rows, _ := table.Size(); rows == 0 {
			t.Error("a Col produced an empty table")
		}
	})
	t.Run("Cols", func(t *testing.T) {
		table := DT.From(Cols{Col{"A": 1}, Col{"A": 2}})
		if rows, _ := table.Size(); rows == 0 {
			t.Error("Cols produced an empty table")
		}
	})
	t.Run("map[string]any", func(t *testing.T) {
		table := DT.From(map[string]any{"A": 1, "B": 2})
		if _, cols := table.Size(); cols == 0 {
			t.Error("a string map produced no columns")
		}
	})
	// map[int]any is listed as a supported input but never works — it
	// stringifies the key to "0" where AppendRowsByColIndex wants an Excel
	// index. Pinned in the change that fixes it, not here.
}

// Reading a CSV that is there, not just one that is missing.
func TestDT_From_CSVFile(t *testing.T) {
	quietTest(t)

	path := filepath.Join(t.TempDir(), "in.csv")
	if err := os.WriteFile(path, []byte("A,B\n1,x\n2,y\n"), 0o600); err != nil {
		t.Fatalf("writing the fixture: %v", err)
	}

	table := DT.From(CSV{FilePath: path, InputOpts: CSV_inOpts{FirstRow2ColNames: true}})
	if table.Err() != nil {
		t.Fatalf("reading the CSV: %v", table.Err())
	}
	rows, cols := table.Size()
	if rows != 2 || cols != 2 {
		t.Errorf("size: got %dx%d, want 2x2", rows, cols)
	}
	if got := table.At(0, "A"); got != int64(1) && got != 1 {
		t.Errorf("first cell: got %v (%T), want 1", got, got)
	}
}

func TestDT_From_Excel_EmptyPath(t *testing.T) {
	quietTest(t)

	table := DT.From(Excel{})
	if table.Err() == nil {
		t.Error("an Excel source with no path recorded no error")
	}
}

func TestDT_Of(t *testing.T) {
	quietTest(t)

	table := DT.Of(DLs{DL.From(1, 2).SetName("A")})
	if rows, _ := table.Size(); rows != 2 {
		t.Errorf("DT.Of produced %d rows, want 2", rows)
	}
}

// --- Selectors --------------------------------------------------------------

func TestDT_ColSelectors(t *testing.T) {
	quietTest(t)
	table := sampleDT()

	if got := table.Col(0).Data(); !reflect.DeepEqual(got, []any{1, 2, 3}) {
		t.Errorf("Col(0): got %v", got)
	}
	if got := table.Col("A").Data(); !reflect.DeepEqual(got, []any{1, 2, 3}) {
		t.Errorf("Col(\"A\") by Excel index: got %v", got)
	}
	if got := table.Col(Name("B")).Data(); !reflect.DeepEqual(got, []any{"x", "y", "z"}) {
		t.Errorf("Col(Name(\"B\")): got %v", got)
	}

	// A selector that is not there, and one of the wrong kind, both record the
	// failure on the returned list rather than handing back nil.
	if l := table.Col(99); l == nil || l.Err() == nil {
		t.Error("Col(99) did not record an error")
	}
	if l := table.Col(1.5); l == nil || l.Err() == nil {
		t.Error("Col with a float selector did not record an error")
	}
}

func TestDT_RowSelectors(t *testing.T) {
	quietTest(t)
	table := sampleDT()
	table.SetRowNames([]string{"r0", "r1", "r2"})

	if got := table.Row(0).Data(); !reflect.DeepEqual(got, []any{1, "x"}) {
		t.Errorf("Row(0): got %v", got)
	}
	if got := table.Row(Name("r1")).Data(); !reflect.DeepEqual(got, []any{2, "y"}) {
		t.Errorf("Row(Name(\"r1\")): got %v", got)
	}

	if l := table.Row(99); l == nil || l.Err() == nil {
		t.Error("Row(99) did not record an error")
	}
	if l := table.Row("nope"); l == nil || l.Err() == nil {
		t.Error("Row with a string selector did not record an error")
	}
}

func TestDT_At(t *testing.T) {
	quietTest(t)
	table := sampleDT()
	table.SetRowNames([]string{"r0", "r1", "r2"})

	tests := []struct {
		name string
		row  any
		col  any
		want any
	}{
		{name: "int row, int col", row: 1, col: 0, want: 2},
		{name: "int row, string col", row: 1, col: "B", want: "y"},
		{name: "int row, named col", row: 1, col: Name("A"), want: 2},
		{name: "named row, int col", row: Name("r2"), col: 0, want: 3},
		{name: "named row, string col", row: Name("r0"), col: "B", want: "x"},
		{name: "named row, named col", row: Name("r0"), col: Name("A"), want: 1},
		// An unusable selector answers nil rather than panicking.
		{name: "unusable col", row: 0, col: 1.5, want: nil},
		{name: "unusable row with int col", row: 1.5, col: 0, want: nil},
		{name: "unusable row with string col", row: 1.5, col: "A", want: nil},
		{name: "unusable row with named col", row: 1.5, col: Name("A"), want: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := table.At(tt.row, tt.col); got != tt.want {
				t.Errorf("At(%v, %v) = %v, want %v", tt.row, tt.col, got, tt.want)
			}
		})
	}
}

// --- Push -------------------------------------------------------------------

func TestDT_PushForms(t *testing.T) {
	quietTest(t)

	t.Run("*insyra.DataList", func(t *testing.T) {
		table := sampleDT().Push(insyra.NewDataList(7, 8, 9).SetName("C"))
		if _, cols := table.Size(); cols != 3 {
			t.Errorf("columns: got %d, want 3", cols)
		}
	})
	t.Run("[]*insyra.DataList", func(t *testing.T) {
		table := sampleDT().Push([]*insyra.DataList{
			insyra.NewDataList(7, 8, 9).SetName("C"),
			insyra.NewDataList(1, 1, 1).SetName("D"),
		})
		if _, cols := table.Size(); cols != 4 {
			t.Errorf("columns: got %d, want 4", cols)
		}
	})
	t.Run("[]dl", func(t *testing.T) {
		table := sampleDT().Push([]dl{*DL.From(7, 8, 9)})
		if _, cols := table.Size(); cols != 3 {
			t.Errorf("columns: got %d, want 3", cols)
		}
	})
	t.Run("Rows", func(t *testing.T) {
		table := sampleDT().Push(Rows{Row{"A": 4, "B": "w"}})
		if rows, _ := table.Size(); rows != 4 {
			t.Errorf("rows: got %d, want 4", rows)
		}
	})
	t.Run("Col", func(t *testing.T) {
		table := sampleDT().Push(Col{"A": 4})
		if rows, _ := table.Size(); rows == 0 {
			t.Error("pushing a Col emptied the table")
		}
	})
	t.Run("unsupported", func(t *testing.T) {
		table := sampleDT().Push(3.14)
		if table.Err() == nil {
			t.Error("pushing an unsupported type recorded no error")
		}
	})
}

// --- name keys --------------------------------------------------------------

// A Row keyed by isr.Name goes in by column name rather than by position, which
// is the branch of fromRowToDT that had never run.
func TestDT_RowWithNameKeys(t *testing.T) {
	quietTest(t)

	table := DT.From(Row{Name("A"): 1, Name("B"): "x"})
	if table.Err() != nil {
		t.Fatalf("building from a name-keyed Row: %v", table.Err())
	}
	if _, cols := table.Size(); cols != 2 {
		t.Errorf("columns: got %d, want 2", cols)
	}
	names := table.ColNames()
	found := map[string]bool{}
	for _, n := range names {
		found[n] = true
	}
	if !found["A"] || !found["B"] {
		t.Errorf("column names: got %v, want A and B", names)
	}
}

// A Row whose keys are neither all ints nor all strings is refused.
func TestDT_RowWithMixedKeys(t *testing.T) {
	quietTest(t)

	table := DT.From(Row{"A": 1, 0: 2})
	if table.Err() == nil {
		t.Error("a Row with mixed key types recorded no error")
	}
}

// --- window wrappers --------------------------------------------------------

func TestDT_WindowWrappers(t *testing.T) {
	quietTest(t)

	build := func() *dt {
		return DT.From(DLs{DL.From(1.0, 3.0, 2.0, 6.0).SetName("v")})
	}

	tests := []struct {
		name string
		get  func(*dt) *insyra.DataList
		want []any
	}{
		{name: "Diff", get: func(t *dt) *insyra.DataList { return t.Diff("v", 1) },
			want: []any{nil, 2.0, -1.0, 4.0}},
		{name: "PctChange", get: func(t *dt) *insyra.DataList { return t.PctChange("v", 1) },
			want: []any{nil, 2.0, -1.0 / 3.0, 2.0}},
		{name: "CumSum", get: func(t *dt) *insyra.DataList { return t.CumSum("v") },
			want: []any{1.0, 4.0, 6.0, 12.0}},
		{name: "CumProd", get: func(t *dt) *insyra.DataList { return t.CumProd("v") },
			want: []any{1.0, 3.0, 6.0, 36.0}},
		{name: "CumMax", get: func(t *dt) *insyra.DataList { return t.CumMax("v") },
			want: []any{1.0, 3.0, 3.0, 6.0}},
		{name: "CumMin", get: func(t *dt) *insyra.DataList { return t.CumMin("v") },
			want: []any{1.0, 1.0, 1.0, 1.0}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.get(build())
			if got == nil {
				t.Fatal("the wrapper returned nil")
			}
			if !reflect.DeepEqual(got.Data(), tt.want) {
				t.Errorf("got %v, want %v", got.Data(), tt.want)
			}
		})
	}
}

// --- CCL --------------------------------------------------------------------

func TestDT_CCL(t *testing.T) {
	quietTest(t)

	table := DT.From(DLs{
		DL.From(1, 2, 3).SetName("A"),
		DL.From(10, 20, 30).SetName("B"),
	}).CCL("NEW('C') = A + B")

	if table.Err() != nil {
		t.Fatalf("CCL: %v", table.Err())
	}
	col := table.GetColByName("C")
	if col == nil {
		t.Fatal("CCL did not create column C")
	}
	if got, want := col.Data(), []any{11.0, 22.0, 33.0}; !reflect.DeepEqual(got, want) {
		t.Errorf("C: got %v, want %v", got, want)
	}
}

func TestDT_CCL_BadStatement(t *testing.T) {
	quietTest(t)

	table := sampleDT().CCL("NEW('C') = (((")
	if table == nil {
		t.Fatal("CCL returned nil")
	}
	if table.Err() == nil {
		t.Error("a statement that does not compile recorded no error")
	}
}

// --- Use* -------------------------------------------------------------------

// A dt or dl value whose embedded pointer is nil must still come back usable.
func TestUse_ValueWithNilPointer(t *testing.T) {
	quietTest(t)

	table := UseDT(dt{})
	if table == nil {
		t.Fatal("UseDT(dt{}) returned nil")
	}
	if table.Err() == nil {
		t.Error("UseDT on a value with a nil table recorded no error")
	}

	list := UseDL(dl{})
	if list == nil {
		t.Fatal("UseDL(dl{}) returned nil")
	}
	if list.Err() == nil {
		t.Error("UseDL on a value with a nil list recorded no error")
	}
}

func TestPtrDT_StillWorks(t *testing.T) {
	quietTest(t)

	table := PtrDT(insyra.NewDataTable(insyra.NewDataList(1, 2)))
	if table == nil {
		t.Fatal("PtrDT returned nil")
	}
	if rows, _ := table.Size(); rows != 2 {
		t.Errorf("rows: got %d, want 2", rows)
	}
}
