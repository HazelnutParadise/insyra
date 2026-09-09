package insyra

import (
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

// CCL-23 / CCL-39: an internal range type used to be written into every cell.
func TestCCLInternalTypesNeverReachACell(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)

	for _, expr := range []string{"A:B", "LAG(@, 1)"} {
		dt := b8Table()
		dt.AddColUsingCCL("r", expr)
		if err := dt.PopErr(); err == nil {
			t.Errorf("%s was accepted; column = %v", expr, dt.GetColByName("r"))
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

// The rejection message should point at the operator, not at a Go type name.
func TestCCLRangeErrorNamesTheOperator(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)

	dt := b8Table()
	dt.AddColUsingCCL("r", "A:B")
	err := dt.PopErr()
	if err == nil {
		t.Fatal("A:B was accepted")
	}
	if !strings.Contains(err.Error(), "column range") {
		t.Errorf("error should explain that a range is not a value: %v", err)
	}
}
