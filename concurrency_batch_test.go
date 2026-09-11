package insyra

import (
	"slices"
	"testing"
)

// K-8 (#209): a nil instance used to be dereferenced inside AtomicDoAll and
// panic. There is nothing to lock on a nil value; the callback still runs.
func TestAtomicDoAllSkipsNilInstance(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)

	var nilList *DataList
	var nilTable *DataTable
	ran := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("AtomicDoAll panicked on a nil instance: %v", r)
			}
		}()
		AtomicDoAll(func() { ran = true }, NewDataList(1), nilList, nilTable)
	}()
	if !ran {
		t.Fatal("the callback did not run")
	}
}

// E-2 (#234): a script that fails on its third statement used to leave the
// first two applied, so the table was half-changed with nothing in the
// documentation to say so.
func TestExecuteCCLIsAllOrNothing(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)

	dt := NewDataTable(
		NewDataList(10, 20, 30).SetName("price"),
		NewDataList(1, 0, 3).SetName("qty"),
	)
	dt.ExecuteCCL("NEW('x') = A + 1\nA = A * 2\nNEW('y') = A / B\nNEW('z') = A")
	if dt.PopErr() == nil {
		t.Fatal("the script should fail on its third statement")
	}
	if n := dt.NumCols(); n != 2 {
		t.Errorf("a failed script left %d columns %v, want the original 2", n, dt.ColNames())
	}
	if got := dt.GetColByName("price").Data(); !slices.Equal(got, []any{10, 20, 30}) {
		t.Errorf("a failed script changed price to %v", got)
	}
}

// Later statements must see what earlier ones wrote, on the working copy just
// as they did when the statements ran against the table directly.
func TestExecuteCCLLaterStatementsSeeEarlierOnes(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)

	dt := NewDataTable(NewDataList(10, 20, 30).SetName("price"))
	dt.ExecuteCCL("A = A * 2\nNEW('d') = A + 1")
	if err := dt.PopErr(); err != nil {
		t.Fatal(err)
	}
	if got := dt.GetColByName("price").Data(); !slices.Equal(got, []any{20.0, 40.0, 60.0}) {
		t.Errorf("price = %v, want the doubled values committed", got)
	}
	if got := dt.GetColByName("d").Data(); !slices.Equal(got, []any{21.0, 41.0, 61.0}) {
		t.Errorf("d = %v, want it computed from the updated A", got)
	}
}
