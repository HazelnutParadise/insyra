package insyra

import "testing"

// PopErr reads and clears in one step so a chain ends with one statement.
func TestPopErrReadsAndClears(t *testing.T) {
	quietLogs(t)

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

// SetErr records the way the internal warn path does: Warning level, the
// error replaces any earlier one, and the record also reaches the global
// buffer.
func TestSetErrRecordsLikeInternalWarnings(t *testing.T) {
	quietLogs(t)
	ClearErrors()
	t.Cleanup(ClearErrors)

	dl := NewDataList(1, 2)
	if got := dl.SetErr("isr", "DT.From", "bad input %d", 7); got != dl {
		t.Fatal("DataList.SetErr must return its receiver")
	}
	err := dl.Err()
	if err == nil || err.Level != LogLevelWarning || err.PackageName != "isr" || err.FuncName != "DT.From" || err.Message != "bad input 7" {
		t.Fatalf("DataList.SetErr recorded %+v", err)
	}
	dl.SetErr("isr", "DT.Push", "second")
	if got := dl.Err(); got == nil || got.FuncName != "DT.Push" {
		t.Fatalf("a later SetErr must replace the earlier error, got %+v", got)
	}

	dt := NewDataTable()
	if got := dt.SetErr("isr", "DT.Col", "no column %q", "x"); got != dt {
		t.Fatal("DataTable.SetErr must return its receiver")
	}
	if got := dt.Err(); got == nil || got.Level != LogLevelWarning || got.Message != `no column "x"` {
		t.Fatalf("DataTable.SetErr recorded %+v", got)
	}

	if n := len(GetErrorsByPackage("isr")); n != 3 {
		t.Fatalf("global buffer holds %d isr records, want 3", n)
	}
}
