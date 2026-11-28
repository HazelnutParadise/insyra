package sqlio

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/stretchr/testify/require"
)

func newSeeded(t *testing.T) *sql.DB {
	t.Helper()
	db := newTestSQLite(t)
	execSQL(t, db, "CREATE TABLE t (n INTEGER, label TEXT, at TEXT, row_name TEXT)")
	for i := 1; i <= 5; i++ {
		execSQL(t, db, "INSERT INTO t VALUES (?, ?, ?, ?)", i, fmt.Sprintf("l%d", i), fmt.Sprintf("2026-01-0%d", i), fmt.Sprintf("r%d", i))
	}
	return db
}

func TestEveryFunctionRefusesTwoOptionsValues(t *testing.T) {
	db := newSeeded(t)
	ctx := context.Background()
	_, err := Read(ctx, db, SQLite, "t", ReadOptions{}, ReadOptions{})
	require.ErrorContains(t, err, "ReadOptions")
	_, err = Query(ctx, db, SQLite, "SELECT n FROM t", QueryOptions{}, QueryOptions{})
	require.ErrorContains(t, err, "QueryOptions")
	for _, err := range Stream(ctx, db, SQLite, "t", 2, ReadOptions{}, ReadOptions{}) {
		require.ErrorContains(t, err, "ReadOptions")
	}
	for _, err := range StreamQuery(ctx, db, SQLite, "SELECT n FROM t", 2, QueryOptions{}, QueryOptions{}) {
		require.ErrorContains(t, err, "QueryOptions")
	}
	dt := insyra.NewDataTable(insyra.NewDataList(1).SetName("a"))
	require.ErrorContains(t, Write(ctx, dt, db, SQLite, "out", WriteOptions{}, WriteOptions{}), "WriteOptions")
}

func TestReadAndQueryAreSeparate(t *testing.T) {
	db := newSeeded(t)
	ctx := context.Background()
	_, err := Read(ctx, db, SQLite, "")
	require.ErrorContains(t, err, "use Query")
	_, err = Query(ctx, db, SQLite, "  ")
	require.ErrorContains(t, err, "query is empty")
	for _, err := range Stream(ctx, db, SQLite, "", 2) {
		require.ErrorContains(t, err, "use StreamQuery")
	}
	// A table name is quoted, so text that would be SQL is not run as SQL.
	_, err = Read(ctx, db, SQLite, "t WHERE 1=1")
	require.ErrorContains(t, err, "does not exist")
}

func TestQueryTakesTheReadSettings(t *testing.T) {
	db := newSeeded(t)
	dt, err := Query(context.Background(), db, SQLite, "SELECT n, at, row_name FROM t WHERE n >= ? ORDER BY n", QueryOptions{
		Params:     []any{4},
		ParseDates: []string{"at"},
		DType:      map[string]reflect.Type{"n": reflect.TypeFor[float64]()},
	})
	require.NoError(t, err)
	require.Equal(t, 2, dt.NumRows())
	require.Equal(t, []string{"n", "at"}, dt.ColNames())
	require.Equal(t, 4.0, dt.GetColByNumber(0).Get(0))
	name, ok := dt.GetRowNameByIndex(1)
	require.True(t, ok)
	require.Equal(t, "r5", name)
	require.Equal(t, "2026-01-05", fmt.Sprint(dt.GetColByNumber(1).Get(1))[:10])
}

func TestQueryRowNameColumnCanBeNamed(t *testing.T) {
	db := newSeeded(t)
	dt, err := Query(context.Background(), db, SQLite, "SELECT n, label FROM t ORDER BY n", QueryOptions{RowNameColumn: "label"})
	require.NoError(t, err)
	require.Equal(t, []string{"n"}, dt.ColNames())
	name, _ := dt.GetRowNameByIndex(0)
	require.Equal(t, "l1", name)
}

func TestStreamQueryReadsInBatches(t *testing.T) {
	db := newSeeded(t)
	var sizes []int
	for dt, err := range StreamQuery(context.Background(), db, SQLite, "SELECT n FROM t WHERE n > ? ORDER BY n", 2, QueryOptions{Params: []any{1}}) {
		require.NoError(t, err)
		sizes = append(sizes, dt.NumRows())
	}
	require.Equal(t, []int{2, 2}, sizes)
}

func TestStreamRefusesABatchSizeBelowOne(t *testing.T) {
	db := newSeeded(t)
	for _, n := range []int{0, -1} {
		calls := 0
		for dt, err := range Stream(context.Background(), db, SQLite, "t", n) {
			calls++
			require.Nil(t, dt)
			require.ErrorContains(t, err, "batchSize must be positive")
		}
		require.Equal(t, 1, calls)
	}
}

func TestAnEmptyResultIsAnEmptyTable(t *testing.T) {
	db := newSeeded(t)
	dt, err := Query(context.Background(), db, SQLite, "SELECT n FROM t WHERE n > 99")
	require.NoError(t, err)
	require.NotNil(t, dt)
	require.Equal(t, 0, dt.NumRows())
}

func TestWriteRefusesANilTable(t *testing.T) {
	db := newTestSQLite(t)
	require.ErrorContains(t, Write(context.Background(), (*insyra.DataTable)(nil), db, SQLite, "t"), "nil")
	require.ErrorContains(t, Write(context.Background(), nil, db, SQLite, "t"), "nil")
}

func TestWriteTakesAnyIDataTable(t *testing.T) {
	db := newTestSQLite(t)
	type wrapped struct{ *insyra.DataTable }
	dt := wrapped{insyra.NewDataTable(insyra.NewDataList(1, 2).SetName("a"))}
	require.NoError(t, Write(context.Background(), dt, db, SQLite, "w"))
	got, err := Read(context.Background(), db, SQLite, "w")
	require.NoError(t, err)
	require.Equal(t, 2, got.NumRows())
}

func TestBlobColumnReadsAsBytesThatCanBeCounted(t *testing.T) {
	db := newTestSQLite(t)
	execSQL(t, db, `CREATE TABLE files (name TEXT, payload BLOB)`)
	execSQL(t, db, `INSERT INTO files VALUES ('a', X'00FF41'), ('b', X'00FF41'), ('c', X'00FF42')`)
	dt, err := Read(context.Background(), db, SQLite, "files")
	require.NoError(t, err)
	col := dt.GetColByNumber(1)
	want := []byte{0x00, 0xff, 0x41}
	require.Equal(t, want, col.Get(0))
	require.Equal(t, 2, col.Count(want))
}

func TestEnumNames(t *testing.T) {
	for p, want := range map[TableExistsPolicy]string{
		TableExistsFail:       "TableExistsFail",
		TableExistsReplace:    "TableExistsReplace",
		TableExistsAppend:     "TableExistsAppend",
		TableExistsPolicy(99): "TableExistsPolicy(99)",
	} {
		require.Equal(t, want, fmt.Sprint(p))
	}
	require.True(t, strings.HasPrefix(checkDialect("oracle").Error(), "unknown SQL dialect"))
}

func TestReadBindsParamsInTheWhereClause(t *testing.T) {
	db := newSeeded(t)
	dt, err := Read(context.Background(), db, SQLite, "t", ReadOptions{WhereClause: "n > ?", Params: []any{3}, OrderBy: "n"})
	require.NoError(t, err)
	require.Equal(t, 2, dt.NumRows())
}
