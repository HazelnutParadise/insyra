package sqlio

import (
	"context"
	"database/sql"
	"github.com/HazelnutParadise/insyra"
	"os"
	"strings"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"
)

// These tests run against a real MySQL or PostgreSQL server, and only when
// its DSN is set. CI sets both in the Test workflow's sql-servers job;
// locally, start the servers with Docker and set, for example:
//
//	INSYRA_TEST_MYSQL_DSN='root:pw@tcp(127.0.0.1:33061)/shop?parseTime=true'
//	INSYRA_TEST_POSTGRES_DSN='postgres://postgres:pw@127.0.0.1:54321/shop?sslmode=disable'
//
// SQLite runs every path in the other tests; what only a server can show is
// its own SQL: information_schema lookups, $n placeholders, MySQL's implicit
// commits and RENAME TABLE, and the drivers' type mapping.
type sqlServer struct {
	name, driver, env, schema, placeholder string
	dialect                                Dialect
}

var sqlServers = []sqlServer{
	{"mysql", "mysql", "INSYRA_TEST_MYSQL_DSN", "shop", "?", MySQL},
	{"postgres", "pgx", "INSYRA_TEST_POSTGRES_DSN", "public", "$1", Postgres},
}

func eachSQLServer(t *testing.T, run func(t *testing.T, s sqlServer, db *sql.DB)) {
	ran := false
	for _, s := range sqlServers {
		dsn := os.Getenv(s.env)
		if dsn == "" {
			continue
		}
		ran = true
		t.Run(s.name, func(t *testing.T) {
			db, err := sql.Open(s.driver, dsn)
			require.NoError(t, err)
			t.Cleanup(func() { _ = db.Close() })
			require.NoError(t, db.Ping())
			run(t, s, db)
		})
	}
	if !ran {
		t.Skip("no INSYRA_TEST_MYSQL_DSN or INSYRA_TEST_POSTGRES_DSN set")
	}
}

// serverTable gives each test a table name of its own and drops it afterwards.
func serverTable(t *testing.T, s sqlServer, db *sql.DB, names ...string) {
	t.Helper()
	drop := func() {
		for _, n := range names {
			_, _ = db.Exec("DROP TABLE IF EXISTS " + quoteSQLIdent(string(s.dialect), n))
		}
	}
	drop()
	t.Cleanup(drop)
}

func serverCount(t *testing.T, s sqlServer, db *sql.DB, table string) int {
	t.Helper()
	var n int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM "+quoteSQLIdent(string(s.dialect), table)).Scan(&n))
	return n
}

func TestSQLServerRoundTrip(t *testing.T) {
	eachSQLServer(t, func(t *testing.T, s sqlServer, db *sql.DB) {
		const table = "insyra it round"
		serverTable(t, s, db, table)
		dt := insyra.NewDataTable(
			insyra.NewDataList(1, 2, 3).SetName("id"),
			insyra.NewDataList(1.5, 2.5, 3.5).SetName("amount"),
			insyra.NewDataList("a", `b"q`, "c`d").SetName("label"),
		)
		require.NoError(t, Write(context.Background(), dt, db, s.dialect, table, WriteOptions{Schema: s.schema}))

		got, err := Read(context.Background(), db, s.dialect, table, ReadOptions{Schema: s.schema, OrderBy: "id"})
		require.NoError(t, err)
		require.Equal(t, 3, got.NumRows())
		require.EqualValues(t, 2, got.GetColByName("id").Get(1))
		require.Equal(t, 2.5, got.GetColByName("amount").Get(1))
		require.Equal(t, `b"q`, got.GetColByName("label").Get(1))

		// A custom query uses the driver's own placeholder.
		q, err := Query(context.Background(), db, s.dialect, "SELECT label FROM "+quoteSQLIdent(string(s.dialect), table)+" WHERE id = "+s.placeholder, QueryOptions{
			Params: []any{3},
		})
		require.NoError(t, err)
		require.Equal(t, 1, q.NumRows())
		require.Equal(t, "c`d", q.GetColByNumber(0).Get(0))

		chunks := 0
		for c, err := range Stream(context.Background(), db, s.dialect, table, 2) {
			require.NoError(t, err)
			require.NotNil(t, c)
			chunks++
		}
		require.Equal(t, 2, chunks)

		err = Write(context.Background(), dt, db, s.dialect, table)
		require.ErrorContains(t, err, "already exists")
	})
}

func TestSQLServerAppendAddsAColumn(t *testing.T) {
	eachSQLServer(t, func(t *testing.T, s sqlServer, db *sql.DB) {
		const table = "insyra_it_append"
		serverTable(t, s, db, table)
		require.NoError(t, Write(context.Background(), insyra.NewDataTable(insyra.NewDataList(1).SetName("a")), db, s.dialect, table))
		more := insyra.NewDataTable(insyra.NewDataList(2).SetName("a"), insyra.NewDataList("x").SetName("b"))
		require.NoError(t, Write(context.Background(), more, db, s.dialect, table, WriteOptions{IfTableExists: TableExistsAppend}))
		got, err := Read(context.Background(), db, s.dialect, table, ReadOptions{OrderBy: "a"})
		require.NoError(t, err)
		require.Equal(t, 2, got.NumRows())
		require.Nil(t, got.GetColByName("b").Get(0))
		require.Equal(t, "x", got.GetColByName("b").Get(1))
	})
}

func TestSQLServerReplace(t *testing.T) {
	eachSQLServer(t, func(t *testing.T, s sqlServer, db *sql.DB) {
		const table = "insyra_it_replace"
		serverTable(t, s, db, table)
		require.NoError(t, Write(context.Background(), insyra.NewDataTable(insyra.NewDataList(1, 2, 3).SetName("id")), db, s.dialect, table))
		require.NoError(t, Write(context.Background(), insyra.NewDataTable(insyra.NewDataList(7).SetName("id")), db, s.dialect, table,
			WriteOptions{IfTableExists: TableExistsReplace, Schema: s.schema}))
		require.Equal(t, 1, serverCount(t, s, db, table))
		requireNoLeftoverTables(t, s, db)
	})
}

// A write that fails on the server, here an INSERT with more placeholders
// than the protocol allows, leaves the old table and its rows as they were.
func TestSQLServerReplaceKeepsOldTableWhenWriteFails(t *testing.T) {
	eachSQLServer(t, func(t *testing.T, s sqlServer, db *sql.DB) {
		const table = "insyra_it_keep"
		serverTable(t, s, db, table)
		require.NoError(t, Write(context.Background(), insyra.NewDataTable(insyra.NewDataList(1, 2, 3).SetName("id")), db, s.dialect, table))

		n := 70000
		vals := make([]any, n)
		for i := range vals {
			vals[i] = i
		}
		big := insyra.NewDataTable(insyra.NewDataList(vals...).SetName("id"))
		// The first batch fits, the second passes 65,535 placeholders.
		err := Write(context.Background(), big, db, s.dialect, table, WriteOptions{IfTableExists: TableExistsReplace, BatchSize: 66000})
		require.Error(t, err)
		require.Equal(t, 3, serverCount(t, s, db, table))
		requireNoLeftoverTables(t, s, db)
	})
}

func TestSQLServerReplaceRefusesAReferencedTableOnMySQL(t *testing.T) {
	eachSQLServer(t, func(t *testing.T, s sqlServer, db *sql.DB) {
		if s.dialect != MySQL {
			t.Skip("the staged replace runs on MySQL only")
		}
		serverTable(t, s, db, "insyra_it_child", "insyra_it_parent")
		_, err := db.Exec("CREATE TABLE insyra_it_parent (id INT PRIMARY KEY)")
		require.NoError(t, err)
		_, err = db.Exec("INSERT INTO insyra_it_parent VALUES (1)")
		require.NoError(t, err)
		_, err = db.Exec("CREATE TABLE insyra_it_child (pid INT, FOREIGN KEY (pid) REFERENCES insyra_it_parent(id))")
		require.NoError(t, err)
		err = Write(context.Background(), insyra.NewDataTable(insyra.NewDataList(5).SetName("id")), db, s.dialect, "insyra_it_parent",
			WriteOptions{IfTableExists: TableExistsReplace})
		require.ErrorContains(t, err, "referenced by a foreign key from insyra_it_child")
		require.Equal(t, 1, serverCount(t, s, db, "insyra_it_parent"))
	})
}

func requireNoLeftoverTables(t *testing.T, s sqlServer, db *sql.DB) {
	t.Helper()
	q := "SELECT table_name FROM information_schema.tables WHERE table_schema = " + bindVars(string(s.dialect), "?")
	rows, err := db.Query(q, s.schema)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var name string
		require.NoError(t, rows.Scan(&name))
		require.False(t, strings.HasPrefix(name, "insyra_new_") || strings.HasPrefix(name, "insyra_old_"), "left behind %s", name)
	}
	require.NoError(t, rows.Err())
}

// More placeholders than the protocol allows: the first batch fits and the
// second fails on the server.
func tooManyPlaceholders() (*insyra.DataTable, WriteOptions) {
	vals := make([]any, 70000)
	for i := range vals {
		vals[i] = i
	}
	return insyra.NewDataTable(insyra.NewDataList(vals...).SetName("id"), insyra.NewDataList(vals...).SetName("extra")),
		WriteOptions{BatchSize: 33000}
}

// A write to a missing table that fails leaves no table behind, on MySQL as
// well, where CREATE TABLE commits on its own.
func TestSQLServerFailedCreateLeavesNoTable(t *testing.T) {
	eachSQLServer(t, func(t *testing.T, s sqlServer, db *sql.DB) {
		const table = "insyra_it_create"
		serverTable(t, s, db, table)
		big, opts := tooManyPlaceholders()
		require.Error(t, Write(context.Background(), big, db, s.dialect, table, opts))
		exists, err := tableExists(context.Background(), db, string(s.dialect), "", table)
		require.NoError(t, err)
		require.False(t, exists, "the failed write left the table behind")
		requireNoLeftoverTables(t, s, db)
	})
}

// An append that adds a column and then fails leaves the table as it was:
// no new column and no new rows, on MySQL as well, where ALTER TABLE
// commits on its own.
func TestSQLServerFailedAppendLeavesTheTableAsItWas(t *testing.T) {
	eachSQLServer(t, func(t *testing.T, s sqlServer, db *sql.DB) {
		const table = "insyra_it_append_fail"
		serverTable(t, s, db, table)
		require.NoError(t, Write(context.Background(), insyra.NewDataTable(insyra.NewDataList(1, 2).SetName("id")), db, s.dialect, table))
		big, opts := tooManyPlaceholders()
		opts.IfTableExists = TableExistsAppend
		require.Error(t, Write(context.Background(), big, db, s.dialect, table, opts))
		require.Equal(t, 2, serverCount(t, s, db, table))
		cols, err := fetchTableColumns(context.Background(), db, string(s.dialect), "", table)
		require.NoError(t, err)
		require.Equal(t, []string{"id"}, cols)
	})
}

func TestSQLServerWritesAnUnnamedColumn(t *testing.T) {
	eachSQLServer(t, func(t *testing.T, s sqlServer, db *sql.DB) {
		const table = "insyra_it_unnamed"
		serverTable(t, s, db, table)
		require.NoError(t, Write(context.Background(), insyra.NewDataTable(insyra.NewDataList(1, 2, 3)), db, s.dialect, table))
		got, err := Read(context.Background(), db, s.dialect, table)
		require.NoError(t, err)
		require.Equal(t, []string{"A"}, got.ColNames())
	})
}
