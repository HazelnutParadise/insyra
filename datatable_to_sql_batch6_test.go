package insyra

import (
	"strings"
	"testing"
)

// SEC-4: every other statement quotes its identifiers; the sqlite column
// lookup interpolated the table name straight into PRAGMA, so a perfectly
// ordinary table name with a space could not be appended to.
func TestAppendToTableWithSpaceInName(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)

	db := newTestSQLite(t)

	const table = "my table"
	dt := NewDataTable(NewDataList(1, 2).SetName("value"))
	if err := dt.ToSQL(db, table, ToSQLOptions{IfExists: SQLActionIfTableExistsFail}); err != nil {
		t.Fatalf("creating %q failed: %v", table, err)
	}
	more := NewDataTable(NewDataList(3).SetName("value"))
	if err := more.ToSQL(db, table, ToSQLOptions{IfExists: SQLActionIfTableExistsAppend}); err != nil {
		t.Fatalf("appending to %q failed: %v", table, err)
	}

	read, err := ReadSQL(db, table)
	if err != nil {
		t.Fatal(err)
	}
	if got := read.NumRows(); got != 3 {
		t.Fatalf("table has %d rows, want 3", got)
	}
}

// A name containing a quote must not break out of the identifier either.
func TestAppendToTableWithQuoteInName(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)

	db := newTestSQLite(t)

	const table = `we"ird`
	dt := NewDataTable(NewDataList(1).SetName("value"))
	if err := dt.ToSQL(db, table, ToSQLOptions{IfExists: SQLActionIfTableExistsFail}); err != nil {
		t.Fatalf("creating %q failed: %v", table, err)
	}
	more := NewDataTable(NewDataList(2).SetName("value"))
	if err := more.ToSQL(db, table, ToSQLOptions{IfExists: SQLActionIfTableExistsAppend}); err != nil {
		if strings.Contains(err.Error(), "syntax") {
			t.Fatalf("identifier was not quoted: %v", err)
		}
		t.Fatalf("appending to %q failed: %v", table, err)
	}
}
