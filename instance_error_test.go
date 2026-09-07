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

// D-5: searching for a name or a value and finding nothing is a normal
// result, not an error. Otherwise a sticky Err() would be permanently set by
// ordinary lookups. An out-of-range *index*, by contrast, is a caller mistake
// and stays an error.
func TestLookupsDoNotRecordErrors(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)
	Config.SetPanicOnError(false)

	dl := NewDataList(1, 2, 3)
	dl.FindFirst(42)
	dl.FindLast(42)
	if err := dl.Err(); err != nil {
		t.Fatalf("a lookup that found nothing set Err(): %v", err)
	}

	indexed := NewDataList(1, 2, 3)
	indexed.Get(99)
	if indexed.Err() == nil {
		t.Fatal("an out-of-range index should be recorded as an error")
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
	dt.GetColByName("nope")
	dt.GetRowByName("nope")
	dt.GetCol("ZZ")
	dt.GetColIndexByName("nope")
	if err := dt.Err(); err != nil {
		t.Fatalf("a DataTable lookup that found nothing set Err(): %v", err)
	}

	indexedTable := NewDataTable(NewDataList(1, 2).SetName("a"))
	indexedTable.GetElementByNumberIndex(0, 99)
	if indexedTable.Err() == nil {
		t.Fatal("an out-of-range index on a DataTable should be recorded as an error")
	}

	// A genuinely invalid call is still an error.
	dt.SetColToRowNames("ZZ")
	if err := dt.Err(); err == nil {
		t.Fatal("an invalid mutation recorded nothing")
	} else if !strings.Contains(err.Message, "ZZ") {
		t.Fatalf("error message should name the column: %q", err.Message)
	}
}
