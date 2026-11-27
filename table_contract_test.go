package insyra

import (
	"reflect"
	"slices"
	"testing"
)

// The tests below pin the row order, missing values and shape that every
// table operation promises, so a backend or a rewrite that changes one of
// them fails here rather than in a user's report (#415 part 4). Each compares
// rows in order: none of these operations is documented as unordered.

func rowsOf(dt *DataTable) [][]any { return dt.To2DSlice() }

func TestContractGroupByFirstAndLastSkipMissingValues(t *testing.T) {
	g := NewDataTable(NewDataList("b", "a", "b", "a", "b").SetName("k"), NewDataList(nil, 1, 5, nil, 7).SetName("v"))
	out := g.GroupBy("k").Aggregate(
		AggregateConfig{SourceCol: "v", Op: OpFirst},
		AggregateConfig{SourceCol: "v", Op: OpLast},
	)
	if err := out.Err(); err != nil {
		t.Fatal(err)
	}
	// Groups come in the order their key is first seen; first and last are
	// the first and last value that is not nil, in row order.
	want := [][]any{{"b", 5, 7}, {"a", 1, 1}}
	if got := rowsOf(out); !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	if !slices.Equal(out.ColNames(), []string{"k", "v_first", "v_last"}) {
		t.Errorf("columns %q", out.ColNames())
	}
}

func TestContractMergeWithDuplicateKeys(t *testing.T) {
	left := NewDataTable(NewDataList("a", "b", "a").SetName("k"), NewDataList(1, 2, 3).SetName("x"))
	right := NewDataTable(NewDataList("a", "a", "c").SetName("k"), NewDataList(10, 20, 30).SetName("y"))
	// Every left row with a key meets every right row with that key, the left
	// rows in their order and, for each, the right rows in theirs.
	inner, err := left.Merge(right, MergeDirectionHorizontal, MergeModeInner, "k")
	if err != nil {
		t.Fatal(err)
	}
	want := [][]any{{"a", 1, 10}, {"a", 1, 20}, {"a", 3, 10}, {"a", 3, 20}}
	if got := rowsOf(inner); !reflect.DeepEqual(got, want) {
		t.Errorf("inner: got %v, want %v", got, want)
	}
	outer, err := left.Merge(right, MergeDirectionHorizontal, MergeModeOuter, "k")
	if err != nil {
		t.Fatal(err)
	}
	if rows, _ := outer.Size(); rows != 6 {
		t.Errorf("outer: %d rows, want the 4 matches, b and c", rows)
	}
	for _, row := range rowsOf(outer) {
		if row[0] == "b" && row[2] != nil || row[0] == "c" && row[1] != nil {
			t.Errorf("outer: an unmatched side is not nil in %v", row)
		}
	}
}

func TestContractMergeOfEmptyTablesKeepsTheSchema(t *testing.T) {
	empty := NewDataTable(NewDataList().SetName("p"), NewDataList().SetName("q"))
	out, err := empty.Merge(empty, MergeDirectionHorizontal, MergeModeInner, "p")
	if err != nil {
		t.Fatal(err)
	}
	if rows, cols := out.Size(); rows != 0 || cols != 3 || !slices.Equal(out.ColNames(), []string{"p", "q", "q_other"}) {
		t.Errorf("got %d x %d with columns %q", rows, cols, out.ColNames())
	}
}

func TestContractAppendColsPadsShorterColumnsWithNil(t *testing.T) {
	dt := NewDataTable(NewDataList().SetName("x"), NewDataList(1, nil).SetName("y"))
	if rows, cols := dt.Size(); rows != 2 || cols != 2 {
		t.Fatalf("got %d x %d, want 2 x 2", rows, cols)
	}
	if got, want := rowsOf(dt), [][]any{{nil, 1}, {nil, nil}}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// A table's row count is its longest column's, so removing its last column
// leaves no rows, and its row names go with them.
func TestContractATableWithNoColumnsHasNoRows(t *testing.T) {
	dt := NewDataTable(NewDataList(1, 2, 3).SetName("x"))
	dt.SetRowNames([]string{"r1", "r2", "r3"})
	dt.DropColsByNumber(0)
	if rows, cols := dt.Size(); rows != 0 || cols != 0 || len(dt.RowNames()) != 0 || dt.Err() != nil {
		t.Errorf("got %d x %d, row names %q, error %v", rows, cols, dt.RowNames(), dt.Err())
	}
}

// Merge groups matched rows by key in the order the key first appears in the
// left table, then the unmatched rows; Docs/DataTable.md states it.
func TestContractMergeRowOrder(t *testing.T) {
	left := NewDataTable(NewDataList("a", "b", "a", "d").SetName("k"), NewDataList(1, 2, 3, 4).SetName("x"))
	right := NewDataTable(NewDataList("b", "a", "c").SetName("k"), NewDataList(10, 20, 30).SetName("y"))
	cases := []struct {
		mode MergeMode
		want [][]any
	}{
		{MergeModeInner, [][]any{{"a", 1, 20}, {"a", 3, 20}, {"b", 2, 10}}},
		{MergeModeLeft, [][]any{{"a", 1, 20}, {"a", 3, 20}, {"b", 2, 10}, {"d", 4, nil}}},
		{MergeModeRight, [][]any{{"b", 2, 10}, {"a", 1, 20}, {"a", 3, 20}, {"c", nil, 30}}},
		{MergeModeOuter, [][]any{{"a", 1, 20}, {"a", 3, 20}, {"b", 2, 10}, {"d", 4, nil}, {"c", nil, 30}}},
	}
	for _, tt := range cases {
		out, err := left.Merge(right, MergeDirectionHorizontal, tt.mode, "k")
		if err != nil {
			t.Fatal(err)
		}
		if got := rowsOf(out); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%v: got %v, want %v", tt.mode, got, tt.want)
		}
	}
}
