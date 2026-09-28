package insyra

import "testing"

// assertEmptyWithError checks the shape a failed per-column transform owes its
// caller: a non-nil, empty, still-usable DataList whose Err() is set. The error
// the caller can reach on the result matters as much as the one on the table,
// because the table's is sticky and may already hold an earlier failure.
func assertEmptyWithError(t *testing.T, name string, got *DataList) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s: returned nil", name)
	}
	if n := got.Len(); n != 0 {
		t.Errorf("%s: expected length 0, got %d", name, n)
	}
	if err := got.Err(); err == nil {
		t.Errorf("%s: expected the returned list to carry an error", name)
	}
}

// newThreeRowTable builds the smallest table these tests need. Each case needs
// its own table because Err() is sticky and a recorded error never clears.
func newThreeRowTable() *DataTable {
	return NewDataTable(NewDataList(1.0, 2.0, 3.0).SetName("a"))
}

// A missing column is a failure the caller asked for, so every scalar column
// transform hands back an empty list carrying the table's error rather than a
// bare empty list that looks like a valid (if empty) result.
func TestTableColumnTransformsCarryMissingColumnError(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)
	Config.SetPanicOnError(false)

	cases := []struct {
		name string
		call func(dt *DataTable) *DataList
	}{
		{"ShiftCol", func(dt *DataTable) *DataList { return dt.ShiftCol(Name("missing"), 1) }},
		{"DiffCol", func(dt *DataTable) *DataList { return dt.DiffCol(Name("missing"), 1) }},
		{"PctChangeCol", func(dt *DataTable) *DataList { return dt.PctChangeCol(Name("missing"), 1) }},
		{"CumSumCol", func(dt *DataTable) *DataList { return dt.CumSumCol(Name("missing")) }},
		{"CumProdCol", func(dt *DataTable) *DataList { return dt.CumProdCol(Name("missing")) }},
		{"CumMaxCol", func(dt *DataTable) *DataList { return dt.CumMaxCol(Name("missing")) }},
		{"CumMinCol", func(dt *DataTable) *DataList { return dt.CumMinCol(Name("missing")) }},
	}

	for _, tc := range cases {
		dt := newThreeRowTable()
		assertEmptyWithError(t, tc.name, tc.call(dt))
		if err := dt.Err(); err == nil {
			t.Errorf("%s: expected an error recorded on dt", tc.name)
		}
	}
}

// Rolling computes on a throwaway snapshot of the column, so an invalid option
// only reaches the caller's own view of the table if RollingCol copies it over.
func TestRollingColInvalidOptionIsRecordedOnTheTable(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)
	Config.SetPanicOnError(false)

	dt := newThreeRowTable()
	got := dt.RollingCol(Name("a"), RollingOptions{Window: 3, MinObs: 5}).Mean()

	assertEmptyWithError(t, "RollingCol", got)
	err := dt.Err()
	if err == nil {
		t.Fatal("expected an error recorded on dt")
	}
	if err.FuncName != "RollingCol" {
		t.Errorf("FuncName = %q, want %q", err.FuncName, "RollingCol")
	}
}

// Same for EWM, whose reducers also run on a snapshot.
func TestEWMColInvalidOptionIsRecordedOnTheTable(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)
	Config.SetPanicOnError(false)

	dt := newThreeRowTable()
	got := dt.EWMCol(Name("a"), EWMOptions{}).Mean()

	assertEmptyWithError(t, "EWMCol", got)
	err := dt.Err()
	if err == nil {
		t.Fatal("expected an error recorded on dt")
	}
	if err.FuncName != "EWMCol" {
		t.Errorf("FuncName = %q, want %q", err.FuncName, "EWMCol")
	}
}

// Copying the snapshot's failure onto the table must not touch a success path.
func TestTableWindowSuccessLeavesNoError(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)
	Config.SetPanicOnError(false)

	dt := newThreeRowTable()
	got := dt.RollingCol(Name("a"), RollingOptions{Window: 2}).Mean()

	if n := got.Len(); n != 3 {
		t.Errorf("Len() = %d, want 3", n)
	}
	if err := got.Err(); err != nil {
		t.Errorf("expected no error on the result, got %v", err.Message)
	}
	if err := dt.Err(); err != nil {
		t.Errorf("expected no error on dt, got %v", err.Message)
	}
}
