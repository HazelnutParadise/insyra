package insyra

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestReadSQL_BasicSelect(t *testing.T) {
	db := newTestSQLite(t)
	require.NoError(t, db.Exec("CREATE TABLE t (a INTEGER, b TEXT);").Error)
	require.NoError(t, db.Exec("INSERT INTO t (a, b) VALUES (1, 'one'), (2, 'two');").Error)

	dt, err := ReadSQL(db, "t")
	require.NoError(t, err)
	require.NotNil(t, dt)

	rows, cols := dt.Size()
	require.Equal(t, 2, rows)
	require.Equal(t, 2, cols)
}

func TestReadSQL_ColumnsSubset(t *testing.T) {
	db := newTestSQLite(t)
	require.NoError(t, db.Exec("CREATE TABLE t (a INTEGER, b TEXT, c REAL);").Error)
	require.NoError(t, db.Exec("INSERT INTO t (a, b, c) VALUES (1, 'one', 1.5);").Error)

	dt, err := ReadSQL(db, "t", ReadSQLOptions{Columns: []string{"a", "c"}})
	require.NoError(t, err)
	_, cols := dt.Size()
	require.Equal(t, 2, cols)
}

func TestReadSQL_ParamsBoundToCustomQuery(t *testing.T) {
	db := newTestSQLite(t)
	require.NoError(t, db.Exec("CREATE TABLE t (n INTEGER);").Error)
	for _, v := range []int{1, 2, 3, 4, 5} {
		require.NoError(t, db.Exec("INSERT INTO t (n) VALUES (?);", v).Error)
	}

	dt, err := ReadSQL(db, "", ReadSQLOptions{
		Query:  "SELECT n FROM t WHERE n > ? AND n < ? ORDER BY n",
		Params: []any{1, 5},
	})
	require.NoError(t, err)
	rows, _ := dt.Size()
	require.Equal(t, 3, rows) // 2, 3, 4
}

func TestReadSQL_IndexColAliasesRowName(t *testing.T) {
	db := newTestSQLite(t)
	require.NoError(t, db.Exec("CREATE TABLE t (label TEXT, n INTEGER);").Error)
	require.NoError(t, db.Exec("INSERT INTO t (label, n) VALUES ('alpha', 1), ('beta', 2);").Error)

	dt, err := ReadSQL(db, "t", ReadSQLOptions{IndexCol: "label"})
	require.NoError(t, err)

	// IndexCol "label" becomes the row name; only "n" should remain as a column.
	_, cols := dt.Size()
	require.Equal(t, 1, cols)

	name0, ok := dt.GetRowNameByIndex(0)
	require.True(t, ok)
	require.Equal(t, "alpha", name0)
	name1, ok := dt.GetRowNameByIndex(1)
	require.True(t, ok)
	require.Equal(t, "beta", name1)
}

func TestReadSQL_ParseDatesParsesIsoString(t *testing.T) {
	db := newTestSQLite(t)
	require.NoError(t, db.Exec("CREATE TABLE t (when_at TEXT);").Error)
	require.NoError(t, db.Exec("INSERT INTO t (when_at) VALUES ('2024-06-01T12:34:56Z');").Error)

	dt, err := ReadSQL(db, "t", ReadSQLOptions{ParseDates: []string{"when_at"}})
	require.NoError(t, err)

	dl := dt.GetColByName("when_at")
	require.NotNil(t, dl)

	v := dl.Get(0)
	parsed, ok := v.(time.Time)
	require.Truef(t, ok, "expected time.Time, got %T", v)
	require.Equal(t, 2024, parsed.Year())
	require.Equal(t, time.June, parsed.Month())
}

func TestReadSQL_DTypeForcesType(t *testing.T) {
	db := newTestSQLite(t)
	require.NoError(t, db.Exec("CREATE TABLE t (n TEXT);").Error)
	require.NoError(t, db.Exec("INSERT INTO t (n) VALUES ('42'), ('100');").Error)

	dt, err := ReadSQL(db, "t", ReadSQLOptions{
		DType: map[string]reflect.Type{"n": reflect.TypeFor[int64]()},
	})
	require.NoError(t, err)

	dl := dt.GetColByName("n")
	require.NotNil(t, dl)
	require.Equal(t, int64(42), dl.Get(0))
	require.Equal(t, int64(100), dl.Get(1))
}

func TestReadSQLContext_RespectsCancellation(t *testing.T) {
	db := newTestSQLite(t)
	require.NoError(t, db.Exec("CREATE TABLE t (n INTEGER);").Error)
	require.NoError(t, db.Exec("INSERT INTO t (n) VALUES (1);").Error)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := ReadSQLContext(ctx, db, "t")
	require.Error(t, err)
}

func TestReadSQLStream_EmitsChunks(t *testing.T) {
	db := newTestSQLite(t)
	require.NoError(t, db.Exec("CREATE TABLE big (n INTEGER);").Error)
	for i := 1; i <= 25; i++ {
		require.NoError(t, db.Exec("INSERT INTO big (n) VALUES (?);", i).Error)
	}

	totalRows := 0
	chunkCount := 0
	for table, err := range ReadSQLStream(context.Background(), db, "big", ReadSQLOptions{ChunkSize: 10}) {
		require.NoError(t, err)
		require.NotNil(t, table)
		r, _ := table.Size()
		totalRows += r
		chunkCount++
	}
	require.Equal(t, 25, totalRows)
	require.Equal(t, 3, chunkCount, "25 rows at chunk size 10 ⇒ chunks of 10 + 10 + 5")
}

func TestReadSQLStream_StopsOnCancelledContext(t *testing.T) {
	db := newTestSQLite(t)
	require.NoError(t, db.Exec("CREATE TABLE big (n INTEGER);").Error)
	for i := 1; i <= 50; i++ {
		require.NoError(t, db.Exec("INSERT INTO big (n) VALUES (?);", i).Error)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Cancel after the first chunk; the next value is the context's error,
	// and the loop ends there.
	chunks := 0
	var last error
	for _, err := range ReadSQLStream(ctx, db, "big", ReadSQLOptions{ChunkSize: 5}) {
		if err != nil {
			last = err
			continue
		}
		chunks++
		cancel()
	}
	require.Equal(t, 1, chunks)
	require.ErrorIs(t, last, context.Canceled)
}

func TestReadSQLStream_BreakingEarlyReleasesTheConnection(t *testing.T) {
	db := newTestSQLite(t)
	require.NoError(t, db.Exec("CREATE TABLE big (n INTEGER);").Error)
	for i := 1; i <= 50; i++ {
		require.NoError(t, db.Exec("INSERT INTO big (n) VALUES (?);", i).Error)
	}
	sqlDB, err := db.DB()
	require.NoError(t, err)

	// No cancel anywhere: leaving the loop must be enough.
	for _, err := range ReadSQLStream(context.Background(), db, "big", ReadSQLOptions{ChunkSize: 5}) {
		require.NoError(t, err)
		break
	}
	require.Zero(t, sqlDB.Stats().InUse, "the query's connection is still checked out after the loop ended")
}

func TestReadSQLStream_NilDBIsOneFailure(t *testing.T) {
	yields := 0
	for table, err := range ReadSQLStream(context.Background(), nil, "t") {
		yields++
		require.Nil(t, table)
		require.Error(t, err)
	}
	require.Equal(t, 1, yields)
}
