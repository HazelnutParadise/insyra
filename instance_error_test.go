package insyra

import (
	"strings"
	"testing"
)

// D-3 / decision: the first error survives the rest of the chain, so a long
// chain reports the root cause instead of the last symptom.
func TestErrIsSticky(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)
	Config.SetPanicOnError(false)

	dl := NewDataList(1.0, 2.0, 3.0)
	dl.MovingAverage(0)                       // first failure: invalid window
	dl.WeightedMovingAverage(2, []float64{1}) // second failure: weights length
	err := dl.Err()
	if err == nil {
		t.Fatal("no error recorded")
	}
	if err.FuncName != "MovingAverage" {
		t.Fatalf("Err() reports %q; the first failure (MovingAverage) must win", err.FuncName)
	}

	dt := NewDataTable(NewDataList(1, 2, 3).SetName("a"))
	dt.SetColToRowNames("ZZ")
	dt.SetRowToColNames(99)
	if got := dt.Err(); got == nil || got.FuncName != "SetColToRowNames" {
		t.Fatalf("DataTable Err() = %v; the first failure must win", got)
	}
}

// PopErr reads and clears in one step so a chain ends with one statement.
func TestPopErrReadsAndClears(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)
	Config.SetPanicOnError(false)

	dl := NewDataList(1.0, "x")
	dl.Normalize()
	first := dl.PopErr()
	if first == nil {
		t.Fatal("PopErr returned nil for a failed call")
	}
	if again := dl.PopErr(); again != nil {
		t.Fatalf("PopErr did not clear the error: %v", again)
	}
	if dl.Err() != nil {
		t.Fatal("Err() still set after PopErr")
	}

	dt := NewDataTable(NewDataList(1, 2))
	dt.SetRowToColNames(99)
	if dt.PopErr() == nil {
		t.Fatal("DataTable.PopErr returned nil for a failed call")
	}
	if dt.PopErr() != nil {
		t.Fatal("DataTable.PopErr did not clear the error")
	}
}

// A clone starts clean, so a failed source does not poison derived objects.
func TestCloneStartsWithoutError(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)
	Config.SetPanicOnError(false)

	dl := NewDataList(1.0, "x")
	dl.Normalize()
	if dl.Err() == nil {
		t.Fatal("setup: expected an error on the source list")
	}
	if c := dl.Clone(); c.Err() != nil {
		t.Fatalf("Clone carried the source error: %v", c.Err())
	}

	dt := NewDataTable(NewDataList(1, 2))
	dt.SetRowToColNames(99)
	if dt.Err() == nil {
		t.Fatal("setup: expected an error on the source table")
	}
	if c := dt.Clone(); c.Err() != nil {
		t.Fatalf("DataTable.Clone carried the source error: %v", c.Err())
	}
}

// D-5: asking whether a value is present and getting "no" is an answer, not a
// failure, so it must not fill in the sticky Err(). Addressing a column, row
// or index that does not exist is a different thing: the caller asserted it
// was there, the only other signal is a nil, and that stays an error.
func TestValueSearchesDoNotRecordErrors(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)
	Config.SetPanicOnError(false)

	dl := NewDataList(1, 2, 3)
	dl.FindFirst(42)
	dl.FindLast(42)
	if err := dl.Err(); err != nil {
		t.Fatalf("a lookup that found nothing set Err(): %v", err)
	}

	if empty := NewDataList().Pop(); empty != nil {
		t.Fatalf("Pop on an empty list should return nil, got %v", empty)
	}
	if err := dl.Err(); err != nil {
		t.Fatalf("Pop on an empty list set Err(): %v", err)
	}

	empty := NewDataList()
	empty.Count(1)
	empty.FindAll(1)
	empty.Mean()
	empty.Max()
	if err := empty.Err(); err != nil {
		t.Fatalf("an operation over an empty list set Err(): %v", err)
	}

	dt := NewDataTable(NewDataList(1, 2).SetName("a"))
	if idx, ok := dt.GetRowIndexByName("nope"); ok || idx != -1 {
		t.Fatalf("GetRowIndexByName = %d, %v; want -1, false", idx, ok)
	}
	if err := dt.Err(); err != nil {
		t.Fatalf("a comma-ok lookup set Err(): %v", err)
	}
}

// The other half of the same rule: addressing something that is not there is
// a caller mistake. These all hand back a bare nil or -1, so without Err()
// the caller would have no signal at all.
func TestStructuralMissesRecordErrors(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)
	Config.SetPanicOnError(false)

	cases := map[string]func(){}
	cases["Get out of range"] = func() { NewDataList(1, 2, 3).Get(99) }
	cases["GetElementByNumberIndex out of range"] = func() {
		NewDataTable(NewDataList(1, 2)).GetElementByNumberIndex(0, 99)
	}
	cases["GetColByName missing"] = func() { NewDataTable(NewDataList(1, 2).SetName("a")).GetColByName("nope") }
	cases["GetCol missing"] = func() { NewDataTable(NewDataList(1, 2).SetName("a")).GetCol("ZZ") }
	cases["GetRowByName missing"] = func() { NewDataTable(NewDataList(1, 2)).GetRowByName("nope") }
	cases["GetColByNumber out of range"] = func() { NewDataTable(NewDataList(1, 2)).GetColByNumber(9) }
	cases["GetRow out of range"] = func() { NewDataTable(NewDataList(1, 2)).GetRow(99) }
	cases["GetColIndexByName missing"] = func() { NewDataTable(NewDataList(1, 2)).GetColIndexByName("nope") }
	cases["GetColNumberByName missing"] = func() { NewDataTable(NewDataList(1, 2)).GetColNumberByName("nope") }

	for name, call := range cases {
		t.Run(name, func(t *testing.T) {
			ClearErrors()
			call()
			errs := GetAllErrors()
			if len(errs) == 0 || errs[len(errs)-1].Level != LogLevelError {
				t.Fatalf("%s did not record an error: %v", name, errs)
			}
		})
	}

	// An invalid mutation is an error too, and the message names the target.
	dt := NewDataTable(NewDataList(1, 2).SetName("a"))
	dt.SetColToRowNames("ZZ")
	if err := dt.Err(); err == nil {
		t.Fatal("an invalid mutation recorded nothing")
	} else if !strings.Contains(err.Message, "ZZ") {
		t.Fatalf("error message should name the column: %q", err.Message)
	}
}
