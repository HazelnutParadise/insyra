package isr_test

import (
	"fmt"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/isr"
)

func quietLogs(t *testing.T) {
	t.Helper()
	level := insyra.Config.GetLogLevel()
	insyra.Config.SetLogLevel(insyra.LogLevelFatal)
	t.Cleanup(func() { insyra.Config.SetLogLevel(level) })
}

// eachDontPanic runs f with Config.SetDontPanic off and then on. On v0.3.2 a
// failure in isr ended the program unless SetDontPanic(true) was set, in which
// case LogFatal only logged and the call returned a value. Both configurations
// must now return that value, and neither may end the program.
func eachDontPanic(t *testing.T, f func(t *testing.T)) {
	t.Helper()
	for _, dontPanic := range []bool{false, true} {
		t.Run(fmt.Sprintf("dontPanic=%v", dontPanic), func(t *testing.T) {
			previous := insyra.Config.GetDontPanicStatus()
			insyra.Config.SetDontPanic(dontPanic)
			t.Cleanup(func() { insyra.Config.SetDontPanic(previous) })
			f(t)
		})
	}
}

// A source DT.From cannot read, or an input it does not support, gives a
// wrapper around a nil DataTable, as on v0.3.2 with SetDontPanic(true).
func TestDTFromUnreadableSourceWrapsANilTable(t *testing.T) {
	quietLogs(t)

	inputs := map[string]any{
		"missing CSV file":   isr.CSV{FilePath: "no_such_file_for_insyra_test.csv"},
		"missing JSON file":  isr.JSON{FilePath: "no_such_file_for_insyra_test.json"},
		"malformed JSON":     isr.JSON{Bytes: []byte("{")},
		"Excel with no path": isr.Excel{},
		"missing Excel file": isr.Excel{FilePath: "no_such_file_for_insyra_test.xlsx"},
		"empty 2D slice":     [][]int{},
		"unsupported type":   struct{ X int }{1},
	}
	eachDontPanic(t, func(t *testing.T) {
		for name, in := range inputs {
			got := isr.DT.From(in)
			if got == nil {
				t.Errorf("%s: DT.From returned nil", name)
				continue
			}
			if got.DataTable != nil {
				t.Errorf("%s: DT.From wrapped a table; v0.3.2 wrapped nil", name)
			}
		}
	})
}

// A Row or Col that cannot be added (its keys mix ints and strings) is skipped
// and the rest still go in. The table is not nil, so the error is recorded.
func TestDTFromSkipsARowItCannotAdd(t *testing.T) {
	quietLogs(t)

	bad := isr.Row{"A": 1, 0: 2}
	tests := []struct {
		name       string
		in         any
		rows, cols int
	}{
		{name: "Row", in: bad, rows: 0, cols: 0},
		{name: "Rows", in: isr.Rows{{"A": 1}, bad, {"A": 3}}, rows: 2, cols: 1},
		{name: "Col", in: isr.Col(bad), rows: 0, cols: 0},
		{name: "Cols", in: isr.Cols{{"A": 1}, isr.Col(bad), {"A": 3}}, rows: 1, cols: 2},
	}
	eachDontPanic(t, func(t *testing.T) {
		for _, tt := range tests {
			got := isr.DT.From(tt.in)
			if got == nil || got.DataTable == nil {
				t.Fatalf("%s: DT.From gave no table", tt.name)
			}
			if rows, cols := got.Size(); rows != tt.rows || cols != tt.cols {
				t.Errorf("%s: size %dx%d, want %dx%d", tt.name, rows, cols, tt.rows, tt.cols)
			}
			if got.Err() == nil {
				t.Errorf("%s: the skipped row was not recorded", tt.name)
			}
		}
	})
}

// Col and Row with a selector of an unsupported type give a wrapper around a
// nil DataList, the same as a column or row that does not exist.
func TestColRowUnsupportedSelectorWrapsANilList(t *testing.T) {
	quietLogs(t)

	eachDontPanic(t, func(t *testing.T) {
		table := isr.DT.From(isr.Row{"A": 1, "B": 2})
		if l := table.Col(3.5); l == nil || l.DataList != nil {
			t.Errorf("Col(3.5) = %v; v0.3.2 returned a wrapper around nil", l)
		}
		if l := table.Row(3.5); l == nil || l.DataList != nil {
			t.Errorf("Row(3.5) = %v; v0.3.2 returned a wrapper around nil", l)
		}
	})
}

// Push returns the same table whatever it is given. A row or column it cannot
// add is skipped, the rest go in, and the error is recorded on the table.
func TestPushFailureReturnsTheSameTable(t *testing.T) {
	quietLogs(t)

	bad := isr.Row{"A": 1, 0: 2}
	tests := []struct {
		name       string
		in         any
		rows, cols int
	}{
		{name: "Row", in: bad, rows: 1, cols: 2},
		{name: "[]Row", in: []isr.Row{{"A": 5}, bad, {"A": 6}}, rows: 3, cols: 2},
		{name: "Col", in: isr.Col(bad), rows: 1, cols: 2},
		{name: "[]Col", in: []isr.Col{{"A": 5}, isr.Col(bad), {"A": 6}}, rows: 1, cols: 4},
		{name: "unsupported type", in: 3.14, rows: 1, cols: 2},
	}
	eachDontPanic(t, func(t *testing.T) {
		for _, tt := range tests {
			table := isr.DT.From(isr.Row{"A": 1, "B": 2})
			got := table.Push(tt.in)
			if got != table {
				t.Errorf("%s: Push returned a different object", tt.name)
				continue
			}
			if rows, cols := got.Size(); rows != tt.rows || cols != tt.cols {
				t.Errorf("%s: size %dx%d, want %dx%d", tt.name, rows, cols, tt.rows, tt.cols)
			}
			if got.PopErr() == nil || got.Err() != nil {
				t.Errorf("%s: the failure was not recorded, or PopErr did not clear it", tt.name)
			}
		}

		// With no table to push into, Push hands back the same wrapper around
		// nil and records nothing.
		empty := isr.DT.From(nil)
		if got := empty.Push(3.5); got != empty || got.DataTable != nil {
			t.Error("Push on a wrapper around nil did not return that wrapper unchanged")
		}
	})
}
