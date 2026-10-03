package commands

import (
	"strings"
	"testing"

	insyra "github.com/HazelnutParadise/insyra"
)

func tableWithCols(names ...string) *insyra.DataTable {
	dt := insyra.NewDataTable()
	for i, name := range names {
		dt.AppendCols(insyra.NewDataList(i, i+10).SetName(name))
	}
	return dt
}

// duplicateNames builds a table whose two columns share a name. Appending
// renames a duplicate, but replacing a column keeps the name it brings.
func duplicateNames() *insyra.DataTable {
	dt := tableWithCols("a", "b")
	dt.UpdateCol(1, insyra.NewDataList(1, 2).SetName("a"))
	return dt
}

// A bare token is read as a number, as letters and as a name; the command
// uses the column when every reading that lands agrees, and refuses when two
// disagree. Owner's ruling on #315, 2026-09-27.
func TestColumnTokenReadings(t *testing.T) {
	xab := tableWithCols("x", "a", "b")
	years := tableWithCols("key", "1", "2021")
	zero := tableWithCols("id", "0")

	cases := []struct {
		table *insyra.DataTable
		token string
		want  int
	}{
		{xab, "C", 2},         // letters only
		{xab, "c", 2},         // letters read without regard to case, as in the library
		{xab, "x", 0},         // X is past the last column, so the name
		{xab, "0", 0},         // a 0-based number
		{xab, "-1", 2},        // negative counts from the end
		{xab, "index:a", 0},   // forced letters
		{xab, "name:a", 1},    // forced name
		{xab, "number:-1", 2}, // forced number
		{years, "1", 1},       // number 1 is the column named "1": both readings agree
		{years, "2", 2},       // a number with no column of that name
		{years, "2021", 2},    // past the last column as a number, so the name
		{years, "name:2021", 2},
	}
	for _, tc := range cases {
		got, err := resolveColumnToken("cmd", tc.table, tc.token)
		if err != nil || got != tc.want {
			t.Errorf("%q on %v: got (%d, %v), want %d", tc.token, tc.table.ColNames(), got, err, tc.want)
		}
	}

	refused := []struct {
		table *insyra.DataTable
		token string
		says  []string
	}{
		{xab, "a", []string{`"a"`, "index:a", "name:a"}},
		{xab, "b", []string{"index:b", "name:b"}},
		{zero, "0", []string{"number:0", "name:0"}},
		{xab, "price", []string{`"price" not found`, "x, a, b"}},
		{xab, "5", []string{"out of range"}},
		{xab, "number:3", []string{"out of range"}},
		{xab, "index:Z", []string{"out of range"}},
		{xab, "name:A", []string{`"A" not found`}},
		{xab, "number:x", []string{"number:"}},
		{duplicateNames(), "name:a", []string{"2 columns are named", "number:"}},
	}
	for _, tc := range refused {
		_, err := resolveColumnToken("cmd", tc.table, tc.token)
		if err == nil {
			t.Errorf("%q on %v: accepted, want an error", tc.token, tc.table.ColNames())
			continue
		}
		for _, s := range tc.says {
			if !strings.Contains(err.Error(), s) {
				t.Errorf("%q on %v: error %q does not say %q", tc.token, tc.table.ColNames(), err, s)
			}
		}
	}
}

func TestRowTokenReadings(t *testing.T) {
	dt := insyra.NewDataTable(insyra.NewDataList(1, 2, 3, 4))
	dt.SetRowNames([]string{"r1", "2", "x", "3"})

	for token, want := range map[string]int{"0": 0, "r1": 0, "x": 2, "-1": 3, "name:2": 1, "number:2": 2, "3": 3} {
		got, err := resolveRowToken("cmd", dt, token)
		if err != nil || got != want {
			t.Errorf("row %q: got (%d, %v), want %d", token, got, err, want)
		}
	}
	for token, says := range map[string]string{"2": "name:2", "nope": "not found", "index:A": "index:", "9": "out of range"} {
		if _, err := resolveRowToken("cmd", dt, token); err == nil || !strings.Contains(err.Error(), says) {
			t.Errorf("row %q: got %v, want an error saying %q", token, err, says)
		}
	}
}

// Through Dispatch: the eight commands #315 names, and a column list, take
// all three spellings and refuse the ambiguous one.
func TestCommandsPickColumnsOneWay(t *testing.T) {
	ctx := newTestExecContext(t)
	fresh := func() *insyra.DataTable {
		dt := insyra.NewDataTable(
			insyra.NewDataList("apple", "pear").SetName("name"),
			insyra.NewDataList(1.5, 2.5).SetName("price"),
			insyra.NewDataList(3.0, 4.0).SetName("qty"),
		)
		dt.SetRowNames([]string{"r0", "r1"})
		return dt
	}
	run := func(line ...string) error {
		ctx.Vars["t"] = fresh()
		return Dispatch(ctx, line[0], line[1:])
	}

	for _, line := range [][]string{
		{"col", "t", "price", "as", "c"}, {"col", "t", "B", "as", "c"}, {"col", "t", "1", "as", "c"},
		{"row", "t", "r1", "as", "r"}, {"row", "t", "1", "as", "r"},
		{"sort", "t", "price"}, {"sort", "t", "B", "desc"}, {"sort", "t", "number:1"},
		{"swap", "t", "col", "price", "C"}, {"swap", "t", "row", "r0", "1"},
		{"dropcol", "t", "price", "C"}, {"droprow", "t", "r0"},
		{"fillna", "t", "mean", "cols", "B,qty", "as", "f"},
		{"groupby", "t", "by", "name", "agg", "B:sum", "as", "g"},
	} {
		if err := run(line...); err != nil {
			t.Errorf("%v: %v", line, err)
		}
	}

	ctx.Vars["t"] = fresh()
	if err := Dispatch(ctx, "get", []string{"t", "0", "price"}); err != nil {
		t.Fatalf("get t 0 price: %v", err)
	}
	if got := strings.TrimSpace(outputOf(ctx)); !strings.HasSuffix(got, "1.5") {
		t.Errorf("get t 0 price printed %q, want 1.5", got)
	}
	ctx.Vars["t"] = fresh()
	if err := Dispatch(ctx, "set", []string{"t", "r1", "qty", "9"}); err != nil {
		t.Fatalf("set t r1 qty 9: %v", err)
	}
	if got := ctx.Vars["t"].(*insyra.DataTable).GetElementByNumberIndex(1, 2); got != int64(9) && got != 9 {
		t.Errorf("set wrote %v (%T) to row 1, qty", got, got)
	}

	// A table where the letters and a name disagree.
	for _, line := range [][]string{
		{"col", "t", "a", "as", "c"}, {"sort", "t", "a"}, {"get", "t", "0", "a"},
		{"dropcol", "t", "a"}, {"fillna", "t", "mean", "cols", "a"},
	} {
		ctx.Vars["t"] = tableWithCols("x", "a")
		err := Dispatch(ctx, line[0], line[1:])
		if err == nil || !strings.Contains(err.Error(), "name:a") {
			t.Errorf("%v: got %v, want the ambiguity refused", line, err)
		}
	}
}

// A failed command used to leave its error on the table, and the next command
// that checked for one reported it as its own.
func TestFailedCommandDoesNotPoisonTheNext(t *testing.T) {
	ctx := newTestExecContext(t)
	ctx.Vars["t"] = insyra.NewDataTable(insyra.NewDataList(2.0, 1.0).SetName("price"))
	ctx.Vars["t"].(*insyra.DataTable).GetColByName("nope") // records an error on t
	if err := Dispatch(ctx, "sort", []string{"t", "price"}); err != nil {
		t.Errorf("sort after an unrelated failure: %v", err)
	}
}
