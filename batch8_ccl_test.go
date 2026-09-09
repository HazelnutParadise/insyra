package insyra

import (
	"reflect"
	"strings"
	"testing"
)

// b8Table is a table whose B column contains a zero, so A / B is the natural
// way to ask whether an operand was evaluated.
func b8Table() *DataTable {
	return NewDataTable(
		NewDataList(10, 20, 30).SetName("price"),
		NewDataList(1, 0, 3).SetName("qty"),
	)
}

// CCL-12, seen from the public API: the guard must run before what it guards.
func TestCCLShortCircuitThroughAddCol(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)

	for _, tc := range []struct {
		expr string
		want []any
	}{
		{"B != 0 && A / B > 1", []any{true, false, true}},
		{"CASE(B != 0, A / B, nil)", []any{10.0, nil, 10.0}},
	} {
		dt := b8Table()
		dt.AddColUsingCCL("r", tc.expr)
		if err := dt.PopErr(); err != nil {
			t.Errorf("%s: %v", tc.expr, err)
			continue
		}
		got := dt.GetColByName("r").Data()
		for i := range tc.want {
			if got[i] != tc.want[i] {
				t.Errorf("%s = %v, want %v", tc.expr, got, tc.want)
				break
			}
		}
	}
}

// CCL-23 / CCL-39: a range and '@' stand for a row, so a column built from one
// holds that row's values. `A:B` used to fill every cell with an internal
// ccl.ColumnRange struct, and `LAG(@, 1)` with the whole flattened table.
func TestCCLRowShapedValues(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)

	dt := b8Table()
	dt.AddColUsingCCL("r", "A:B")
	if err := dt.PopErr(); err != nil {
		t.Fatalf("A:B: %v", err)
	}
	got := dt.GetColByName("r").Data()
	want := [][]any{{10, 1}, {20, 0}, {30, 3}}
	for i := range want {
		if !reflect.DeepEqual(got[i], []any(want[i])) {
			t.Fatalf("A:B = %v, want %v", got, want)
		}
	}

	dt = b8Table()
	dt.AddColUsingCCL("prev", "LAG(@, 1)")
	if err := dt.PopErr(); err != nil {
		t.Fatalf("LAG(@, 1): %v", err)
	}
	prev := dt.GetColByName("prev").Data()
	if prev[0] != nil {
		t.Errorf("LAG(@, 1): first row = %v, want nil", prev[0])
	}
	if !reflect.DeepEqual(prev[1], []any{10, 1}) {
		t.Errorf("LAG(@, 1): second row = %v, want [10 1]", prev[1])
	}
}

// The readings that consume a range keep the values they always had. This is a
// regression guard: making a bare `A:B` mean the current row briefly made
// `SUM(A.(0:1))` count the same slice once per row.
func TestCCLRangeConsumersUnchanged(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)

	dt := NewDataTable(
		NewDataList(1, 2, 3, 4, 5, 6).SetName("a"),
		NewDataList(10, 20, 30, 40, 50, 60).SetName("b"),
		NewDataList(100, 200, 300, 400, 500, 600).SetName("c"),
	)
	for _, tc := range []struct {
		expr string
		want float64
	}{
		{"SUM(A:C)", 2331},
		{"SUM((A:C).(2:5))", 1998},
		{"SUM(A.(0:1))", 3},
		{"COUNT((A:C).(2:5))", 12},
	} {
		table := dt.Clone()
		table.AddColUsingCCL("r", tc.expr)
		if err := table.PopErr(); err != nil {
			t.Errorf("%s: %v", tc.expr, err)
			continue
		}
		for _, v := range table.GetColByName("r").Data() {
			if v != tc.want {
				t.Errorf("%s = %v, want %v in every row", tc.expr, v, tc.want)
				break
			}
		}
	}
}

// A row range names rows but not of what, and a sequence function that does
// arithmetic cannot take a row.
func TestCCLRowShapedValuesThatStayErrors(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)

	for _, expr := range []string{"1:2", "CUMSUM(@)", "ROLLING_SUM(A:B, 2)"} {
		dt := b8Table()
		dt.AddColUsingCCL("r", expr)
		if err := dt.PopErr(); err == nil {
			t.Errorf("%s was accepted; column = %v", expr, colData(dt, "r"))
		}
	}
}

// CCL-11: Go's "<nil>" placeholder must not become data.
func TestCCLNilConcatThroughAddCol(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)

	dt := NewDataTable(NewDataList("a", nil, "c").SetName("s"))
	dt.AddColUsingCCL("r", "A & '!'")
	if err := dt.PopErr(); err != nil {
		t.Fatal(err)
	}
	got := dt.GetColByName("r").Data()
	want := []any{"a!", "!", "c!"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

// CCL-13 / CCL-25: a typo must not compile into a different question.
func TestCCLMalformedExpressionsAreRejected(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)

	for _, expr := range []string{
		"SUM(A B)",
		"IF(A > 15, 1, 0,)",
		"A.(1.7)",
		"ROLLING_MEAN(A, 2.9)",
		"AND()",
		"AND('abc', true)",
		"'hello' > 5",
	} {
		dt := b8Table()
		dt.AddColUsingCCL("r", expr)
		if err := dt.PopErr(); err == nil {
			t.Errorf("%s was accepted; column = %v", expr, colData(dt, "r"))
		}
	}
}

// colData returns a column's values, or nil when the column is absent, so a
// failure message never panics on its way out.
func colData(dt *DataTable, name string) []any {
	if col := dt.GetColByName(name); col != nil {
		return col.Data()
	}
	return nil
}

// CCL-14: & now binds looser than +, so a sum can be concatenated.
func TestCCLConcatPrecedenceThroughAddCol(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)

	dt := b8Table()
	dt.AddColUsingCCL("r", "'total: ' & A + B")
	if err := dt.PopErr(); err != nil {
		t.Fatal(err)
	}
	got := dt.GetColByName("r").Data()
	want := []any{"total: 11", "total: 20", "total: 33"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

// CCL-9: a filter on text columns returned nothing, because both directions
// of a string comparison were false.
func TestCCLStringOrderingThroughFilter(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)

	dt := NewDataTable(NewDataList("banana", "apple", "cherry").SetName("fruit"))
	dt.AddColUsingCCL("before_c", "A < 'c'")
	if err := dt.PopErr(); err != nil {
		t.Fatal(err)
	}
	got := dt.GetColByName("before_c").Data()
	want := []any{true, true, false}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

// The rejection message should say what to do, not name a Go type.
func TestCCLRowRangeErrorSaysWhatToDo(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)

	dt := b8Table()
	dt.AddColUsingCCL("r", "1:2")
	err := dt.PopErr()
	if err == nil {
		t.Fatal("a bare row range was accepted")
	}
	if !strings.Contains(err.Error(), "attach it to a column") {
		t.Errorf("error should say how to write it instead: %v", err)
	}
}
