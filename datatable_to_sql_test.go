package insyra

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	_ "modernc.org/sqlite"
)

// newTestSQLite returns a GORM-wrapped pure-Go SQLite in-memory database for
// tests. The underlying *sql.DB is closed via t.Cleanup.
func newTestSQLite(t *testing.T) *gorm.DB {
	t.Helper()
	sqlDB, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })
	db, err := gorm.Open(sqlite.New(sqlite.Config{Conn: sqlDB}), &gorm.Config{})
	require.NoError(t, err)
	return db
}

func TestToSQL_TransactionRollbackOnInsertError(t *testing.T) {
	db := newTestSQLite(t)

	// 建立表格，並加上 CHECK constraint 使插入會失敗
	require.NoError(t, db.Exec("CREATE TABLE test_tx (id INTEGER PRIMARY KEY AUTOINCREMENT, value INTEGER CHECK(value > 0));").Error)

	// 建立 DataTable，帶入違反約束的值
	valueCol := NewDataList(-1).SetName("value")
	dt := NewDataTable(valueCol)

	// 使用 AppendIfExists，因為表格已存在
	err := dt.ToSQL(db, "test_tx", ToSQLOptions{IfExists: SQLActionIfTableExistsAppend})
	require.Error(t, err, "expected ToSQL to return an error due to CHECK constraint violation")

	var cnt int64
	require.NoError(t, db.Raw("SELECT COUNT(*) FROM test_tx;").Scan(&cnt).Error)
	require.Equal(t, int64(0), cnt, "expected zero rows after rollback")
}

func TestToSQL_BatchInsertAcrossBatches(t *testing.T) {
	db := newTestSQLite(t)

	const total = 1500
	values := make([]any, total)
	for i := range total {
		values[i] = i + 1
	}
	dt := NewDataTable(NewDataList(values...).SetName("n"))

	// BatchSize 500 ⇒ 3 batched INSERTs.
	err := dt.ToSQL(db, "nums", ToSQLOptions{BatchSize: 500})
	require.NoError(t, err)

	var cnt int64
	require.NoError(t, db.Raw("SELECT COUNT(*) FROM nums").Scan(&cnt).Error)
	require.Equal(t, int64(total), cnt)

	// Spot-check a value.
	var got int64
	require.NoError(t, db.Raw("SELECT n FROM nums WHERE n = ?", 1234).Scan(&got).Error)
	require.Equal(t, int64(1234), got)
}

func TestToSQL_AppendModeAddsMissingColumnOnce(t *testing.T) {
	db := newTestSQLite(t)

	// Pre-create table with only column "a".
	require.NoError(t, db.Exec("CREATE TABLE evolving (a INTEGER);").Error)

	dt := NewDataTable(
		NewDataList(1, 2, 3).SetName("a"),
		NewDataList("x", "y", "z").SetName("b"),
	)
	err := dt.ToSQL(db, "evolving", ToSQLOptions{IfExists: SQLActionIfTableExistsAppend})
	require.NoError(t, err)

	// Column "b" should have been added once and populated.
	var bs []string
	rows, err := db.Raw("SELECT b FROM evolving ORDER BY a").Rows()
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var v string
		require.NoError(t, rows.Scan(&v))
		bs = append(bs, v)
	}
	require.Equal(t, []string{"x", "y", "z"}, bs)
}

func TestToSQL_AllNilColumnFallsBackToText(t *testing.T) {
	db := newTestSQLite(t)

	dt := NewDataTable(
		NewDataList(1, 2).SetName("a"),
		NewDataList(nil, nil).SetName("b"),
	)
	require.NoError(t, dt.ToSQL(db, "with_nil"))

	// Verify the column "b" was created with TEXT-ish affinity by reading PRAGMA.
	rows, err := db.Raw("PRAGMA table_info(with_nil)").Rows()
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
	require.Equal(t, "TEXT", types["b"])
}

func TestToSQL_TimeAndBytesInference(t *testing.T) {
	db := newTestSQLite(t)

	now := time.Now().UTC().Truncate(time.Second)
	// NewDataList flattens slice arguments, so use Append to keep the []byte
	// as a single cell rather than three uint8s.
	blobCol := NewDataList().SetName("blob")
	blobCol.Append([]byte{0x01, 0x02, 0x03})
	dt := NewDataTable(
		NewDataList(now).SetName("ts"),
		blobCol,
	)
	require.NoError(t, dt.ToSQL(db, "typed"))

	rows, err := db.Raw("PRAGMA table_info(typed)").Rows()
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
	require.Equal(t, "DATETIME", types["ts"])
	require.Equal(t, "BLOB", types["blob"])
}

func TestToSQLContext_RespectsCancelledContext(t *testing.T) {
	db := newTestSQLite(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-cancelled

	dt := NewDataTable(NewDataList(1, 2, 3).SetName("n"))
	err := dt.ToSQLContext(ctx, db, "x")
	require.Error(t, err)
}

func TestToSQL_RowNamesPersistsRowNameColumn(t *testing.T) {
	db := newTestSQLite(t)

	dt := NewDataTable(NewDataList(10, 20).SetName("v"))
	dt.SetRowNameByIndex(0, "alpha")
	dt.SetRowNameByIndex(1, "beta")

	require.NoError(t, dt.ToSQL(db, "with_rn", ToSQLOptions{RowNames: true}))

	type row struct {
		Name  string
		Value int
	}
	var got []row
	rows, err := db.Raw("SELECT row_name, v FROM with_rn ORDER BY v").Rows()
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var r row
		require.NoError(t, rows.Scan(&r.Name, &r.Value))
		got = append(got, r)
	}
	require.Equal(t, []row{{"alpha", 10}, {"beta", 20}}, got)
}
