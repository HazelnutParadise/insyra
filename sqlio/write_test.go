package sqlio

import (
	"context"
	"database/sql"
	"github.com/HazelnutParadise/insyra"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	_ "modernc.org/sqlite"
)

// newTestSQLite returns a pure-Go SQLite in-memory database for tests,
// closed via t.Cleanup. One connection, because every connection to
// ":memory:" opens a database of its own.
func newTestSQLite(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// execSQL runs a statement the test needs, failing the test if it fails.
func execSQL(t *testing.T, db *sql.DB, query string, args ...any) {
	t.Helper()
	_, err := db.Exec(query, args...)
	require.NoError(t, err)
}

func TestToSQL_TransactionRollbackOnInsertError(t *testing.T) {
	db := newTestSQLite(t)

	// 建立表格，並加上 CHECK constraint 使插入會失敗
	execSQL(t, db, "CREATE TABLE test_tx (id INTEGER PRIMARY KEY AUTOINCREMENT, value INTEGER CHECK(value > 0));")

	// 建立 insyra.DataTable，帶入違反約束的值
	valueCol := insyra.NewDataList(-1).SetName("value")
	dt := insyra.NewDataTable(valueCol)

	// 使用 AppendIfExists，因為表格已存在
	err := Write(context.Background(), dt, db, SQLite, "test_tx", WriteOptions{IfTableExists: TableExistsAppend})
	require.Error(t, err, "expected ToSQL to return an error due to CHECK constraint violation")

	var cnt int64
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM test_tx;").Scan(&cnt))
	require.Equal(t, int64(0), cnt, "expected zero rows after rollback")
}

func TestToSQL_BatchInsertAcrossBatches(t *testing.T) {
	db := newTestSQLite(t)

	const total = 1500
	values := make([]any, total)
	for i := range total {
		values[i] = i + 1
	}
	dt := insyra.NewDataTable(insyra.NewDataList(values...).SetName("n"))

	// BatchSize 500 ⇒ 3 batched INSERTs.
	err := Write(context.Background(), dt, db, SQLite, "nums", WriteOptions{BatchSize: 500})
	require.NoError(t, err)

	var cnt int64
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM nums").Scan(&cnt))
	require.Equal(t, int64(total), cnt)

	// Spot-check a value.
	var got int64
	require.NoError(t, db.QueryRow("SELECT n FROM nums WHERE n = ?", 1234).Scan(&got))
	require.Equal(t, int64(1234), got)
}

func TestToSQL_AppendModeAddsMissingColumnOnce(t *testing.T) {
	db := newTestSQLite(t)

	// Pre-create table with only column "a".
	execSQL(t, db, "CREATE TABLE evolving (a INTEGER);")

	dt := insyra.NewDataTable(
		insyra.NewDataList(1, 2, 3).SetName("a"),
		insyra.NewDataList("x", "y", "z").SetName("b"),
	)
	err := Write(context.Background(), dt, db, SQLite, "evolving", WriteOptions{IfTableExists: TableExistsAppend})
	require.NoError(t, err)

	// Column "b" should have been added once and populated.
	var bs []string
	rows, err := db.Query("SELECT b FROM evolving ORDER BY a")
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var v string
		require.NoError(t, rows.Scan(&v))
		bs = append(bs, v)
	}
	require.NoError(t, rows.Err())
	require.Equal(t, []string{"x", "y", "z"}, bs)
}

func TestToSQL_AllNilColumnFallsBackToText(t *testing.T) {
	db := newTestSQLite(t)

	dt := insyra.NewDataTable(
		insyra.NewDataList(1, 2).SetName("a"),
		insyra.NewDataList(nil, nil).SetName("b"),
	)
	require.NoError(t, Write(context.Background(), dt, db, SQLite, "with_nil"))

	// Verify the column "b" was created with TEXT-ish affinity by reading PRAGMA.
	rows, err := db.Query("PRAGMA table_info(with_nil)")
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()
	types := map[string]string{}
	for rows.Next() {
		var (
			cid            int
			name, typeName string
			notnull, pk    int
			dfltValue      *string
		)
		require.NoError(t, rows.Scan(&cid, &name, &typeName, &notnull, &dfltValue, &pk))
		types[name] = strings.ToUpper(typeName)
	}
	require.NoError(t, rows.Err())
	require.Equal(t, "TEXT", types["b"])
}

func TestToSQL_TimeAndBytesInference(t *testing.T) {
	db := newTestSQLite(t)

	now := time.Now().UTC().Truncate(time.Second)
	// insyra.NewDataList flattens slice arguments, so use Append to keep the []byte
	// as a single cell rather than three uint8s.
	blobCol := insyra.NewDataList().SetName("blob")
	blobCol.Append([]byte{0x01, 0x02, 0x03})
	dt := insyra.NewDataTable(
		insyra.NewDataList(now).SetName("ts"),
		blobCol,
	)
	require.NoError(t, Write(context.Background(), dt, db, SQLite, "typed"))

	rows, err := db.Query("PRAGMA table_info(typed)")
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()
	types := map[string]string{}
	for rows.Next() {
		var (
			cid            int
			name, typeName string
			notnull, pk    int
			dfltValue      *string
		)
		require.NoError(t, rows.Scan(&cid, &name, &typeName, &notnull, &dfltValue, &pk))
		types[name] = strings.ToUpper(typeName)
	}
	require.NoError(t, rows.Err())
	require.Equal(t, "DATETIME", types["ts"])
	require.Equal(t, "BLOB", types["blob"])
}

func TestToSQLContext_RespectsCancelledContext(t *testing.T) {
	db := newTestSQLite(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-cancelled

	dt := insyra.NewDataTable(insyra.NewDataList(1, 2, 3).SetName("n"))
	err := Write(ctx, dt, db, SQLite, "x")
	require.Error(t, err)
}

func TestToSQL_RowNamesPersistsRowNameColumn(t *testing.T) {
	db := newTestSQLite(t)

	dt := insyra.NewDataTable(insyra.NewDataList(10, 20).SetName("v"))
	dt.SetRowNameByIndex(0, "alpha")
	dt.SetRowNameByIndex(1, "beta")

	require.NoError(t, Write(context.Background(), dt, db, SQLite, "with_rn", WriteOptions{HasRowNames: true}))

	type row struct {
		Name  string
		Value int
	}
	var got []row
	rows, err := db.Query("SELECT row_name, v FROM with_rn ORDER BY v")
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var r row
		require.NoError(t, rows.Scan(&r.Name, &r.Value))
		got = append(got, r)
	}
	require.NoError(t, rows.Err())
	require.Equal(t, []row{{"alpha", 10}, {"beta", 20}}, got)
}

// SEC-4: every other statement quotes its identifiers; the sqlite column
// lookup interpolated the table name straight into PRAGMA, so a perfectly
// ordinary table name with a space could not be appended to.
func TestAppendToTableWithSpaceInName(t *testing.T) {
	restoreConfig(t)
	insyra.Config.SetLogLevel(insyra.LogLevelFatal)

	db := newTestSQLite(t)

	const table = "my table"
	dt := insyra.NewDataTable(insyra.NewDataList(1, 2).SetName("value"))
	if err := Write(context.Background(), dt, db, SQLite, table, WriteOptions{IfTableExists: TableExistsFail}); err != nil {
		t.Fatalf("creating %q failed: %v", table, err)
	}
	more := insyra.NewDataTable(insyra.NewDataList(3).SetName("value"))
	if err := Write(context.Background(), more, db, SQLite, table, WriteOptions{IfTableExists: TableExistsAppend}); err != nil {
		t.Fatalf("appending to %q failed: %v", table, err)
	}

	read, err := Read(context.Background(), db, SQLite, table)
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
	insyra.Config.SetLogLevel(insyra.LogLevelFatal)

	db := newTestSQLite(t)

	const table = `we"ird`
	dt := insyra.NewDataTable(insyra.NewDataList(1).SetName("value"))
	if err := Write(context.Background(), dt, db, SQLite, table, WriteOptions{IfTableExists: TableExistsFail}); err != nil {
		t.Fatalf("creating %q failed: %v", table, err)
	}
	more := insyra.NewDataTable(insyra.NewDataList(2).SetName("value"))
	if err := Write(context.Background(), more, db, SQLite, table, WriteOptions{IfTableExists: TableExistsAppend}); err != nil {
		if strings.Contains(err.Error(), "syntax") {
			t.Fatalf("identifier was not quoted: %v", err)
		}
		t.Fatalf("appending to %q failed: %v", table, err)
	}
}

// restoreConfig puts back the settings a test changed.
func restoreConfig(t *testing.T) {
	t.Helper()
	level := insyra.Config.GetLogLevel()
	panicOnError := insyra.Config.GetPanicOnError()
	colored := insyra.Config.GetUseColoredOutput()
	hook := insyra.Config.GetDefaultErrHandlingFunc()
	t.Cleanup(func() {
		insyra.Config.SetLogLevel(level)
		insyra.Config.SetPanicOnError(panicOnError)
		insyra.Config.SetUseColoredOutput(colored)
		insyra.Config.SetDefaultErrHandlingFunc(hook)
	})
}
